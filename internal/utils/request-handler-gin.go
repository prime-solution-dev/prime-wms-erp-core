package utils

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"prime-erp-core/internal/apperr"
	"prime-erp-core/internal/requestcontext"

	"github.com/gin-gonic/gin"
)

// buildContext ย้ายข้อมูลจากฝั่ง gin มาใส่ context ก่อนส่งให้ service
//
// middleware ตัวใหม่ใส่ user/token ลง c.Request.Context() ให้อยู่แล้ว ส่วนที่เติมตรงนี้
// เป็นตาข่ายรองสำหรับกรณีที่ middleware ยังเป็นตัวเก่า (เก็บด้วย c.Set อย่างเดียว)
// หรือ token มากับ request แต่ middleware แกะ JWT ไม่ผ่าน
func buildContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()

	if _, ok := requestcontext.GetUser(ctx); !ok {
		if user := c.GetString("user"); user != "" {
			ctx = requestcontext.WithUser(ctx, user)
		}
	}

	if _, ok := requestcontext.GetToken(ctx); !ok {
		if token := c.GetHeader("Authorization"); token != "" {
			ctx = requestcontext.WithToken(ctx, token)
		}
	}

	return ctx
}

// writeError ตอบ error ตาม status ที่ error พกมา
func writeError(c *gin.Context, err error) {
	// service เขียน response ไปเองแล้ว ไม่ต้องเขียนซ้ำ
	if c.Writer.Written() {
		return
	}

	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, gin.H{"code": appErr.Code, "error": appErr.Message})
		return
	}

	// price-service ยังคืน *BindingError จาก 3 endpoint ที่เดิมใช้ ProcessRequestWithBinding
	// (ตอนนี้ validate เองด้วย validator.New().SetTagName("binding") แทน ctx.ShouldBindJSON)
	// รูป response ต้องเหมือนเดิมทุก byte ไม่งั้นหน้าเว็บที่อ่าน body["details"] จะพัง
	var bindingErr *BindingError
	if errors.As(err, &bindingErr) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": bindingErr.Message,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// ProcessContextRequest อ่าน JSON payload จาก body แล้วส่งต่อให้ service ที่รับ context.Context
//
// ใช้แทน ProcessRequest ในทุก route ที่ service ถูกแปลงเป็น context.Context แล้ว
func ProcessContextRequest(
	c *gin.Context,
	serviceFunc func(context.Context, string) (interface{}, error),
) {
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := serviceFunc(buildContext(c), string(jsonData))
	if err != nil {
		writeError(c, err)
		return
	}

	if c.Writer.Written() {
		return
	}

	c.JSON(http.StatusOK, response)
}

// MultipartInput คือไฟล์และ form ที่แยกออกมาจาก request แล้ว
// service จึงไม่ต้องรู้จัก gin
type MultipartInput struct {
	Files map[string][]*multipart.FileHeader
	Form  map[string][]string
}

// ProcessContextRequestMultipart ใช้กับ route ที่รับไฟล์ เช่น upload pricelist
func ProcessContextRequestMultipart(
	c *gin.Context,
	serviceFunc func(context.Context, MultipartInput) (interface{}, error),
) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get multipart form: " + err.Error()})
		return
	}

	input := MultipartInput{
		Files: form.File,
		Form:  form.Value,
	}

	response, err := serviceFunc(buildContext(c), input)
	if err != nil {
		writeError(c, err)
		return
	}

	if c.Writer.Written() {
		return
	}

	c.JSON(http.StatusOK, response)
}
