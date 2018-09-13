package errs

// Error is an alias to a string
type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	// ErrMissingConfig config not provided
	ErrMissingConfig = Error("missing config")
)
