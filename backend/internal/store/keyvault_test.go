package store

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"unigate/backend/internal/errcode"
)

const (
	testEventA = "123e4567-e89b-12d3-a456-426614174000"
	testEventB = "123e4567-e89b-12d3-a456-426614174001"
)

func testVault(t *testing.T) (*KeyVault, string) {
	t.Helper()
	dir := t.TempDir()
	v, err := NewKeyVault(testCrypto(t), dir)
	if err != nil {
		t.Fatalf("NewKeyVault: %v", err)
	}
	return v, dir
}

func assertTokenExpired(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected failure, got nil")
	}
	ec := errcode.AsError(err)
	if ec == nil {
		t.Fatalf("error is not *errcode.Error: %v", err)
	}
	if ec.Status != 410 || ec.Code != "TOKEN_EXPIRED" {
		t.Fatalf("err = %d/%s, want 410/TOKEN_EXPIRED", ec.Status, ec.Code)
	}
}

func TestKeyVaultSealOpenRoundTrip(t *testing.T) {
	v, _ := testVault(t)
	for _, text := range []string{"", "event personal data 活动明细", "row json {\"a\":1}"} {
		sealed, err := v.Seal(testEventA, text)
		if err != nil {
			t.Fatalf("Seal: %v", err)
		}
		got, err := v.Open(sealed)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if got != text {
			t.Fatalf("round trip = %q, want %q", got, text)
		}
	}
}

// TestKeyVaultSealLayout freezes the version-2 wire format from EventKeyVault.java:
// [0x02][36 byte event UUID ASCII][12 byte IV][crypt + 16 byte tag], with the
// event UUID ASCII bound as AAD. It recovers the DEK from the .key file and
// independently decrypts to prove the layout and AAD.
func TestKeyVaultSealLayout(t *testing.T) {
	v, dir := testVault(t)
	const text = "layout probe"
	sealed, err := v.Seal(testEventA, text)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if sealed[0] != 0x02 {
		t.Fatalf("version byte = %#x, want 0x02", sealed[0])
	}
	if got := string(sealed[1:37]); got != testEventA {
		t.Fatalf("[1:37] = %q, want event UUID %q", got, testEventA)
	}
	if want := 49 + len(text) + 16; len(sealed) != want {
		t.Fatalf("sealed length = %d, want %d", len(sealed), want)
	}

	// Recover the DEK exactly as key() does: master.Open(.key) then Base64Std.
	keyFile, err := os.ReadFile(filepath.Join(dir, testEventA+".key"))
	if err != nil {
		t.Fatalf("read .key: %v", err)
	}
	if keyFile[0] != 0x01 {
		t.Fatalf(".key is not a version-1 master seal: %#x", keyFile[0])
	}
	dekPlain, err := testCrypto(t).Open(keyFile)
	if err != nil {
		t.Fatalf("master.Open(.key): %v", err)
	}
	dek, err := base64.StdEncoding.DecodeString(dekPlain)
	if err != nil || len(dek) != 32 {
		t.Fatalf("DEK decode: %v (len %d)", err, len(dek))
	}
	block, _ := aes.NewCipher(dek)
	aead, _ := cipher.NewGCM(block)
	plain, err := aead.Open(nil, sealed[37:49], sealed[49:], []byte(testEventA))
	if err != nil {
		t.Fatalf("independent GCM open with event AAD: %v", err)
	}
	if string(plain) != text {
		t.Fatalf("independent decrypt = %q, want %q", plain, text)
	}
	// AAD is bound to the event UUID: a different AAD must fail.
	if _, err := aead.Open(nil, sealed[37:49], sealed[49:], []byte(testEventB)); err == nil {
		t.Fatalf("GCM open succeeded with wrong AAD; AAD is not bound to event UUID")
	}
}

func TestKeyVaultOpenCrossEventFails(t *testing.T) {
	v, _ := testVault(t)
	sealed, err := v.Seal(testEventA, "cross event")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	// Re-point the embedded UUID at an event that has no DEK.
	bad := append([]byte(nil), sealed...)
	copy(bad[1:37], []byte(testEventB))
	_, err = v.Open(bad)
	assertTokenExpired(t, err)

	// Even with a DEK present for the other event, AAD/DEK mismatch must fail.
	if _, err := v.Seal(testEventB, "seed B"); err != nil {
		t.Fatalf("seed B: %v", err)
	}
	_, err = v.Open(bad)
	assertTokenExpired(t, err)
}

func TestKeyVaultOpenRejectsShortAndTamper(t *testing.T) {
	v, _ := testVault(t)
	sealed, err := v.Seal(testEventA, "probe")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := v.Open(sealed[:48]); err == nil {
		t.Fatalf("Open accepted <49 byte blob")
	}
	// Flipping a ciphertext byte breaks the GCM tag.
	bad := append([]byte(nil), sealed...)
	bad[len(bad)-1] ^= 0xFF
	_, err = v.Open(bad)
	assertTokenExpired(t, err)
}

func TestKeyVaultRevoke(t *testing.T) {
	v, dir := testVault(t)
	sealed, err := v.Seal(testEventA, "to be revoked")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if err := v.RequireNotRevoked(testEventA); err != nil {
		t.Fatalf("RequireNotRevoked before revoke: %v", err)
	}
	if err := v.Revoke(testEventA); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	// Tombstone written, DEK file removed.
	if _, err := os.Stat(filepath.Join(dir, testEventA+".revoked")); err != nil {
		t.Fatalf(".revoked tombstone missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, testEventA+".key")); !os.IsNotExist(err) {
		t.Fatalf(".key still present after revoke: %v", err)
	}
	// requireNotRevoked now fails with the distinct revoked message.
	err = v.RequireNotRevoked(testEventA)
	assertTokenExpired(t, err)
	if ec := errcode.AsError(err); ec == nil || ec.Message != "活动数据密钥已撤销" {
		t.Fatalf("revoked message = %v, want 活动数据密钥已撤销", err)
	}
	// Existing ciphertext can no longer be opened, and new seals are refused.
	_, err = v.Open(sealed)
	assertTokenExpired(t, err)
	_, err = v.Seal(testEventA, "after revoke")
	assertTokenExpired(t, err)
}

func TestKeyVaultRejectsInvalidUUID(t *testing.T) {
	v, _ := testVault(t)
	for _, bad := range []string{"../evil", "not-a-uuid", "", "123e4567-e89b-12d3-a456-42661417400g"} {
		if _, err := v.Seal(bad, "x"); err == nil {
			t.Errorf("Seal accepted invalid uuid %q", bad)
		}
	}
}
