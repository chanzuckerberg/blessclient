// Copyright 2018, Honeycomb, Hound Technology, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package honeycomb contains a trace exporter for Honeycomb
package honeycomb

import (
	"time"

	libhoney "github.com/honeycombio/libhoney-go"
	"go.opencensus.io/trace"
)

// Exporter is an implementation of trace.Exporter that uploads a span to Honeycomb
type Exporter struct {
	Builder        *libhoney.Builder
	SampleFraction float64
	// Service Name identifies your application. While optional, setting this
	// field is extremely valuable when you instrument multiple services. If set
	// it will be added to all events as `service_name`
	ServiceName string
}

// Annotation represents an annotation with a value and a timestamp.
type Annotation struct {
	Timestamp time.Time `json:"timestamp"`
	Value     string    `json:"value"`
}

// Span is the format of trace events that Honeycomb accepts
type Span struct {
	TraceID     string       `json:"trace.trace_id"`
	Name        string       `json:"name"`
	ID          string       `json:"trace.span_id"`
	ParentID    string       `json:"trace.parent_id,omitempty"`
	DurationMs  float64      `json:"duration_ms"`
	Timestamp   time.Time    `json:"timestamp,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Close waits for all in-flight messages to be sent. You should
// call Close() before app termination.
func (e *Exporter) Close() {
	libhoney.Close()
}

// NewExporter returns an implementation of trace.Exporter that uploads spans to Honeycomb
//
// writeKey is your Honeycomb writeKey (also known as your API key)
// dataset is the name of your Honeycomb dataset to send trace events to
//
// Don't have a Honeycomb account? Sign up at https://ui.honeycomb.io/signup
func NewExporter(writeKey, dataset string) *Exporter {
	// Developer note: bump this with each release
	versionStr := "1.0.1"
	libhoney.UserAgentAddition = "Honeycomb-OpenCensus-exporter/" + versionStr

	libhoney.Init(libhoney.Config{
		WriteKey: writeKey,
		Dataset:  dataset,
	})
	builder := libhoney.NewBuilder()
	// default sample reate is 1: aka no sampling.
	// set sampleRate on the exporter to be the sample rate given to the
	// ProbabilitySampler if used.
	return &Exporter{
		Builder:        builder,
		SampleFraction: 1,
		ServiceName:    "",
	}
}

// ExportSpan exports a span to Honeycomb
func (e *Exporter) ExportSpan(sd *trace.SpanData) {
	ev := e.Builder.NewEvent()
	if e.SampleFraction != 0 {
		ev.SampleRate = uint(1 / e.SampleFraction)
	}
	if e.ServiceName != "" {
		ev.AddField("service_name", e.ServiceName)
	}
	ev.Timestamp = sd.StartTime
	hs := honeycombSpan(sd)
	ev.Add(hs)

	// Add an event field for each attribute
	if len(sd.Attributes) != 0 {
		for key, value := range sd.Attributes {
			ev.AddField(key, value)
		}
	}

	// Add an event field for status code and status message
	if sd.Status.Code != 0 {
		ev.AddField("status_code", sd.Status.Code)
	}
	if sd.Status.Message != "" {
		ev.AddField("status_description", sd.Status.Message)
	}
	ev.SendPresampled()
}

func honeycombSpan(s *trace.SpanData) Span {
	sc := s.SpanContext
	hcSpan := Span{
		TraceID:   sc.TraceID.String(),
		ID:        sc.SpanID.String(),
		Name:      s.Name,
		Timestamp: s.StartTime,
	}

	if s.ParentSpanID != (trace.SpanID{}) {
		hcSpan.ParentID = s.ParentSpanID.String()
	}

	if s, e := s.StartTime, s.EndTime; !s.IsZero() && !e.IsZero() {
		hcSpan.DurationMs = float64(e.Sub(s)) / float64(time.Millisecond)
	}

	if len(s.Annotations) != 0 || len(s.MessageEvents) != 0 {
		hcSpan.Annotations = make([]Annotation, 0, len(s.Annotations)+len(s.MessageEvents))
		for _, a := range s.Annotations {
			hcSpan.Annotations = append(hcSpan.Annotations, Annotation{
				Timestamp: a.Time,
				Value:     a.Message,
			})
		}
		for _, m := range s.MessageEvents {
			a := Annotation{
				Timestamp: m.Time,
			}
			switch m.EventType {
			case trace.MessageEventTypeSent:
				a.Value = "SENT"
			case trace.MessageEventTypeRecv:
				a.Value = "RECV"
			default:
				a.Value = "<?>"
			}
			hcSpan.Annotations = append(hcSpan.Annotations, a)
		}
	}
	return hcSpan
}
