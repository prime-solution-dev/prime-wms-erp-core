// Package apperr เป็น error ที่พก HTTP status มาด้วย
//
// service ที่รับ context.Context เรียก ctx.JSON เองไม่ได้แล้ว จึงต้องคืน error ตัวนี้แทน
// เพื่อให้ตัวจัดการ request ตอบ 400/404 ได้เหมือนเดิม ไม่กลายเป็น 500 ทั้งหมด
package apperr

import "net/http"

// AppError represents a structured application error with HTTP context.
type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

// New creates a new AppError.
func New(httpStatus int, code string, message string) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

// BadRequest คืน 400 ใช้แทน ctx.JSON(http.StatusBadRequest, ...) ที่เดิมอยู่ใน service
func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, "BAD_REQUEST", message)
}

// NotFound คืน 404
func NotFound(message string) *AppError {
	return New(http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict คืน 409 ใช้กับเคสที่ข้อมูลชนกัน เช่นเอกสารถูกยกเลิกไปแล้ว
func Conflict(message string) *AppError {
	return New(http.StatusConflict, "CONFLICT", message)
}

// Unauthorized คืน 401
func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message)
}
