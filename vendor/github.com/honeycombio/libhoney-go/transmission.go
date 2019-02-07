package libhoney

// txClient handles the transmission of events to Honeycomb.
//
// Overview
//
// Create a new instance of Client.
// Set any of the public fields for which you want to override the defaults.
// Call Start() to spin up the background goroutines necessary for transmission
// Call Add(Event) to queue an event for transmission
// Ensure Stop() is called to flush all in-flight messages.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/facebookgo/muster"
)

const (
	apiMaxBatchSize    int = 5000000 // 5MB
	apiEventSizeMax    int = 100000  // 100KB
	maxOverflowBatches int = 10
)

// Output is responsible for handling events after Send() is called.
// Implementations of Add() must be safe for concurrent calls.
type Output interface {
	Add(ev *Event)
	Start() error
	Stop() error
}

type txDefaultClient struct {
	maxBatchSize         uint          // how many events to collect into a batch before sending
	batchTimeout         time.Duration // how often to send off batches
	maxConcurrentBatches uint          // how many batches can be inflight simultaneously
	pendingWorkCapacity  uint          // how many events to allow to pile up
	blockOnSend          bool          // whether to block or drop events when the queue fills
	blockOnResponses     bool          // whether to block or drop responses when the queue fills

	transport http.RoundTripper

	muster muster.Client
}

func (t *txDefaultClient) Start() error {
	logger.Printf("default transmission starting")
	t.muster.MaxBatchSize = t.maxBatchSize
	t.muster.BatchTimeout = t.batchTimeout
	t.muster.MaxConcurrentBatches = t.maxConcurrentBatches
	t.muster.PendingWorkCapacity = t.pendingWorkCapacity
	t.muster.BatchMaker = func() muster.Batch {
		return &batchAgg{
			batches: map[string][]*Event{},
			httpClient: &http.Client{
				Transport: t.transport,
				Timeout:   10 * time.Second,
			},
			blockOnResponses: t.blockOnResponses,
		}
	}
	return t.muster.Start()
}

func (t *txDefaultClient) Stop() error {
	logger.Printf("default transmission stopping")
	return t.muster.Stop()
}

func (t *txDefaultClient) Add(ev *Event) {
	logger.Printf("adding event to transmission; queue length %d", len(t.muster.Work))
	sd.Gauge("queue_length", len(t.muster.Work))
	if t.blockOnSend {
		t.muster.Work <- ev
		sd.Increment("messages_queued")
	} else {
		select {
		case t.muster.Work <- ev:
			sd.Increment("messages_queued")
		default:
			sd.Increment("queue_overflow")
			r := Response{
				Err:      errors.New("queue overflow"),
				Metadata: ev.Metadata,
			}
			writeToResponse(r, t.blockOnResponses)
		}
	}
}

// batchAgg is a batch aggregator - it's actually collecting what will
// eventually be one or more batches sent to the /1/batch/dataset endpoint.
type batchAgg struct {
	// map of batch key to a list of events destined for that batch
	batches map[string][]*Event
	// Used to reenque events when an initial batch is too large
	overflowBatches  map[string][]*Event
	httpClient       *http.Client
	blockOnResponses bool
	// numEncoded       int

	// allows manipulation of the value of "now" for testing
	testNower   nower
	testBlocker *sync.WaitGroup
}

// batch is a collection of events that will all be POSTed as one HTTP call
// type batch []*Event

func (b *batchAgg) Add(ev interface{}) {
	// from muster godoc: "The Batch does not need to be safe for concurrent
	// access; synchronization will be handled by the Client."
	if b.batches == nil {
		b.batches = map[string][]*Event{}
	}
	e := ev.(*Event)
	// collect separate buckets of events to send based on the trio of api/wk/ds
	// if all three of those match it's safe to send all the events in one batch
	key := fmt.Sprintf("%s_%s_%s", e.APIHost, e.WriteKey, e.Dataset)
	b.batches[key] = append(b.batches[key], e)
}

func (b *batchAgg) enqueueResponse(resp Response) {
	if writeToResponse(resp, b.blockOnResponses) {
		if b.testBlocker != nil {
			b.testBlocker.Done()
		}
	}
}

func (b *batchAgg) reenqueueEvents(events []*Event) {
	if b.overflowBatches == nil {
		b.overflowBatches = make(map[string][]*Event)
	}
	for _, e := range events {
		key := fmt.Sprintf("%s_%s_%s", e.APIHost, e.WriteKey, e.Dataset)
		b.overflowBatches[key] = append(b.overflowBatches[key], e)
	}
}

func (b *batchAgg) Fire(notifier muster.Notifier) {
	defer notifier.Done()

	// send each batchKey's collection of event as a POST to /1/batch/<dataset>
	// we don't need the batch key anymore; it's done its sorting job
	for _, events := range b.batches {
		b.fireBatch(events)
	}
	// The initial batches could have had payloads that were greater than 5MB.
	// The remaining events will have overflowed into overflowBatches
	// Process these until complete. Overflow batches can also overflow, so we
	// have to prepare to process it multiple times
	overflowCount := 0
	if b.overflowBatches != nil {
		for len(b.overflowBatches) > 0 {
			// We really shouldn't get here but defensively avoid an endless
			// loop of re-enqueued events
			if overflowCount > maxOverflowBatches {
				break
			}
			overflowCount++
			// fetch the keys in this map - we can't range over the map
			// because it's possible that fireBatch will reenqueue more overflow
			// events
			keys := make([]string, len(b.overflowBatches))
			i := 0
			for k := range b.overflowBatches {
				keys[i] = k
				i++
			}

			for _, k := range keys {
				events := b.overflowBatches[k]
				// fireBatch may append more overflow events
				// so we want to clear this key before firing the batch
				delete(b.overflowBatches, k)
				b.fireBatch(events)
			}
		}
	}
}

func (b *batchAgg) fireBatch(events []*Event) {
	start := time.Now().UTC()
	if b.testNower != nil {
		start = b.testNower.Now()
	}
	if len(events) == 0 {
		// we managed to create a batch key with no events. odd. move on.
		return
	}
	encEvs, numEncoded := b.encodeBatch(events)
	// if we failed to encode any events skip this batch
	if numEncoded == 0 {
		return
	}
	// get some attributes common to this entire batch up front
	apiHost := events[0].APIHost
	writeKey := events[0].WriteKey
	dataset := events[0].Dataset

	// sigh. dislike
	userAgent := fmt.Sprintf("libhoney-go/%s", version)
	if UserAgentAddition != "" {
		userAgent = fmt.Sprintf("%s %s", userAgent, strings.TrimSpace(UserAgentAddition))
	}

	// build the HTTP request
	reqBody, gzipped := buildReqReader(encEvs)
	url, err := url.Parse(apiHost)
	if err != nil {
		end := time.Now().UTC()
		if b.testNower != nil {
			end = b.testNower.Now()
		}
		dur := end.Sub(start)
		sd.Increment("send_errors")
		for _, ev := range events {
			// Pass the parsing error down responses channel for each event that
			// didn't already error during encoding
			if ev != nil {
				b.enqueueResponse(Response{
					Duration: dur / time.Duration(numEncoded),
					Metadata: ev.Metadata,
					Err:      err,
				})
			}
		}
		return
	}
	url.Path = path.Join(url.Path, "/1/batch", dataset)
	req, err := http.NewRequest("POST", url.String(), reqBody)
	req.Header.Set("Content-Type", "application/json")
	if gzipped {
		req.Header.Set("Content-Encoding", "gzip")
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Add("X-Honeycomb-Team", writeKey)
	// send off batch!
	resp, err := b.httpClient.Do(req)
	end := time.Now().UTC()
	if b.testNower != nil {
		end = b.testNower.Now()
	}
	dur := end.Sub(start)

	// if the entire HTTP POST failed, send a failed response for every event
	if err != nil {
		sd.Increment("send_errors")
		// Pass the top-level send error down responses channel for each event
		// that didn't already error during encoding
		b.enqueueErrResponses(err, events, dur/time.Duration(numEncoded))
		// the POST failed so we're done with this batch key's worth of events
		return
	}

	// ok, the POST succeeded, let's process each individual response
	sd.Increment("batches_sent")
	sd.Count("messages_sent", numEncoded)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sd.Increment("send_errors")
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			b.enqueueErrResponses(fmt.Errorf("Got HTTP error code but couldn't read response body: %v", err),
				events, dur/time.Duration(numEncoded))
			return
		}
		for _, ev := range events {
			if ev != nil {
				b.enqueueResponse(Response{
					StatusCode: resp.StatusCode,
					Body:       body,
					Duration:   dur / time.Duration(numEncoded),
					Metadata:   ev.Metadata,
				})
			}
		}
		return
	}

	// decode the responses
	batchResponses := []Response{}
	err = json.NewDecoder(resp.Body).Decode(&batchResponses)
	if err != nil {
		// if we can't decode the responses, just error out all of them
		sd.Increment("response_decode_errors")
		b.enqueueErrResponses(err, events, dur/time.Duration(numEncoded))
		return
	}

	// Go through the responses and send them down the queue. If an Event
	// triggered a JSON error, it wasn't sent to the server and won't have a
	// returned response... so we have to be a bit more careful matching up
	// responses with Events.
	var eIdx int
	for _, resp := range batchResponses {
		resp.Duration = dur / time.Duration(numEncoded)
		for events[eIdx] == nil {
			eIdx++
		}
		if eIdx == len(events) { // just in case
			break
		}
		resp.Metadata = events[eIdx].Metadata
		b.enqueueResponse(resp)
		eIdx++
	}
}

// create the JSON for this event list manually so that we can send
// responses down the response queue for any that fail to marshal
func (b *batchAgg) encodeBatch(events []*Event) ([]byte, int) {
	// track first vs. rest events for commas
	first := true
	// track how many we successfully encode for later bookkeeping
	var numEncoded int
	buf := bytes.Buffer{}
	buf.WriteByte('[')
	bytesTotal := 1
	// ok, we've got our array, let's populate it with JSON events
	for i, ev := range events {
		if !first {
			buf.WriteByte(',')
			bytesTotal++
		}
		first = false
		evByt, err := json.Marshal(ev)
		if err != nil {
			b.enqueueResponse(Response{
				Err:      err,
				Metadata: ev.Metadata,
			})
			// nil out the invalid Event so we can line up sent Events with server
			// responses if needed. don't delete to preserve slice length.
			events[i] = nil
			continue
		}
		// if the event is too large to ever send, add an error to the queue
		if len(evByt) > apiEventSizeMax {
			b.enqueueResponse(Response{
				Err:      fmt.Errorf("event exceeds max event size of %d bytes, API will not accept this event", apiEventSizeMax),
				Metadata: ev.Metadata,
			})
			events[i] = nil
			continue
		}
		bytesTotal += len(evByt)

		// count for the trailing ]
		if bytesTotal+1 > apiMaxBatchSize {
			b.reenqueueEvents(events[i:])
			break
		}
		buf.Write(evByt)
		numEncoded++
	}
	buf.WriteByte(']')
	return buf.Bytes(), numEncoded
}

func (b *batchAgg) enqueueErrResponses(err error, events []*Event, duration time.Duration) {
	for _, ev := range events {
		if ev != nil {
			b.enqueueResponse(Response{
				Err:      err,
				Duration: duration,
				Metadata: ev.Metadata,
			})
		}
	}
}

// buildReqReader returns an io.Reader and a boolean, indicating whether or not
// the io.Reader is gzip-compressed.
func buildReqReader(jsonEncoded []byte) (io.Reader, bool) {
	buf := bytes.Buffer{}
	g := gzip.NewWriter(&buf)
	if _, err := g.Write(jsonEncoded); err == nil {
		if err = g.Close(); err == nil { // flush
			return &buf, true
		}
	}
	return bytes.NewReader(jsonEncoded), false
}

// nower to make testing easier
type nower interface {
	Now() time.Time
}

// WriterOutput implements the Output interface by marshalling events to JSON
// and writing to STDOUT, or to the writer W if one is specified.
type WriterOutput struct {
	W io.Writer

	sync.Mutex
}

func (w *WriterOutput) Start() error { return nil }
func (w *WriterOutput) Stop() error  { return nil }

func (w *WriterOutput) Add(ev *Event) {
	var m []byte
	func() {
		ev.lock.RLock()
		defer ev.lock.RUnlock()

		tPointer := &(ev.Timestamp)
		if ev.Timestamp.IsZero() {
			tPointer = nil
		}

		// don't include sample rate if it's 1; this is the default
		sampleRate := ev.SampleRate
		if sampleRate == 1 {
			sampleRate = 0
		}

		m, _ = json.Marshal(struct {
			Data       marshallableMap `json:"data"`
			SampleRate uint            `json:"samplerate,omitempty"`
			Timestamp  *time.Time      `json:"time,omitempty"`
			Dataset    string          `json:"dataset,omitempty"`
		}{ev.data, sampleRate, tPointer, ev.Dataset})
		m = append(m, '\n')
	}()

	w.Lock()
	defer w.Unlock()
	if w.W == nil {
		w.W = os.Stdout
	}
	w.W.Write(m)
}

// DiscardOutput implements the Output interface and drops all events.
type DiscardOutput struct {
	*WriterOutput
}

func (d *DiscardOutput) Add(ev *Event) {}

// MockOutput implements the Output interface by retaining a slice of added
// events, for use in unit tests.
type MockOutput struct {
	events []*Event
	sync.Mutex
}

func (m *MockOutput) Add(ev *Event) {
	m.Lock()
	m.events = append(m.events, ev)
	m.Unlock()
}

func (m *MockOutput) Start() error { return nil }
func (m *MockOutput) Stop() error  { return nil }

func (m *MockOutput) Events() []*Event {
	m.Lock()
	defer m.Unlock()
	output := make([]*Event, len(m.events))
	copy(output, m.events)
	return output
}
