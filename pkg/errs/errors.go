package errs

// Error is an alias to a string
type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	// ErrMissingConfig config not provided
	ErrMissingConfig = Error("missing config")
	// ErrLoginRequired login required
	ErrLoginRequired = Error("login required")
	// ErrSSHKeyNotFound ssh key not found
	ErrSSHKeyNotFound = Error("ssh key not found")
	// ErrNoCertificateInResponse no certificarte in response
	ErrNoCertificateInResponse = Error("no certificate in response")
	// ErrMissingConfigURL missing config url
	ErrMissingConfigURL = Error("missing config url")
)
