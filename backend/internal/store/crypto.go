package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"unigate/backend/internal/errcode"
)

// Crypto reproduces com.unigate.Crypto byte-for-byte. The master key is a
// Base64Std-encoded 32 byte deployment secret (UNIGATE_MASTER_KEY). Existing
// ciphertexts and HMAC digests must remain decryptable/comparable, so the
// version byte, IV layout and GCM tag size are frozen.
type Crypto struct {
	key []byte
}

// NewCrypto decodes and validates the master key.
func NewCrypto(encoded string) (*Crypto, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("UNIGATE_MASTER_KEY must be a Base64 encoded 32 byte deployment secret")
	}
	return &Crypto{key: key}, nil
}

// Token returns 32 random bytes encoded as Base64URL without padding (43 chars).
func (c *Crypto) Token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err)) // process-level invariant, startup only
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Hash returns the raw HMAC-SHA256(key, utf8(s)) bytes.
func (c *Crypto) Hash(s string) []byte {
	m := hmac.New(sha256.New, c.key)
	m.Write([]byte(s))
	return m.Sum(nil)
}

func (c *Crypto) gcm() (cipher.AEAD, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block) // 12 byte nonce, 128 bit tag
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return aead, nil
}

// Seal encrypts text as [0x01][12 byte IV][ciphertext+tag].
func (c *Crypto) Seal(text string) ([]byte, error) {
	aead, err := c.gcm()
	if err != nil {
		return nil, err
	}
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("random iv: %w", err)
	}
	crypt := aead.Seal(nil, iv, []byte(text), nil)
	out := make([]byte, 13+len(crypt))
	out[0] = 1
	copy(out[1:13], iv)
	copy(out[13:], crypt)
	return out, nil
}

// Open decrypts a version-1 blob. Any failure maps to 410 CAPABILITY_EXPIRED
// (canonical TOKEN_EXPIRED) exactly like the Java implementation.
func (c *Crypto) Open(b []byte) (string, error) {
	if len(b) < 13 || b[0] != 1 {
		return "", errcode.New(410, "CAPABILITY_EXPIRED", "凭证已失效")
	}
	aead, err := c.gcm()
	if err != nil {
		return "", errcode.New(410, "CAPABILITY_EXPIRED", "凭证已失效")
	}
	plain, err := aead.Open(nil, b[1:13], b[13:], nil)
	if err != nil {
		return "", errcode.New(410, "CAPABILITY_EXPIRED", "凭证已失效")
	}
	return string(plain), nil
}

// Signed seals the JSON encoding of data and returns Base64URL without padding.
func (c *Crypto) Signed(data map[string]any) (string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal signed payload: %w", err)
	}
	sealed, err := c.Seal(string(raw))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Verify decodes and opens a signed capability token into a map.
func (c *Crypto) Verify(s string) (map[string]any, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, errcode.New(410, "CAPABILITY_EXPIRED", "凭证已失效")
	}
	text, err := c.Open(raw)
	if err != nil {
		return nil, errcode.New(410, "CAPABILITY_EXPIRED", "凭证已失效")
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, errcode.New(410, "CAPABILITY_EXPIRED", "凭证已失效")
	}
	return out, nil
}