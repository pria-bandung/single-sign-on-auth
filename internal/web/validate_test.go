package web

import "testing"

// validateNext is a pure helper, so it is tested white-box (package web) with a
// table of inputs. Anything that is not a local, relative path must fall back to
// the default to prevent open redirects.
func TestValidateNext(t *testing.T) {
	const def = "/protected"
	cases := []struct {
		in   string
		want string
	}{
		{"/protected", "/protected"},
		{"/settings/profile", "/settings/profile"},
		{"", def},
		{"https://evil.com", def},
		{"http://evil.com/path", def},
		{"//evil.com", def},
		{`/\evil.com`, def},
		{"javascript:alert(1)", def},
		{"ftp://host", def},
	}
	for _, c := range cases {
		if got := validateNext(c.in); got != c.want {
			t.Errorf("validateNext(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
