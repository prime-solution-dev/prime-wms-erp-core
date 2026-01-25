package utils

import (
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ProcessRequest เป็นฟังก์ชันกลางที่ใช้สำหรับอ่าน JSON payload จาก body และส่งต่อไปยัง service function
func ProcessRequest(c *gin.Context, serviceFunc func(*gin.Context, string) (interface{}, error)) {
	// อ่าน JSON payload จาก body และแปลงเป็น string
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// เรียกใช้ service function ที่ส่งเข้ามา
	response, err := serviceFunc(c, string(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ส่ง response กลับไปยัง client
	c.JSON(http.StatusOK, response)
}

// ProcessRequestWithBinding processes requests using Gin's binding and validation
// This function uses ShouldBindJSON which returns errors that should be handled by the caller
func ProcessRequestWithBinding(c *gin.Context, serviceFunc func(*gin.Context) (interface{}, error)) {
	// Call service function which should use ShouldBindJSON internally
	response, err := serviceFunc(c)
	if err != nil {
		// Check if it's a binding/validation error
		if bindingErr, ok := err.(*BindingError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": bindingErr.Message,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Send response back to client
	c.JSON(http.StatusOK, response)
}

// BindingError represents a binding/validation error
type BindingError struct {
	Message string
}

func (e *BindingError) Error() string {
	return e.Message
}

func ProcessRequestMultiPart(c *gin.Context, serviceFunc func(*gin.Context) (interface{}, error)) {
	form, err := c.MultipartForm()
	if err != nil {
		log.Println("Error parsing multipart form:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get multipart form: " + err.Error()})
		return
	}

	for fieldName, fileHeaders := range form.File {
		for _, fileHeader := range fileHeaders {
			log.Println("Field Name:", fieldName)
			log.Println("Uploaded File:", fileHeader.Filename)
		}
	}

	response, err := serviceFunc(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
