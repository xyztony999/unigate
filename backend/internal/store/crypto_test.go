package store

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"unigate/backend/internal/errcode"
)

// testMasterKeyB64 is Base64Std of the 32 bytes 0x00..0x1f. It is a fixed test
// vector only, never a production credential.
const testMasterKeyB64 = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="

func testCrypto(t *testing.T) *Crypto {
	t.Helper()
	c, err := NewCrypto(testMasterKeyB64)
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}
	return c
}

func TestCryptoSealOpenRoundTrip(t *testing.T) {
	c := testCrypto(t)
	for _, text := range []string{"", "hello unigate", "活动明细 personal data 密码", strings.Repeat("x", 4096)} {
		sealed, err := c.Seal(text)
		if err != nil {
			t.Fatalf("Seal(%q): %v", text, err)
		}
		got, err := c.Open(sealed)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if got != text {
			t.Fatalf("round trip = %q, want %q", got, text)
		}
	}
}

// TestCryptoSealLayout freezes the version-1 wire format from Crypto.java:
// [0x01][12 byte IV][ciphertext + 16 byte GCM tag]. It independently decrypts
// with stdlib AES-GCM (12 byte nonce, 128 bit tag) to prove the IV placement and
// tag size rather than trusting Open.
func TestCryptoSealLayout(t *testing.T) {
	c := testCrypto(t)
	const text = "layout probe"
	sealed, err := c.Seal(text)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if sealed[0] != 0x01 {
		t.Fatalf("version byte = %#x, want 0x01", sealed[0])
	}
	// 1 version + 12 IV + len(plaintext) + 16 tag.
	if want := 13 + len(text) + 16; len(sealed) != want {
		t.Fatalf("sealed length = %d, want %d (128 bit GCM tag)", len(sealed), want)
	}
	key, _ := base64.StdEncoding.DecodeString(testMasterKeyB64)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes: %v", err)
	}
	aead, err := cipher.NewGCM(block) // default 12 byte nonce / 16 byte tag
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	if aead.NonceSize() != 12 {
		t.Fatalf("nonce size = %d, want 12", aead.NonceSize())
	}
	plain, err := aead.Open(nil, sealed[1:13], sealed[13:], nil)
	if err != nil {
		t.Fatalf("independent GCM open with [1:13] IV / [13:] crypt: %v", err)
	}
	if string(plain) != text {
		t.Fatalf("independent decrypt = %q, want %q", plain, text)
	}
}

// TestCryptoHashKnownAnswer pins hash() to HMAC-SHA256(key, utf8(s)) using a
// fixed key/input vector computed independently (not via Crypto.Hash).
func TestCryptoHashKnownAnswer(t *testing.T) {
	c := testCrypto(t)
	const wantHex = "9ffbd0904921d50424032780bc94cd15a090192858f5198998136ad3da97ac9b"
	got := c.Hash("hello unigate")
	if hex.EncodeToString(got) != wantHex {
		t.Fatalf("Hash = %s, want %s", hex.EncodeToString(got), wantHex)
	}
	if len(got) != 32 {
		t.Fatalf("Hash length = %d, want 32", len(got))
	}
}

func TestCryptoToken(t *testing.T) {
	c := testCrypto(t)
	seen := map[string]bool{}
	for i := 0; i < 16; i++ {
		tok := c.Token()
		if len(tok) != 43 {
			t.Fatalf("token length = %d, want 43 (32 bytes Base64URL no padding)", len(tok))
		}
		if strings.ContainsAny(tok, "=+/") {
			t.Fatalf("token %q contains padding or non-URL chars", tok)
		}
		raw, err := base64.RawURLEncoding.DecodeString(tok)
		if err != nil || len(raw) != 32 {
			t.Fatalf("token does not decode to 32 bytes: %v (len %d)", err, len(raw))
		}
		if seen[tok] {
			t.Fatalf("token repeated: %q", tok)
		}
		seen[tok] = true
	}
}

func TestNewCryptoRejectsBadMasterKey(t *testing.T) {
	cases := map[string]string{
		"empty":      "",
		"not base64": "!!!not-base64!!!",
		"31 bytes":   base64.StdEncoding.EncodeToString(make([]byte, 31)),
		"33 bytes":   base64.StdEncoding.EncodeToString(make([]byte, 33)),
		"16 bytes":   base64.StdEncoding.EncodeToString(make([]byte, 16)),
	}
	for name, encoded := range cases {
		if _, err := NewCrypto(encoded); err == nil {
			t.Errorf("%s: NewCrypto accepted invalid key", name)
		}
	}
	if _, err := NewCrypto(testMasterKeyB64); err != nil {
		t.Fatalf("valid 32 byte key rejected: %v", err)
	}
}

func TestCryptoOpenRejectsTamperAndVersion(t *testing.T) {
	c := testCrypto(t)
	sealed, err := c.Seal("tamper probe")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	assertExpired := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("expected failure, got nil")
		}
		ec := errcode.AsError(err)
		if ec == nil {
			t.Fatalf("error is not *errcode.Error: %v", err)
		}
		if ec.Status != 410 || ec.Code != "TOKEN_EXPIRED" {
			t.Fatalf("err = %d/%s, want 410/TOKEN_EXPIRED (canonical CAPABILITY_EXPIRED)", ec.Status, ec.Code)
		}
	}

	t.Run("wrong version byte", func(t *testing.T) {
		bad := append([]byte(nil), sealed...)
		bad[0] = 0x02
		_, err := c.Open(bad)
		assertExpired(t, err)
	})
	t.Run("zero version byte", func(t *testing.T) {
		bad := append([]byte(nil), sealed...)
		bad[0] = 0x00
		_, err := c.Open(bad)
		assertExpired(t, err)
	})
	t.Run("flipped ciphertext byte", func(t *testing.T) {
		bad := append([]byte(nil), sealed...)
		bad[len(bad)-1] ^= 0xFF
		_, err := c.Open(bad)
		assertExpired(t, err)
	})
	t.Run("flipped IV byte", func(t *testing.T) {
		bad := append([]byte(nil), sealed...)
		bad[1] ^= 0xFF
		_, err := c.Open(bad)
		assertExpired(t, err)
	})
	t.Run("too short", func(t *testing.T) {
		_, err := c.Open(sealed[:12])
		assertExpired(t, err)
	})
	t.Run("wrong key", func(t *testing.T) {
		other, err := NewCrypto(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
		if err != nil {
			t.Fatalf("NewCrypto: %v", err)
		}
		_, err = other.Open(sealed)
		assertExpired(t, err)
	})
}

func TestCryptoSignedVerifyRoundTrip(t *testing.T) {
	c := testCrypto(t)
	data := map[string]any{"sub": "abc", "n": float64(3)}
	tok, err := c.Signed(data)
	if err != nil {
		t.Fatalf("Signed: %v", err)
	}
	if strings.ContainsAny(tok, "=+/") {
		t.Fatalf("signed token %q is not Base64URL without padding", tok)
	}
	got, err := c.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got["sub"] != "abc" {
		t.Fatalf("Verify sub = %v, want abc", got["sub"])
	}
	if _, err := c.Verify("!!!invalid!!!"); err == nil {
		t.Fatalf("Verify accepted invalid token")
	}
}
