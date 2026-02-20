package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

// --- splitToken ---

func TestSplitToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"valid two parts", "abc.def", 2},
		{"no dot", "abcdef", 0},
		{"empty string", "", 0},
		{"dot only", ".", 2},
		{"multiple dots splits on last", "a.b.c", 2},
		{"trailing dot", "abc.", 2},
		{"leading dot", ".abc", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitToken(tt.token)
			if len(got) != tt.want {
				t.Fatalf("splitToken(%q) returned %d parts, want %d", tt.token, len(got), tt.want)
			}
		})
	}
}

func TestSplitTokenMultipleDots(t *testing.T) {
	parts := splitToken("a.b.c")
	if parts[0] != "a.b" {
		t.Fatalf("splitToken first part = %q, want %q", parts[0], "a.b")
	}
	if parts[1] != "c" {
		t.Fatalf("splitToken second part = %q, want %q", parts[1], "c")
	}
}

// --- encodeHMAC / decodeHMAC round-trip ---

func TestEncodeDecodeRoundTrip(t *testing.T) {
	secret := []byte("test-secret-key")
	payloads := []string{
		`{"uid":"1","usr":"alice"}`,
		``,
		`{"nested":{"key":"value"},"arr":[1,2,3]}`,
	}
	for _, p := range payloads {
		token := encodeHMAC(secret, []byte(p))
		got, err := decodeHMAC(secret, token)
		if err != nil {
			t.Fatalf("decodeHMAC(%q) error: %v", p, err)
		}
		if string(got) != p {
			t.Fatalf("decodeHMAC round-trip: got %q, want %q", got, p)
		}
	}
}

func TestDecodeWrongSecret(t *testing.T) {
	token := encodeHMAC([]byte("secret-a"), []byte("payload"))
	_, err := decodeHMAC([]byte("secret-b"), token)
	if err == nil || err.Error() != "invalid signature" {
		t.Fatalf("expected 'invalid signature', got %v", err)
	}
}

// --- malformed tokens ---

func TestDecodeMalformedTokens(t *testing.T) {
	secret := []byte("s")
	tests := []struct {
		name    string
		token   string
		wantErr string
	}{
		{"no dot", "nodot", "malformed token"},
		{"empty string", "", "malformed token"},
		{"bad base64 payload", "!!!.validbase64", "malformed payload"},
		{"bad base64 signature", base64.RawURLEncoding.EncodeToString([]byte("ok")) + ".!!!", "malformed signature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeHMAC(secret, tt.token)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// --- signature tampering ---

func TestDecodeTamperedSignature(t *testing.T) {
	secret := []byte("key")
	token := encodeHMAC(secret, []byte("data"))
	parts := strings.SplitN(token, ".", 2)
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte("fake"))
	_, err := decodeHMAC(secret, tampered)
	if err == nil || err.Error() != "invalid signature" {
		t.Fatalf("expected 'invalid signature', got %v", err)
	}
}

func TestDecodeTamperedPayload(t *testing.T) {
	secret := []byte("key")
	token := encodeHMAC(secret, []byte("original"))
	parts := splitToken(token)
	newPayload := base64.RawURLEncoding.EncodeToString([]byte("modified"))
	tampered := newPayload + "." + parts[1]
	_, err := decodeHMAC(secret, tampered)
	if err == nil || err.Error() != "invalid signature" {
		t.Fatalf("expected 'invalid signature', got %v", err)
	}
}

// --- token format ---

func TestTokenFormat(t *testing.T) {
	token := encodeHMAC([]byte("s"), []byte("p"))
	parts := splitToken(token)
	if len(parts) != 2 {
		t.Fatalf("token should have exactly 2 parts separated by dot")
	}
	if parts[0] == "" || parts[1] == "" {
		t.Fatal("neither part should be empty")
	}
}

// --- empty secret ---

func TestEmptySecret(t *testing.T) {
	secret := []byte("")
	payload := []byte("test")
	token := encodeHMAC(secret, payload)
	got, err := decodeHMAC(secret, token)
	if err != nil {
		t.Fatalf("empty secret should still work: %v", err)
	}
	if string(got) != "test" {
		t.Fatalf("got %q, want %q", got, "test")
	}
}
