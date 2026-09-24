package utils

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/requestcontext"

	"github.com/gin-gonic/gin"
)

func TestProcessContextRequestMultipartPassesFileAndUser(t *testing.T) {
	gotFileName := ""
	gotForm := ""
	gotUser := ""

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(requestcontext.WithUser(c.Request.Context(), "somchai"))
		c.Next()
	})
	router.POST("/upload", func(c *gin.Context) {
		ProcessContextRequestMultipart(c, func(ctx context.Context, input MultipartInput) (interface{}, error) {
			gotUser = requestcontext.GetUserOrDefault(ctx)
			if files := input.Files["file"]; len(files) > 0 {
				gotFileName = files[0].Filename
			}
			if values := input.Form["company_code"]; len(values) > 0 {
				gotForm = values[0]
			}

			return gin.H{"ok": true}, nil
		})
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "pricelist.xlsx")
	part.Write([]byte("dummy"))
	writer.WriteField("company_code", "CM")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	if gotFileName != "pricelist.xlsx" {
		t.Fatalf("filename = %q", gotFileName)
	}

	if gotForm != "CM" {
		t.Fatalf("form company_code = %q", gotForm)
	}

	if gotUser != "somchai" {
		t.Fatalf("user = %q", gotUser)
	}
}
