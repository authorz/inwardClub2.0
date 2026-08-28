package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/inwardclub/server/internal/modules/asset"
)

const maxLegacyImageBytes = 10 << 20

type legacyImage struct {
	kind         string
	id           int64
	relativePath string
	required     bool
}
type legacyImageResult struct {
	key string
	id  int64
	err error
}

func (i *importer) migrateAssets(ctx context.Context) error {
	if i.assetStore.Bucket() == "" {
		return errors.New("Qiniu configuration is required unless -skip-assets is used")
	}
	baseURL, err := url.Parse(i.sourceBase)
	if err != nil {
		return err
	}
	jobs := []legacyImage{}
	for _, item := range []struct {
		kind, table, column string
		required            bool
	}{{"category", "categories", "image", false}, {"product", "products", "image", false}, {"banner", "banner", "image", true}} {
		rows, err := i.source.QueryContext(ctx, fmt.Sprintf("SELECT id,%s FROM %s WHERE %s IS NOT NULL AND %s<>'' ORDER BY id", item.column, item.table, item.column, item.column))
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			var relative string
			if err = rows.Scan(&id, &relative); err != nil {
				rows.Close()
				return err
			}
			jobs = append(jobs, legacyImage{item.kind, id, relative, item.required})
		}
		rows.Close()
	}
	jobCh := make(chan legacyImage)
	resultCh := make(chan legacyImageResult, len(jobs))
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 45 * time.Second}
	for worker := 0; worker < 6; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				id, err := i.ensureAsset(ctx, client, baseURL, job)
				if err != nil && !job.required {
					err = nil
				}
				resultCh <- legacyImageResult{assetMapKey(job.kind, job.id), id, err}
			}
		}()
	}
	go func() {
		for _, job := range jobs {
			jobCh <- job
		}
		close(jobCh)
	}()
	go func() { wg.Wait(); close(resultCh) }()
	for result := range resultCh {
		if result.err != nil {
			return result.err
		}
		if result.id > 0 {
			i.assetIDs[result.key] = result.id
		}
	}
	i.metrics["legacyImagesFound"] = int64(len(jobs))
	i.metrics["legacyImagesBound"] = int64(len(i.assetIDs))
	return nil
}

func (i *importer) ensureAsset(ctx context.Context, client *http.Client, baseURL *url.URL, job legacyImage) (int64, error) {
	ext := strings.ToLower(path.Ext(job.relativePath))
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	objectKey := fmt.Sprintf("inwardclub/%s/%s/v1/%d%s", cleanPart(i.appEnv), job.kind, job.id, ext)
	var id int64
	var status string
	err := i.target.QueryRowContext(ctx, `SELECT id,status FROM assets WHERE object_key=?`, objectKey).Scan(&id, &status)
	if err == nil && (status == "uploaded" || status == "bound") {
		return id, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	resolved, err := baseURL.Parse(strings.TrimLeft(job.relativePath, "/"))
	if err != nil {
		return 0, err
	}
	body, contentType, err := downloadLegacyImage(ctx, client, resolved.String())
	if err != nil {
		return 0, fmt.Errorf("download %s %d: %w", job.kind, job.id, err)
	}
	uploaded, err := i.assetStore.UploadPublicObject(ctx, asset.PublicUploadInput{ObjectKey: objectKey, Reader: bytes.NewReader(body), SizeBytes: int64(len(body)), ContentType: contentType})
	if err != nil {
		return 0, err
	}
	_, err = i.target.ExecContext(ctx, `INSERT INTO assets
		(bucket,object_key,etag,original_filename,content_type,size_bytes,purpose,visibility,status,uploaded_by_type,uploaded_by_id,created_at,uploaded_at)
		VALUES (?,?,?,?,?,?,?,'public','bound','super_admin',1,UTC_TIMESTAMP(),UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE etag=VALUES(etag),content_type=VALUES(content_type),size_bytes=VALUES(size_bytes),status='bound',uploaded_at=UTC_TIMESTAMP(),deleted_at=NULL`,
		i.assetStore.Bucket(), objectKey, uploaded.Hash, path.Base(job.relativePath), contentType, len(body), job.kind)
	if err != nil {
		return 0, err
	}
	if err = i.target.QueryRowContext(ctx, `SELECT id FROM assets WHERE object_key=?`, objectKey).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func downloadLegacyImage(ctx context.Context, client *http.Client, imageURL string) ([]byte, string, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
		if err != nil {
			return nil, "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			last = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxLegacyImageBytes+1))
		resp.Body.Close()
		if readErr != nil {
			last = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			last = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		if len(body) == 0 || len(body) > maxLegacyImageBytes {
			return nil, "", fmt.Errorf("invalid size %d", len(body))
		}
		contentType := http.DetectContentType(body)
		if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
			return nil, "", fmt.Errorf("unexpected content type %s", contentType)
		}
		return body, contentType, nil
	}
	return nil, "", last
}

func assetMapKey(kind string, id int64) string { return fmt.Sprintf("%s:%d", kind, id) }
func cleanPart(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" {
		return "production"
	}
	return strings.ReplaceAll(value, "/", "-")
}
