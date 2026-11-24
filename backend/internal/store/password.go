package store

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher reproduces com.unigate.PasswordHasher byte-for-byte:
// versioned domain-separated SHA-256 prehash + bcrypt(cost 12). Existing
// hashes (both prefixed and legacy bare bcrypt) must keep verifying, so the
// prefix, domain separator and Base64Std prehash encoding are frozen.
// Argon2id from the design docs was a rejected candidate and is NOT used.
const passwordPrefix = "$ug$sha256-bcrypt$v1$"

// PasswordHasher encodes and matches staff passwords.
type PasswordHasher struct{}

func prehash(password string) string {
	d := sha256.New()
	d.Write([]byte("UniGate password v1\x00"))
	d.Write([]byte(password))
	return base64.StdEncoding.EncodeToString(d.Sum(nil))
}

// Encode returns prefix + bcrypt(cost12).encode(prehash).
func (PasswordHasher) Encode(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(prehash(password)), 12)
	if err != nil {
		return "", err
	}
	return passwordPrefix + string(hash), nil
}

// Matches verifies a password against a stored hash. Prefixed hashes compare
// the prehash; legacy bare bcrypt hashes compare the raw password and reject
// passwords longer than bcrypt's 72 byte limit (same as Spring's encoder).
func (PasswordHasher) Matches(password, stored string) bool {
	if stored == "" {
		return false
	}
	if strings.HasPrefix(stored, passwordPrefix) {
		return bcrypt.CompareHashAndPassword([]byte(stored[len(passwordPrefix):]), []byte(prehash(password))) == nil
	}
	if len([]byte(password)) > 72 {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil
}