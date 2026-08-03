package unifi

// AuthError is returned when the Integration API responds 401 or 403 — a missing
// or invalid X-API-KEY. Unlike the legacy session cookie the key is static, so
// there is no re-auth to perform: a persistent AuthError means the configured key
// is wrong. Exported so tests (and callers wanting a clearer log) can inspect it
// via errors.As.
type AuthError struct {
	msg string
}

func (e *AuthError) Error() string { return e.msg }
