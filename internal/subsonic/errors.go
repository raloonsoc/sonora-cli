package subsonic

import "errors"

// Sentinel errors for Subsonic error codes that callers branch on. Codes not
// listed here are wrapped as opaque *ServerError values.
var (
	ErrUnauthorized = errors.New("subsonic: unauthorized")
	ErrNotFound     = errors.New("subsonic: not found")
)

// Subsonic error codes, from the OpenSubsonic API spec.
const (
	codeGeneric               = 0
	codeMissingParam          = 10
	codeIncompatibleClient    = 20
	codeIncompatibleServer    = 30
	codeBadCredentials        = 40
	codeTokenAuthNotSupported = 41
	codeUnauthorized          = 50
	codeTrialExpired          = 60
	codeNotFound              = 70
)

// ServerError wraps a Subsonic error response that has no dedicated
// sentinel.
type ServerError struct {
	Code    int
	Message string
}

func (e *ServerError) Error() string {
	return e.Message
}

// mapError translates a Subsonic error code into a Go error, using a
// sentinel for conditions callers branch on and *ServerError otherwise.
func mapError(code int, message string) error {
	switch code {
	case codeBadCredentials, codeUnauthorized, codeTokenAuthNotSupported:
		return ErrUnauthorized
	case codeNotFound:
		return ErrNotFound
	default:
		return &ServerError{Code: code, Message: message}
	}
}
