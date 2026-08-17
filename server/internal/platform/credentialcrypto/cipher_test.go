package credentialcrypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestCipherDecryptsRSAOAEPPassword(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := NewWithPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &key.PublicKey, []byte("correct horse"), nil)
	if err != nil {
		t.Fatal(err)
	}
	password, err := cipher.Decrypt(cipher.PublicKey().KeyID, base64.StdEncoding.EncodeToString(encrypted))
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if password != "correct horse" {
		t.Fatalf("unexpected password %q", password)
	}
}

func TestCipherRejectsWrongKeyAndMalformedCiphertext(t *testing.T) {
	cipher, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		keyID, ciphertext string
	}{
		{"wrong", base64.StdEncoding.EncodeToString(make([]byte, 256))},
		{cipher.PublicKey().KeyID, "not-base64"},
		{cipher.PublicKey().KeyID, base64.StdEncoding.EncodeToString(make([]byte, 256))},
	} {
		if _, err := cipher.Decrypt(tc.keyID, tc.ciphertext); err == nil {
			t.Fatalf("expected rejection for key=%q ciphertext=%q", tc.keyID, tc.ciphertext)
		}
	}
}
