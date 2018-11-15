package kmsauth

import (
	"time"
)

const (
	//TimeFormat the time format for kmsauth tokens
	// TimeFormat  = "%Y%m%dT%H%M%SZ"
	TimeFormat = "20060102T150405Z"
	// Mon Jan 2 15:04:05 MST 2006
	// timeSkew how much to compensate for time skew
	timeSkew = time.Duration(3) * time.Minute
)

// TokenVersion is a token version
type TokenVersion int

const (
	// TokenVersion1 is a token version
	TokenVersion1 = 1
	// TokenVersion2 is a token version
	TokenVersion2 = 2
)
