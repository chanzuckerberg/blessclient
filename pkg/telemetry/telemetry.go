package telemetry

// some common labels for telemetry
const (
	FieldBlessclientVersion = "blessclient.version"
	FieldBlessclientGitSha  = "blessclient.git_sha"
	FieldBlessclientRelease = "blessclient.release"
	FieldBlessclientDirty   = "blessclient.dirty"

	FieldID        = "id"
	FieldRegion    = "aws_region"
	FieldError     = "error"
	FieldUser      = "user"
	FieldFreshCert = "cert.is_fresh"
	FieldIsCached  = "is_cached"
)
