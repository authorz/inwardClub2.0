package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccessLogIncludesResolvedSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&output, nil))
	r := gin.New()
	r.Use(AccessLog(log))
	r.GET("/protected", func(c *gin.Context) {
		c.Set(CtxSubjectType, "pre_member")
		c.Set(CtxSubjectID, int64(27))
		c.Status(http.StatusForbidden)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))
	line := output.String()
	for _, fragment := range []string{`"subject_type":"pre_member"`, `"subject_id":27`} {
		if !strings.Contains(line, fragment) {
			t.Fatalf("access log missing %s: %s", fragment, line)
		}
	}
}
