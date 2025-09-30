package utils

import (
	"io"
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
