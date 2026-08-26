package coupon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/joho/godotenv"

	platdb "github.com/inwardclub/server/internal/platform/db"
	"github.com/inwardclub/server/internal/platform/httpx"
)

func TestCouponCategoriesMigrationAndReadPathsIntegration(t *testing.T) {
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

	miniRepo := NewRepository(database)
	categories, err := miniRepo.ListActiveCategories(ctx)
	if err != nil {
		t.Fatalf("list active categories: %v", err)
	}
	if len(categories) != 7 {
		t.Fatalf("active category count = %d, want 7", len(categories))
	}

	consoleRepo := NewConsoleRepository(database)
	categoryRepo, ok := consoleRepo.(CategoryRepository)
	if !ok {
		t.Fatal("console repository does not implement category management")
	}
	created, err := categoryRepo.CreateCategory(ctx, CategoryInput{
		Name: fmt.Sprintf("集成测试酒水券-%d", time.Now().UnixNano()), BusinessType: TypeAlcohol,
		SortOrder: 999, Status: CategoryStatusActive,
	})
	if err != nil {
		t.Fatalf("create a second category for the same business type: %v", err)
	}
	defer database.ExecContext(ctx, `DELETE FROM coupon_categories WHERE id = ?`, created.ID)
	if created.BusinessType != TypeAlcohol {
		t.Fatalf("created category business type = %q", created.BusinessType)
	}

	templates, _, err := consoleRepo.ListTemplates(ctx, ConsoleScope{}, httpx.Page{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	for _, template := range templates {
		if template.CategoryID <= 0 || template.CategoryName == "" || !validCouponType(template.CouponType) {
			t.Fatalf("template was not backfilled correctly: %+v", template)
		}
	}
}
