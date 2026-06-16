package auth

import "testing"

// parseGoogleIdentity is the pure JSON-decoding half of the Google identity
// fetch, so it is tested white-box with sample userinfo payloads. The HTTP/token
// plumbing around it is verified manually against real Google.
func TestParseGoogleIdentity(t *testing.T) {
	body := []byte(`{"sub":"123","email":"ann@example.com","email_verified":true,"name":"Ann"}`)

	id, err := parseGoogleIdentity(body)
	if err != nil {
		t.Fatalf("parseGoogleIdentity: %v", err)
	}
	if id.Sub != "123" {
		t.Errorf("Sub = %q, want %q", id.Sub, "123")
	}
	if id.Email != "ann@example.com" {
		t.Errorf("Email = %q, want %q", id.Email, "ann@example.com")
	}
	if !id.EmailVerified {
		t.Error("EmailVerified = false, want true")
	}
	if id.Name != "Ann" {
		t.Errorf("Name = %q, want %q", id.Name, "Ann")
	}
}

func TestParseGoogleIdentityUnverifiedEmail(t *testing.T) {
	body := []byte(`{"sub":"1","email":"x@example.com","email_verified":false}`)
	id, err := parseGoogleIdentity(body)
	if err != nil {
		t.Fatalf("parseGoogleIdentity: %v", err)
	}
	if id.EmailVerified {
		t.Error("EmailVerified = true, want false")
	}
}

func TestParseGoogleIdentityRejectsMalformedJSON(t *testing.T) {
	if _, err := parseGoogleIdentity([]byte(`{not json`)); err == nil {
		t.Error("expected an error for malformed JSON")
	}
}
