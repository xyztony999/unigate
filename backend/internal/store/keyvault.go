package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"unigate/backend/internal/errcode"
)

// KeyVault reproduces com.unigate.EventKeyVault. Event data keys live outside
// database backups; revocation tombstones survive database restoration. The
// sealed blob layout [0x02][36 byte event UUID ASCII][12 byte IV][crypt+tag]
// and the event-UUID AAD are frozen for byte-level compatibility.
type KeyVault struct {
	root   string
	master *Crypto
	mu     sync.Mutex
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// NewKeyVault resolves the key directory to an absolute normalized path.
func NewKeyVault(master *Crypto, directory string) (*KeyVault, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve key directory: %w", err)
	}
	return &KeyVault{root: filepath.Clean(root), master: master}, nil
}

// file validates the event UUID before touching the filesystem (path escape
// prevention) and returns the managed path for the given suffix.
func (v *KeyVault) file(event, suffix string) (string, error) {
	if !uuidPattern.MatchString(event) {
		return "", fmt.Errorf("invalid event uuid")
	}
	return filepath.Join(v.root, event+suffix), nil
}

// RequireNotRevoked fails with 410 TOKEN_EXPIRED when a tombstone exists.
func (v *KeyVault) RequireNotRevoked(event string) error {
	p, err := v.file(event, ".revoked")
	if err != nil {
		return errcode.New(410, "TOKEN_EXPIRED", "活动数据密钥已撤销")
	}
	if _, err := os.Stat(p); err == nil {
		return errcode.New(410, "TOKEN_EXPIRED", "活动数据密钥已撤销")
	}
	return nil
}

// key loads (or creates) the DEK for an event. Caller holds v.mu.
func (v *KeyVault) key(event string, create bool) ([]byte, error) {
	if err := os.MkdirAll(v.root, 0o700); err != nil {
		return nil, fmt.Errorf("create key directory: %w", err)
	}
	if err := v.RequireNotRevoked(event); err != nil {
		return nil, err
	}
	p, err := v.file(event, ".key")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if !create {
			return nil, errcode.New(410, "TOKEN_EXPIRED", "活动数据密钥不可用")
		}
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("random dek: %w", err)
		}
		sealed, err := v.master.Seal(base64.StdEncoding.EncodeToString(raw))
		if err != nil {
			return nil, err
		}
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, werr := f.Write(sealed)
			cerr := f.Close()
			if werr != nil {
				return nil, fmt.Errorf("write dek: %w", werr)
			}
			if cerr != nil {
				return nil, fmt.Errorf("close dek: %w", cerr)
			}
		} else if !os.IsExist(err) {
			return nil, fmt.Errorf("create dek file: %w", err)
		}
		// FileAlreadyExists is ignored: another writer won the race.
	}
	sealed, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read dek: %w", err)
	}
	plain, err := v.master.Open(sealed)
	if err != nil {
		return nil, err
	}
	dek, err := base64.StdEncoding.DecodeString(plain)
	if err != nil {
		return nil, fmt.Errorf("decode dek: %w", err)
	}
	return dek, nil
}

func gcmWithKey(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// Seal encrypts text under the event DEK with the event UUID ASCII as AAD.
func (v *KeyVault) Seal(event, text string) ([]byte, error) {
	v.mu.Lock()
	dek, err := v.key(event, true)
	v.mu.Unlock()
	if err != nil {
		if ec := errcode.AsError(err); ec != nil {
			return nil, ec
		}
		return nil, fmt.Errorf("event key storage unavailable: %w", err)
	}
	aead, err := gcmWithKey(dek)
	if err != nil {
		return nil, fmt.Errorf("event key storage unavailable: %w", err)
	}
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("event key storage unavailable: %w", err)
	}
	aad := []byte(event) // ASCII UUID
	crypt := aead.Seal(nil, iv, []byte(text), aad)
	out := make([]byte, 49+len(crypt))
	out[0] = 2
	copy(out[1:37], aad)
	copy(out[37:49], iv)
	copy(out[49:], crypt)
	return out, nil
}

// Open decrypts a version-2 blob. Failures map to 410 TOKEN_EXPIRED; a
// revocation tombstone keeps its distinct "已撤销" message.
func (v *KeyVault) Open(data []byte) (string, error) {
	if len(data) < 49 {
		return "", errcode.New(410, "TOKEN_EXPIRED", "活动数据密钥不可用")
	}
	event := string(data[1:37])
	v.mu.Lock()
	dek, err := v.key(event, false)
	v.mu.Unlock()
	if err != nil {
		if ec := errcode.AsError(err); ec != nil {
			return "", ec
		}
		return "", errcode.New(410, "TOKEN_EXPIRED", "活动数据密钥不可用")
	}
	aead, err := gcmWithKey(dek)
	if err != nil {
		return "", errcode.New(410, "TOKEN_EXPIRED", "活动数据密钥不可用")
	}
	plain, err := aead.Open(nil, data[37:49], data[49:], []byte(event))
	if err != nil {
		return "", errcode.New(410, "TOKEN_EXPIRED", "活动数据密钥不可用")
	}
	return string(plain), nil
}

// Revoke writes the tombstone and deletes the DEK file.
func (v *KeyVault) Revoke(event string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := os.MkdirAll(v.root, 0o700); err != nil {
		return fmt.Errorf("cannot revoke event key: %w", err)
	}
	tomb, err := v.file(event, ".revoked")
	if err != nil {
		return fmt.Errorf("cannot revoke event key: %w", err)
	}
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(tomb, []byte(stamp), 0o600); err != nil {
		return fmt.Errorf("cannot revoke event key: %w", err)
	}
	keyFile, err := v.file(event, ".key")
	if err != nil {
		return fmt.Errorf("cannot revoke event key: %w", err)
	}
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot revoke event key: %w", err)
	}
	return nil
}