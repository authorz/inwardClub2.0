// Package credentialcrypto protects short-lived password confirmations with
// RSA-OAEP in addition to the mandatory HTTPS transport layer.
package credentialcrypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const Algorithm = "RSA-OAEP-256"

// PublicKeyView is returned to an authenticated console immediately before a
// sensitive operation. PublicKey is a base64-encoded SPKI DER value.
type PublicKeyView struct {
	KeyID     string `json:"keyId"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"publicKey"`
}

// Cipher owns the server-side RSA private key used for password confirmations.
type Cipher struct {
	privateKey *rsa.PrivateKey
	publicKey  PublicKeyView
}

// Decryptor is the narrow dependency sensitive-operation handlers use.
type Decryptor interface {
	Decrypt(keyID, ciphertext string) (string, error)
}

// DecryptPassword rejects missing handler wiring instead of falling back to a
// plaintext password request.
func DecryptPassword(decryptor Decryptor, keyID, ciphertext string) (string, error) {
	if decryptor == nil {
		return "", apperr.Internal(fmt.Errorf("credential decryptor is not configured"))
	}
	return decryptor.Decrypt(keyID, ciphertext)
}

// New loads a PEM private key. In non-production local development an empty
// path may be supplied to generate an ephemeral key for the current process.
func New(privateKeyPath string) (*Cipher, error) {
	if strings.TrimSpace(privateKeyPath) == "" {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate credential encryption key: %w", err)
		}
		return NewWithPrivateKey(key)
	}
	raw, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read credential encryption private key: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("credential encryption private key is not PEM")
	}
	var key *rsa.PrivateKey
	if parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes); parseErr == nil {
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("credential encryption private key is not RSA")
		}
	} else {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse credential encryption private key: %w", err)
		}
	}
	return NewWithPrivateKey(key)
}

// NewWithPrivateKey builds a cipher from an already parsed key.
func NewWithPrivateKey(key *rsa.PrivateKey) (*Cipher, error) {
	if key == nil || key.N.BitLen() < 2048 {
		return nil, fmt.Errorf("credential encryption RSA key must be at least 2048 bits")
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal credential encryption public key: %w", err)
	}
	digest := sha256.Sum256(publicDER)
	return &Cipher{
		privateKey: key,
		publicKey: PublicKeyView{
			KeyID:     hex.EncodeToString(digest[:]),
			Algorithm: Algorithm,
			PublicKey: base64.StdEncoding.EncodeToString(publicDER),
		},
	}, nil
}

// PublicKey returns the browser-importable public key metadata.
func (c *Cipher) PublicKey() PublicKeyView { return c.publicKey }

// Decrypt decrypts one RSA-OAEP/SHA-256 password confirmation. Every malformed
// input returns the same public error to avoid exposing a decryption oracle.
func (c *Cipher) Decrypt(keyID, ciphertext string) (string, error) {
	if c == nil || keyID != c.publicKey.KeyID || strings.TrimSpace(ciphertext) == "" {
		return "", invalidEncryptedPassword()
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil || len(raw) != c.privateKey.Size() {
		return "", invalidEncryptedPassword()
	}
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, c.privateKey, raw, nil)
	if err != nil || len(plain) == 0 {
		return "", invalidEncryptedPassword()
	}
	password := string(plain)
	for i := range plain {
		plain[i] = 0
	}
	return password, nil
}

func invalidEncryptedPassword() error {
	return apperr.Invalid("密码加密数据无效，请刷新页面后重试")
}
