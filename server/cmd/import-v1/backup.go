package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-sql-driver/mysql"
)

type backupResult struct{ path, sha256 string }

func backupMySQL(ctx context.Context, dsn, directory string) (backupResult, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return backupResult{}, fmt.Errorf("parse target DSN for backup: %w", err)
	}
	if cfg.Net != "tcp" {
		return backupResult{}, fmt.Errorf("backup requires a tcp MySQL DSN, got %q", cfg.Net)
	}
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return backupResult{}, fmt.Errorf("parse target MySQL address: %w", err)
	}
	if err = os.MkdirAll(directory, 0o700); err != nil {
		return backupResult{}, err
	}
	path := filepath.Join(directory, fmt.Sprintf("%s-before-v1-import-%s.sql.gz", cfg.DBName, time.Now().UTC().Format("20060102T150405Z")))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return backupResult{}, err
	}
	gz := gzip.NewWriter(file)
	if host == "127.0.0.1" || host == "localhost" {
		host = "host.docker.internal"
	}
	if _, err = exec.LookPath("docker"); err != nil {
		_ = gz.Close()
		_ = file.Close()
		_ = os.Remove(path)
		return backupResult{}, errors.New("docker is required for the pinned MySQL 8 backup client")
	}
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--env", "MYSQL_PWD", "mysql:8.0", "mysqldump",
		"--host="+host, "--port="+port, "--user="+cfg.User,
		"--single-transaction", "--quick", "--triggers", "--hex-blob",
		"--default-character-set=utf8mb4", "--set-gtid-purged=OFF", "--no-tablespaces", cfg.DBName)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+cfg.Passwd)
	cmd.Stdout = gz
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	closeGzipErr := gz.Close()
	closeFileErr := file.Close()
	if runErr != nil || closeGzipErr != nil || closeFileErr != nil {
		_ = os.Remove(path)
		if runErr != nil {
			return backupResult{}, fmt.Errorf("mysqldump failed: %w: %s", runErr, stderr.String())
		}
		if closeGzipErr != nil {
			return backupResult{}, closeGzipErr
		}
		return backupResult{}, closeFileErr
	}
	file, err = os.Open(path)
	if err != nil {
		return backupResult{}, err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return backupResult{}, copyErr
	}
	if closeErr != nil {
		return backupResult{}, closeErr
	}
	return backupResult{path: path, sha256: hex.EncodeToString(hash.Sum(nil))}, nil
}
