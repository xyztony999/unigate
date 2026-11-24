package store

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Known-answer prehash vectors: Base64Std(SHA256("UniGate password v1\x00" + pw)),
// computed independently of PasswordHasher. They freeze the domain separator and
// the Base64Std encoding so existing Java hashes keep verifying.
const (
	prehashASCII   = "hy7/ig9piQNhQH4VBzSiONZ9CKJR0oMbPzQVX0Dac3w=" // "correct horse battery staple"
	prehashUnicode = "7YQcgcKBe0Db90hm28Db+HVfCX6nmVUZOfifp+r4IGM=" // "密码Pässw0rd!"
)

func TestPasswordPrehashKnownAnswer(t *testing.T) {
	cases := []struct{ pw, want string }{
		{"correct horse battery staple", prehashASCII},
		{"密码Pässw0rd!", prehashUnicode},
	}
	for _, tc := range cases {
		if got := prehash(tc.pw); got != tc.want {
			t.Fatalf("prehash(%q) = %s, want %s", tc.pw, got, tc.want)
		}
		// Domain separation: the prehash must differ from a plain SHA256(pw).
		plain := sha256.Sum256([]byte(tc.pw))
		if prehash(tc.pw) == base64.StdEncoding.EncodeToString(plain[:]) {
			t.Fatalf("prehash(%q) equals plain SHA256; domain separator missing", tc.pw)
		}
	}
}

func TestPasswordEncodePrefixAndCost(t *testing.T) {
	h := PasswordHasher{}
	const pw = "correct horse battery staple"
	encoded, err := h.Encode(pw)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	const prefix = "$ug$sha256-bcrypt$v1$"
	if !strings.HasPrefix(encoded, prefix) {
		t.Fatalf("encoded hash %q lacks prefix %q", encoded, prefix)
	}
	bare := encoded[len(prefix):]
	cost, err := bcrypt.Cost([]byte(bare))
	if err != nil {
		t.Fatalf("bcrypt.Cost: %v", err)
	}
	if cost != 12 {
		t.Fatalf("bcrypt cost = %d, want 12", cost)
	}
	// The bcrypt payload must be over the known prehash, not the raw password.
	if err := bcrypt.CompareHashAndPassword([]byte(bare), []byte(prehashASCII)); err != nil {
		t.Fatalf("bcrypt payload is not the domain-separated prehash: %v", err)
	}
}

func TestPasswordMatchesRoundTrip(t *testing.T) {
	h := PasswordHasher{}
	for _, pw := range []string{"correct horse battery staple", "密码Pässw0rd!", ""} {
		encoded, err := h.Encode(pw)
		if err != nil {
			t.Fatalf("Encode(%q): %v", pw, err)
		}
		if !h.Matches(pw, encoded) {
			t.Fatalf("Matches(%q) = false, want true", pw)
		}
		if h.Matches(pw+"wrong", encoded) {
			t.Fatalf("Matches accepted wrong password for %q", pw)
		}
	}
	if h.Matches("anything", "") {
		t.Fatalf("Matches accepted empty stored hash")
	}
}

// TestPasswordLongPasswordPrefixed verifies the prehash keeps passwords beyond
// bcrypt's 72 byte limit usable under the prefixed scheme.
func TestPasswordLongPasswordPrefixed(t *testing.T) {
	h := PasswordHasher{}
	long := strings.Repeat("a", 200)
	encoded, err := h.Encode(long)
	if err != nil {
		t.Fatalf("Encode long: %v", err)
	}
	if !h.Matches(long, encoded) {
		t.Fatalf("Matches(long) = false, want true (prehash bypasses 72 byte limit)")
	}
}

// TestPasswordLegacyBareBcrypt verifies existing bare bcrypt hashes (no prefix)
// still match the raw password, exactly like Spring's BCryptPasswordEncoder.
func TestPasswordLegacyBareBcrypt(t *testing.T) {
	h := PasswordHasher{}
	const pw = "legacy password"
	bare, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		t.Fatalf("generate legacy bcrypt: %v", err)
	}
	stored := string(bare)
	if strings.HasPrefix(stored, "$ug$") {
		t.Fatalf("test fixture unexpectedly prefixed")
	}
	if !h.Matches(pw, stored) {
		t.Fatalf("Matches legacy bare bcrypt = false, want true")
	}
	if h.Matches("wrong", stored) {
		t.Fatalf("Matches accepted wrong password against legacy hash")
	}
}

// TestPasswordLegacyOver72Rejected verifies a legacy bare hash rejects passwords
// longer than 72 bytes (bcrypt limit), matching PasswordHasher.matches.
func TestPasswordLegacyOver72Rejected(t *testing.T) {
	h := PasswordHasher{}
	bare, err := bcrypt.GenerateFromPassword([]byte(strings.Repeat("a", 72)), 12)
	if err != nil {
		t.Fatalf("generate legacy bcrypt: %v", err)
	}
	stored := string(bare)
	if !h.Matches(strings.Repeat("a", 72), stored) {
		t.Fatalf("Matches(72 bytes) = false, want true")
	}
	if h.Matches(strings.Repeat("a", 73), stored) {
		t.Fatalf("Matches(73 bytes) = true, want false (legacy >72 byte rejection)")
	}
}
