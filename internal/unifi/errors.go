package unifi

// HttpAuthError is returned when the controller responds 401 (an invalid or
// expired session). Callers use errors.As(err, &authErr) to drive transparent
// re-authentication. The exported name is kept for source compatibility with the
// upstream client this package was ported from.
type HttpAuthError struct {
	msg string
}

func (e *HttpAuthError) Error() string { return e.msg }
