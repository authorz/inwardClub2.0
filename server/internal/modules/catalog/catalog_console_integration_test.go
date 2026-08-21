package catalog

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	platdb "github.com/inwardclub/server/internal/platform/db"
	"github.com/joho/godotenv"
)

func TestSQLConsoleRepository_BatchDeleteOrphanStoreCatalogIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_INTEGRATION") != "1" {
		t.Skip("set RUN_MYSQL_INTEGRATION=1 to run")
	}

	ctx := context.Background()
	_, sourceFile, _, _ := runtime.Caller(0)
	env, err := godotenv.Read(filepath.Join(filepath.Dir(sourceFile), "../../../.env"))
	if err != nil {
		t.Fatalf("load .env: %v", err)
	}
	database, err := platdb.Open(ctx, env["MYSQL_DSN"])
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()

	missingStoreID := time.Now().UnixNano()
	name := fmt.Sprintf("批量真删除回归-%d", missingStoreID)
	res, err := database.ExecContext(ctx, `INSERT INTO catalog_categories
		(scope_type, store_id, name, category_type, sort_order, status, created_at, updated_at)
		VALUES ('store', ?, ?, 'product', 0, 'active', NOW(), NOW())`, missingStoreID, name)
	if err != nil {
		t.Fatalf("insert category fixture: %v", err)
	}
	categoryID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read category fixture id: %v", err)
	}
	defer database.ExecContext(ctx, `DELETE FROM catalog_categories WHERE id = ?`, categoryID)

	res, err = database.ExecContext(ctx, `INSERT INTO catalog_items
		(scope_type, store_id, category_id, name, item_type, price_cent, stock_quantity,
		 pay_channels, points_reward, sort_order, status, created_at, updated_at)
		VALUES ('store', ?, ?, ?, 'food', 100, 1, JSON_ARRAY('wechat'), 0, 0, 'active', NOW(), NOW())`,
		missingStoreID, categoryID, name)
	if err != nil {
		t.Fatalf("insert item fixture: %v", err)
	}
	itemID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read item fixture id: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, `DELETE FROM store_item_overrides WHERE item_id = ?`, itemID)
		_, _ = database.ExecContext(ctx, `DELETE FROM catalog_variants WHERE item_id = ?`, itemID)
		_, _ = database.ExecContext(ctx, `DELETE FROM catalog_items WHERE id = ?`, itemID)
	}()

	if _, err := database.ExecContext(ctx, `INSERT INTO catalog_variants
		(item_id, sku_code, name, price_cent, stock_quantity, status, created_at, updated_at)
		VALUES (?, ?, '默认', 100, 1, 'active', NOW(), NOW())`, itemID, fmt.Sprintf("DELETE-%d", itemID)); err != nil {
		t.Fatalf("insert variant fixture: %v", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO store_item_overrides
		(store_id, item_id, category_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'active', NOW(), NOW())`, missingStoreID, itemID, categoryID); err != nil {
		t.Fatalf("insert override fixture: %v", err)
	}

	handler := NewConsoleHandler(NewConsoleService(NewConsoleRepository(database)))
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/api/v2/admin/catalog/items/batch-delete", handler.BatchDeleteItems)
	request := httptest.NewRequest(http.MethodPost, "/api/v2/admin/catalog/items/batch-delete",
		bytes.NewBufferString(fmt.Sprintf(`{"ids":[%d]}`, itemID)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("delete orphan-store item: status=%d body=%s", response.Code, response.Body.String())
	}
	assertCatalogRowCount(t, database, "catalog_items", "id", itemID, 0)
	assertCatalogRowCount(t, database, "catalog_variants", "item_id", itemID, 0)
	assertCatalogRowCount(t, database, "store_item_overrides", "item_id", itemID, 0)

	router.POST("/api/v2/admin/catalog/categories/batch-delete", handler.BatchDeleteCategories)
	request = httptest.NewRequest(http.MethodPost, "/api/v2/admin/catalog/categories/batch-delete",
		bytes.NewBufferString(fmt.Sprintf(`{"ids":[%d]}`, categoryID)))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("delete orphan-store category: status=%d body=%s", response.Code, response.Body.String())
	}
	assertCatalogRowCount(t, database, "catalog_categories", "id", categoryID, 0)
}

func assertCatalogRowCount(t *testing.T, database *platdb.DB, table, column string, id int64, want int) {
	t.Helper()
	var got int
	if err := database.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM `+table+` WHERE `+column+` = ?`, id).Scan(&got); err != nil {
		t.Fatalf("count %s fixture rows: %v", table, err)
	}
	if got != want {
		t.Fatalf("unexpected %s fixture row count: got %d want %d", table, got, want)
	}
}
