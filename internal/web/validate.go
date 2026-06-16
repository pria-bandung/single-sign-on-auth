package web

// defaultNext is where authenticated users land when no valid "next" is given.
const defaultNext = "/protected"

// validateNext returns next only if it is a safe, local, relative path; anything
// else (empty, absolute URL, scheme-relative "//host", or backslash trick)
// falls back to defaultNext. This prevents the login flow from being abused as
// an open redirect.
func validateNext(next string) string {
	if next == "" || next[0] != '/' {
		return defaultNext
	}
	// "//host" and "/\host" are treated by browsers as protocol-relative URLs to
	// an external host, so reject them.
	if len(next) > 1 && (next[1] == '/' || next[1] == '\\') {
		return defaultNext
	}
	return next
}
