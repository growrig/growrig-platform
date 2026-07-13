package integrations

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

type vault struct{ aead cipher.AEAD }

func openVault(path string) (*vault, error) {
	key, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		if err = os.WriteFile(path, key, 0o600); err != nil {
			return nil, fmt.Errorf("create integration secret key: %w", err)
		}
	} else if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("integration secret key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &vault{aead: aead}, nil
}

func (v *vault) encrypt(values map[string]string) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := v.aead.Seal(nonce, nonce, raw, nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}
func (v *vault) decrypt(encoded string) (map[string]string, error) {
	out := map[string]string{}
	if encoded == "" {
		return out, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	n := v.aead.NonceSize()
	if len(raw) < n {
		return nil, fmt.Errorf("invalid encrypted integration secrets")
	}
	plain, err := v.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(plain, &out)
	return out, err
}
