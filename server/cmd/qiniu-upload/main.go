// Command qiniu-upload is a reusable CLI that uploads a local file to the
// public Qiniu bucket using the shared asset.QiniuObjectStore. It loads config
// from server/.env and never prints secrets.
//
// Usage:
//
//	go run ./cmd/qiniu-upload -file ./logo.png -key seed/logo.png
//	go run ./cmd/qiniu-upload -file ./logo.png -key seed/logo.png -content-type image/png
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/inwardclub/server/internal/modules/asset"
	"github.com/inwardclub/server/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "qiniu-upload error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		filePath    = flag.String("file", "", "local file path to upload (required)")
		objectKey   = flag.String("key", "", "qiniu object key (required)")
		contentType = flag.String("content-type", "", "optional content type, e.g. image/png")
		envPath     = flag.String("env", "", "path to .env (default: .env then server/.env)")
	)
	flag.Parse()

	if *filePath == "" || *objectKey == "" {
		flag.Usage()
		return fmt.Errorf("-file and -key are required")
	}

	if err := loadEnv(*envPath); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	q := cfg.Qiniu
	if q.AccessKey == "" || q.SecretKey == "" || q.Bucket == "" || q.PublicDomain == "" {
		return fmt.Errorf("missing QINIU_ACCESS_KEY / QINIU_SECRET_KEY / QINIU_BUCKET / QINIU_PUBLIC_DOMAIN")
	}

	f, err := os.Open(*filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	store := asset.NewQiniuObjectStore(q)
	res, err := store.UploadPublicObject(context.Background(), asset.PublicUploadInput{
		ObjectKey:   *objectKey,
		Reader:      f,
		SizeBytes:   info.Size(),
		ContentType: *contentType,
	})
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// loadEnv parses a dotenv file and sets any keys not already present in the
// environment. It never logs the values it reads.
func loadEnv(path string) error {
	if path == "" {
		for _, candidate := range []string{".env", "server/.env"} {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path == "" {
		return fmt.Errorf("no .env found (looked for .env and server/.env); pass -env")
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open env: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, val); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}
