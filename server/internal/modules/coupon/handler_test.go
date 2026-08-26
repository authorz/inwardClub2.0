package coupon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListCategoriesReturnsManagedCategoryEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/mini/coupon-categories", nil)
	repo := &memRepo{categories: []CouponCategory{{ID: 3, Name: "酒水福利", BusinessType: TypeAlcohol, Status: CategoryStatusActive}}}

	NewHandler(NewService(repo)).ListCategories(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Data []CouponCategoryView `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].ID != 3 || response.Data[0].Name != "酒水福利" {
		t.Fatalf("unexpected coupon categories: %+v", response.Data)
	}
}
