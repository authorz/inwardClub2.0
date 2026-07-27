// Command import-v1-catalog imports the v1 categories/products dump into one
// v2 store. It downloads legacy images, uploads them to Qiniu under stable
// relative object keys, records assets, and upserts catalog rows so reruns do
// not duplicate data.
package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/inwardclub/server/internal/modules/asset"
	"github.com/inwardclub/server/internal/platform/config"
	appdb "github.com/inwardclub/server/internal/platform/db"
)

const maxImageBytes = 10 << 20

type legacyCategory struct {
	ID        int64
	ParentID  *int64
	Name      string
	Image     string
	SortOrder int
	Status    string
	CreatedAt string
	UpdatedAt string
}

type legacyProduct struct {
	ID          int64
	CategoryID  int64
	Name        string
	Image       string
	Description string
	PriceCent   int64
	Stock       int64
	Type        int
	Points      int64
	SortOrder   int
	Status      string
	CreatedAt   string
	UpdatedAt   string
}

type imageJob struct {
	MapKey      string
	Purpose     string
	LegacyID    int64
	RelativeURL string
	Required    bool
}

type imageResult struct {
	MapKey  string
	AssetID int64
	Err     error
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "import-v1-catalog error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dumpPath   = flag.String("sql", "../tmp/inwardclub-data.sql", "path to the v1 SQL dump")
		storeID    = flag.Int64("store-id", 1, "target v2 store id")
		sourceBase = flag.String("source-base", "https://api.inwardclub.com/storage/", "legacy image URL prefix")
		workers    = flag.Int("workers", 6, "concurrent image upload workers")
		dryRun     = flag.Bool("dry-run", false, "parse and validate without network or database writes")
	)
	flag.Parse()

	if *storeID <= 0 {
		return fmt.Errorf("store-id must be positive")
	}
	if *workers < 1 || *workers > 16 {
		return fmt.Errorf("workers must be between 1 and 16")
	}
	categories, products, err := parseDump(*dumpPath)
	if err != nil {
		return err
	}
	if err := validateLegacyData(categories, products); err != nil {
		return err
	}
	fmt.Printf("parsed %d categories and %d products\n", len(categories), len(products))
	if *dryRun {
		return nil
	}

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := appdb.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := requireStore(ctx, db, *storeID); err != nil {
		return err
	}

	baseURL, err := url.Parse(*sourceBase)
	if err != nil {
		return fmt.Errorf("parse source-base: %w", err)
	}
	qiniu := asset.NewQiniuObjectStore(cfg.Qiniu)
	assetIDs, err := migrateImages(ctx, db, qiniu, cfg.AppEnv, baseURL, categories, products, *workers)
	if err != nil {
		return err
	}
	if err := upsertCatalog(ctx, db, *storeID, categories, products, assetIDs); err != nil {
		return err
	}
	if err := verifyCatalog(ctx, db, *storeID, categories, products); err != nil {
		return err
	}
	fmt.Printf("import complete: store=%d categories=%d products=%d images=%d\n", *storeID, len(categories), len(products), len(assetIDs))
	return nil
}

func parseDump(filename string) ([]legacyCategory, []legacyProduct, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("open dump: %w", err)
	}
	defer f.Close()

	var categories []legacyCategory
	var products []legacyProduct
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "INSERT INTO `categories`"):
			rows, err := parseInsert(line)
			if err != nil {
				return nil, nil, fmt.Errorf("parse categories: %w", err)
			}
			for _, row := range rows {
				category, err := categoryFromRow(row)
				if err != nil {
					return nil, nil, err
				}
				categories = append(categories, category)
			}
		case strings.HasPrefix(line, "INSERT INTO `products`"):
			rows, err := parseInsert(line)
			if err != nil {
				return nil, nil, fmt.Errorf("parse products: %w", err)
			}
			for _, row := range rows {
				product, err := productFromRow(row)
				if err != nil {
					return nil, nil, err
				}
				products = append(products, product)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan dump: %w", err)
	}
	if len(categories) == 0 || len(products) == 0 {
		return nil, nil, fmt.Errorf("dump did not contain categories and products inserts")
	}
	return categories, products, nil
}

func parseInsert(line string) ([]map[string]string, error) {
	marker := ") VALUES "
	markerAt := strings.Index(line, marker)
	columnsAt := strings.IndexByte(line, '(')
	if columnsAt < 0 || markerAt < 0 || markerAt <= columnsAt {
		return nil, fmt.Errorf("unsupported INSERT format")
	}
	columnText := line[columnsAt+1 : markerAt]
	columnParts := strings.Split(columnText, ",")
	columns := make([]string, 0, len(columnParts))
	for _, column := range columnParts {
		columns = append(columns, strings.Trim(strings.TrimSpace(column), "`"))
	}
	tuples, err := splitSQLTuples(strings.TrimSuffix(line[markerAt+len(marker):], ";"))
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(tuples))
	for _, values := range tuples {
		if len(values) != len(columns) {
			return nil, fmt.Errorf("column/value count mismatch: %d != %d", len(columns), len(values))
		}
		row := make(map[string]string, len(columns))
		for i, column := range columns {
			row[column] = values[i]
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func splitSQLTuples(input string) ([][]string, error) {
	var rows [][]string
	var row []string
	var field strings.Builder
	depth := 0
	quoted := false
	escaped := false
	for _, ch := range input {
		if quoted {
			field.WriteRune(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == '\'' {
				quoted = false
			}
			continue
		}
		switch ch {
		case '\'':
			if depth == 1 {
				quoted = true
				field.WriteRune(ch)
			}
		case '(':
			depth++
			if depth > 1 {
				field.WriteRune(ch)
			}
		case ')':
			if depth != 1 {
				return nil, fmt.Errorf("unexpected closing parenthesis")
			}
			row = append(row, strings.TrimSpace(field.String()))
			field.Reset()
			rows = append(rows, row)
			row = nil
			depth--
		case ',':
			if depth == 1 {
				row = append(row, strings.TrimSpace(field.String()))
				field.Reset()
			}
		default:
			if depth == 1 {
				field.WriteRune(ch)
			}
		}
	}
	if quoted || depth != 0 {
		return nil, fmt.Errorf("unterminated SQL tuple")
	}
	return rows, nil
}

func categoryFromRow(row map[string]string) (legacyCategory, error) {
	id, err := requiredInt(row, "id")
	if err != nil {
		return legacyCategory{}, err
	}
	parentID, err := nullableInt(row["parent_id"])
	if err != nil {
		return legacyCategory{}, fmt.Errorf("category %d parent_id: %w", id, err)
	}
	sortOrder, err := requiredInt(row, "sort_order")
	if err != nil {
		return legacyCategory{}, err
	}
	return legacyCategory{
		ID: id, ParentID: parentID, Name: sqlString(row["name"]), Image: sqlString(row["image"]),
		SortOrder: int(sortOrder), Status: sqlString(row["status"]),
		CreatedAt: sqlString(row["created_at"]), UpdatedAt: sqlString(row["updated_at"]),
	}, nil
}

func productFromRow(row map[string]string) (legacyProduct, error) {
	id, err := requiredInt(row, "id")
	if err != nil {
		return legacyProduct{}, err
	}
	categoryID, err := requiredInt(row, "category_id")
	if err != nil {
		return legacyProduct{}, err
	}
	stock, err := requiredInt(row, "stock")
	if err != nil {
		return legacyProduct{}, err
	}
	typeID, err := requiredInt(row, "type")
	if err != nil {
		return legacyProduct{}, err
	}
	points, err := nullableInt(row["points"])
	if err != nil {
		return legacyProduct{}, err
	}
	sortOrder, err := nullableInt(row["sort_order"])
	if err != nil {
		return legacyProduct{}, err
	}
	priceCent, err := decimalToCent(row["price"])
	if err != nil {
		return legacyProduct{}, fmt.Errorf("product %d price: %w", id, err)
	}
	var pointValue, sortValue int64
	if points != nil {
		pointValue = *points
	}
	if sortOrder != nil {
		sortValue = *sortOrder
	}
	return legacyProduct{
		ID: id, CategoryID: categoryID, Name: sqlString(row["name"]), Image: sqlString(row["image"]),
		Description: sqlString(row["description"]), PriceCent: priceCent, Stock: stock,
		Type: int(typeID), Points: pointValue, SortOrder: int(sortValue), Status: sqlString(row["status"]),
		CreatedAt: sqlString(row["created_at"]), UpdatedAt: sqlString(row["updated_at"]),
	}, nil
}

func requiredInt(row map[string]string, key string) (int64, error) {
	value, err := nullableInt(row[key])
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if value == nil {
		return 0, fmt.Errorf("%s is NULL", key)
	}
	return *value, nil
}

func nullableInt(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "NULL") {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func decimalToCent(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid decimal %q", raw)
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 2 {
		return 0, fmt.Errorf("more than two decimal places")
	}
	fraction += strings.Repeat("0", 2-len(fraction))
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return whole*100 + cents, nil
}

func sqlString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "NULL") {
		return ""
	}
	if len(raw) < 2 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return raw
	}
	raw = raw[1 : len(raw)-1]
	var out strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' || i+1 >= len(raw) {
			out.WriteByte(raw[i])
			continue
		}
		i++
		switch raw[i] {
		case '0':
			out.WriteByte(0)
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case 'b':
			out.WriteByte('\b')
		case 'Z':
			out.WriteByte(26)
		default:
			out.WriteByte(raw[i])
		}
	}
	return out.String()
}

func validateLegacyData(categories []legacyCategory, products []legacyProduct) error {
	categoryIDs := make(map[int64]struct{}, len(categories))
	categoryNames := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		if category.ID <= 0 || strings.TrimSpace(category.Name) == "" {
			return fmt.Errorf("invalid category: %+v", category)
		}
		if _, exists := categoryIDs[category.ID]; exists {
			return fmt.Errorf("duplicate category id %d", category.ID)
		}
		if _, exists := categoryNames[category.Name]; exists {
			return fmt.Errorf("duplicate category name %q", category.Name)
		}
		categoryIDs[category.ID] = struct{}{}
		categoryNames[category.Name] = struct{}{}
	}
	productIDs := make(map[int64]struct{}, len(products))
	for _, product := range products {
		if product.ID <= 0 || strings.TrimSpace(product.Name) == "" {
			return fmt.Errorf("invalid product: %+v", product)
		}
		if _, exists := productIDs[product.ID]; exists {
			return fmt.Errorf("duplicate product id %d", product.ID)
		}
		if _, exists := categoryIDs[product.CategoryID]; !exists {
			return fmt.Errorf("product %d references missing category %d", product.ID, product.CategoryID)
		}
		productIDs[product.ID] = struct{}{}
	}
	return nil
}

func requireStore(ctx context.Context, db *appdb.DB, storeID int64) error {
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stores WHERE id = ?)`, storeID).Scan(&exists); err != nil {
		return fmt.Errorf("check target store: %w", err)
	}
	if !exists {
		return fmt.Errorf("target store %d does not exist", storeID)
	}
	return nil
}

func migrateImages(ctx context.Context, db *appdb.DB, objectStore asset.ObjectStore, env string, baseURL *url.URL, categories []legacyCategory, products []legacyProduct, workers int) (map[string]int64, error) {
	jobs := make([]imageJob, 0, len(categories)+len(products))
	for _, category := range categories {
		if category.Image != "" {
			jobs = append(jobs, imageJob{MapKey: imageMapKey("category", category.ID), Purpose: "category", LegacyID: category.ID, RelativeURL: category.Image})
		}
	}
	for _, product := range products {
		if product.Image != "" {
			jobs = append(jobs, imageJob{MapKey: imageMapKey("product", product.ID), Purpose: "product", LegacyID: product.ID, RelativeURL: product.Image, Required: true})
		}
	}

	jobCh := make(chan imageJob)
	resultCh := make(chan imageResult, len(jobs))
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 45 * time.Second}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				assetID, err := ensureAsset(ctx, db, objectStore, client, env, baseURL, job)
				if err != nil && !job.Required {
					fmt.Printf("warning: skipped unavailable %s image %d: %v\n", job.Purpose, job.LegacyID, err)
					err = nil
				}
				resultCh <- imageResult{MapKey: job.MapKey, AssetID: assetID, Err: err}
			}
		}()
	}
	go func() {
		defer close(jobCh)
		for _, job := range jobs {
			jobCh <- job
		}
	}()
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	assetIDs := make(map[string]int64, len(jobs))
	completed := 0
	for result := range resultCh {
		if result.Err != nil {
			return nil, result.Err
		}
		if result.AssetID > 0 {
			assetIDs[result.MapKey] = result.AssetID
		}
		completed++
		if completed%10 == 0 || completed == len(jobs) {
			fmt.Printf("images %d/%d\n", completed, len(jobs))
		}
	}
	return assetIDs, nil
}

func ensureAsset(ctx context.Context, db *appdb.DB, objectStore asset.ObjectStore, client *http.Client, env string, baseURL *url.URL, job imageJob) (int64, error) {
	ext := strings.ToLower(path.Ext(job.RelativeURL))
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	objectKey := fmt.Sprintf("inwardclub/%s/%s/v1/%d%s", cleanPathPart(env), job.Purpose, job.LegacyID, ext)
	var existingID int64
	var existingStatus string
	err := db.QueryRowContext(ctx, `SELECT id, status FROM assets WHERE object_key = ?`, objectKey).Scan(&existingID, &existingStatus)
	if err == nil && (existingStatus == "uploaded" || existingStatus == "bound") {
		return existingID, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lookup asset %s: %w", objectKey, err)
	}

	imageURL, err := baseURL.Parse(strings.TrimLeft(job.RelativeURL, "/"))
	if err != nil {
		return 0, fmt.Errorf("resolve image %q: %w", job.RelativeURL, err)
	}
	body, contentType, err := downloadImage(ctx, client, imageURL.String())
	if err != nil {
		return 0, fmt.Errorf("download %s %d: %w", job.Purpose, job.LegacyID, err)
	}
	uploaded, err := objectStore.UploadPublicObject(ctx, asset.PublicUploadInput{
		ObjectKey: objectKey, Reader: bytes.NewReader(body), SizeBytes: int64(len(body)), ContentType: contentType,
	})
	if err != nil {
		return 0, fmt.Errorf("upload %s %d: %w", job.Purpose, job.LegacyID, err)
	}
	filename := path.Base(job.RelativeURL)
	_, err = db.ExecContext(ctx, `INSERT INTO assets
		(bucket, object_key, etag, original_filename, content_type, size_bytes, purpose,
		 visibility, status, uploaded_by_type, uploaded_by_id, created_at, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'public', 'bound', 'super_admin', 1, NOW(), NOW())
		ON DUPLICATE KEY UPDATE etag = VALUES(etag), original_filename = VALUES(original_filename),
		 content_type = VALUES(content_type), size_bytes = VALUES(size_bytes), purpose = VALUES(purpose),
		 status = 'bound', uploaded_at = NOW(), deleted_at = NULL`,
		objectStore.Bucket(), objectKey, uploaded.Hash, filename, contentType, len(body), job.Purpose)
	if err != nil {
		return 0, fmt.Errorf("record asset %s: %w", objectKey, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT id FROM assets WHERE object_key = ?`, objectKey).Scan(&existingID); err != nil {
		return 0, fmt.Errorf("reload asset %s: %w", objectKey, err)
	}
	return existingID, nil
}

func downloadImage(ctx context.Context, client *http.Client, imageURL string) ([]byte, string, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
		if err != nil {
			return nil, "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		if len(body) == 0 || len(body) > maxImageBytes {
			return nil, "", fmt.Errorf("invalid image size %d", len(body))
		}
		contentType := http.DetectContentType(body)
		switch contentType {
		case "image/jpeg", "image/png", "image/webp":
			return body, contentType, nil
		default:
			return nil, "", fmt.Errorf("unexpected content type %s", contentType)
		}
	}
	return nil, "", lastErr
}

func upsertCatalog(ctx context.Context, db *appdb.DB, storeID int64, categories []legacyCategory, products []legacyProduct, assetIDs map[string]int64) error {
	return db.WithinTx(ctx, func(tx *sql.Tx) error {
		categoryByOldID := make(map[int64]legacyCategory, len(categories))
		for _, category := range categories {
			categoryByOldID[category.ID] = category
		}
		categoryIDs := make(map[int64]int64, len(categories))
		visiting := make(map[int64]bool, len(categories))
		var ensureCategory func(int64) (int64, error)
		ensureCategory = func(oldID int64) (int64, error) {
			if id := categoryIDs[oldID]; id > 0 {
				return id, nil
			}
			if visiting[oldID] {
				return 0, fmt.Errorf("category parent cycle at %d", oldID)
			}
			category, ok := categoryByOldID[oldID]
			if !ok {
				return 0, fmt.Errorf("missing category %d", oldID)
			}
			visiting[oldID] = true
			var parentID *int64
			if category.ParentID != nil {
				resolved, err := ensureCategory(*category.ParentID)
				if err != nil {
					return 0, err
				}
				parentID = &resolved
			}
			assetID := nullableAssetID(assetIDs, imageMapKey("category", category.ID))
			id, err := upsertCategory(ctx, tx, storeID, category, parentID, assetID)
			if err != nil {
				return 0, err
			}
			categoryIDs[oldID] = id
			visiting[oldID] = false
			return id, nil
		}
		sortedCategories := append([]legacyCategory(nil), categories...)
		sort.Slice(sortedCategories, func(i, j int) bool { return sortedCategories[i].ID < sortedCategories[j].ID })
		for _, category := range sortedCategories {
			if _, err := ensureCategory(category.ID); err != nil {
				return err
			}
		}
		for _, product := range products {
			categoryID := categoryIDs[product.CategoryID]
			assetID := nullableAssetID(assetIDs, imageMapKey("product", product.ID))
			if err := upsertProduct(ctx, tx, storeID, categoryID, product, assetID); err != nil {
				return err
			}
		}
		return nil
	})
}

func upsertCategory(ctx context.Context, tx *sql.Tx, storeID int64, category legacyCategory, parentID, assetID *int64) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM catalog_categories
		WHERE scope_type = 'store' AND store_id = ? AND name = ? ORDER BY id ASC LIMIT 1`, storeID, category.Name).Scan(&id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("find category %d: %w", category.ID, err)
	}
	status := "active"
	if category.Status != "active" {
		status = "inactive"
	}
	createdAt, updatedAt := migrationTimes(category.CreatedAt, category.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		res, err := tx.ExecContext(ctx, `INSERT INTO catalog_categories
			(scope_type, store_id, parent_id, name, asset_id, sort_order, status, created_at, updated_at)
			VALUES ('store', ?, ?, ?, ?, ?, ?, ?, ?)`, storeID, parentID, category.Name, assetID,
			category.SortOrder, status, createdAt, updatedAt)
		if err != nil {
			return 0, fmt.Errorf("insert category %d: %w", category.ID, err)
		}
		id, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	_, err = tx.ExecContext(ctx, `UPDATE catalog_categories SET parent_id = ?, asset_id = ?,
		sort_order = ?, status = ?, updated_at = ? WHERE id = ?`, parentID, assetID, category.SortOrder, status, updatedAt, id)
	if err != nil {
		return 0, fmt.Errorf("update category %d: %w", category.ID, err)
	}
	return id, nil
}

func upsertProduct(ctx context.Context, tx *sql.Tx, storeID, categoryID int64, product legacyProduct, assetID *int64) error {
	itemType := "food"
	if product.Type == 2 {
		itemType = "coupon"
	}
	status := "draft"
	if product.Status == "active" {
		status = "published"
	}
	createdAt, updatedAt := migrationTimes(product.CreatedAt, product.UpdatedAt)
	const channels = `["wechat","coin"]`
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM catalog_items
		WHERE scope_type = 'store' AND store_id = ? AND source_item_id = ? ORDER BY id ASC LIMIT 1`, storeID, product.ID).Scan(&id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("find product %d: %w", product.ID, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `INSERT INTO catalog_items
			(scope_type, store_id, source_item_id, category_id, name, description, asset_id, item_type,
			 price_cent, stock_quantity, pay_channels, points_reward, sort_order, status, created_at, updated_at)
			VALUES ('store', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			storeID, product.ID, categoryID, product.Name, product.Description, assetID, itemType,
			product.PriceCent, product.Stock, channels, product.Points, product.SortOrder, status, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert product %d: %w", product.ID, err)
		}
		return nil
	}
	_, err = tx.ExecContext(ctx, `UPDATE catalog_items SET category_id = ?, name = ?, description = ?,
		asset_id = ?, item_type = ?, price_cent = ?, stock_quantity = ?, pay_channels = ?,
		points_reward = ?, sort_order = ?, status = ?, updated_at = ? WHERE id = ?`,
		categoryID, product.Name, product.Description, assetID, itemType, product.PriceCent,
		product.Stock, channels, product.Points, product.SortOrder, status, updatedAt, id)
	if err != nil {
		return fmt.Errorf("update product %d: %w", product.ID, err)
	}
	return nil
}

func verifyCatalog(ctx context.Context, db *appdb.DB, storeID int64, categories []legacyCategory, products []legacyProduct) error {
	categoryNames := make(map[int64]string, len(categories))
	for _, category := range categories {
		categoryNames[category.ID] = category.Name
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_categories
			WHERE scope_type = 'store' AND store_id = ? AND name = ?`, storeID, category.Name).Scan(&count); err != nil {
			return fmt.Errorf("verify category %d: %w", category.ID, err)
		}
		if count != 1 {
			return fmt.Errorf("verify category %d: expected one row, got %d", category.ID, count)
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT i.source_item_id, COALESCE(c.name,''), i.asset_id,
		COALESCE(a.object_key,''), COALESCE(a.status,'')
		FROM catalog_items i
		LEFT JOIN catalog_categories c ON c.id = i.category_id
		LEFT JOIN assets a ON a.id = i.asset_id
		WHERE i.scope_type = 'store' AND i.store_id = ? AND i.source_item_id IS NOT NULL`, storeID)
	if err != nil {
		return fmt.Errorf("verify products: %w", err)
	}
	defer rows.Close()
	type itemCheck struct {
		CategoryName string
		AssetID      *int64
		ObjectKey    string
		AssetStatus  string
	}
	checks := make(map[int64]itemCheck, len(products))
	for rows.Next() {
		var sourceID int64
		var check itemCheck
		if err := rows.Scan(&sourceID, &check.CategoryName, &check.AssetID, &check.ObjectKey, &check.AssetStatus); err != nil {
			return fmt.Errorf("scan product verification: %w", err)
		}
		if _, duplicate := checks[sourceID]; duplicate {
			return fmt.Errorf("verify product %d: duplicate source_item_id", sourceID)
		}
		checks[sourceID] = check
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("verify products: %w", err)
	}
	for _, product := range products {
		check, ok := checks[product.ID]
		if !ok {
			return fmt.Errorf("verify product %d: row missing", product.ID)
		}
		if check.CategoryName != categoryNames[product.CategoryID] {
			return fmt.Errorf("verify product %d: category %q, want %q", product.ID, check.CategoryName, categoryNames[product.CategoryID])
		}
		if product.Image != "" {
			if check.AssetID == nil || check.ObjectKey == "" || check.AssetStatus != "bound" {
				return fmt.Errorf("verify product %d: image asset incomplete", product.ID)
			}
			if strings.Contains(check.ObjectKey, "://") || strings.HasPrefix(check.ObjectKey, "/") {
				return fmt.Errorf("verify product %d: object key is not relative: %q", product.ID, check.ObjectKey)
			}
		}
	}
	fmt.Printf("verified database: categories=%d products=%d relative_image_paths=%d\n", len(categories), len(products), len(products))
	return nil
}

func migrationTimes(createdAt, updatedAt string) (string, string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	if createdAt == "" {
		createdAt = now
	}
	if updatedAt == "" {
		updatedAt = createdAt
	}
	return createdAt, updatedAt
}

func nullableAssetID(assetIDs map[string]int64, key string) *int64 {
	id := assetIDs[key]
	if id <= 0 {
		return nil
	}
	return &id
}

func imageMapKey(purpose string, id int64) string {
	return purpose + ":" + strconv.FormatInt(id, 10)
}

func cleanPathPart(value string) string {
	value = strings.Trim(strings.ToLower(value), "/ ")
	if value == "" {
		return "development"
	}
	return strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(value)
}
