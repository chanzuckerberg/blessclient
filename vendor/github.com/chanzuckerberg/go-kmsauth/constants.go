package kmsauth

import (
	"time"
)

const (
	//TimeFormat the time format for kmsauth tokens
	TimeFormat = time.RFC3339
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
