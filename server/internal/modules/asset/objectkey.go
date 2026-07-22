package asset

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// buildObjectKey generates the fixed-format, server-controlled object key:
// inwardclub/{env}/{purpose}/{yyyy}/{mm}/{assetID}-{random}.{ext}
// The client filename never participates in the path.
func buildObjectKey(env string, purpose UploadPurpose, assetID int64, ext string) (string, error) {
	random, err := randomHex(8) // 16 hex chars
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	return fmt.Sprintf("inwardclub/%s/%s/%04d/%02d/%d-%s.%s",
		env, purpose, now.Year(), int(now.Month()), assetID, random, ext), nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
