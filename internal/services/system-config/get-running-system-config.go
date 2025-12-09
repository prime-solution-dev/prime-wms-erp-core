package systemConfigService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

type GetRunningSystemConfigRequest struct {
	ConfigCode string `json:"config_code"`
	Count      int    `json:"count"`
}

type GetRunningSystemConfigResponse struct {
	ConfigCode string   `json:"config_code"`
	Data       []string `json:"data"`
}

func GetRunningSystemConfig(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetRunningSystemConfigRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Validate request
	if req.ConfigCode == "" {
		return nil, errors.New("config_code is required")
	}
	if req.Count <= 0 {
		return nil, errors.New("count must be greater than 0")
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return nil, err
	}
	defer db.CloseGORM(gormx)

	// Get system config
	var systemConfig models.SystemConfig
	if err := gormx.Where("config_code = ?", req.ConfigCode).First(&systemConfig).Error; err != nil {
		return nil, errors.New("config not found: " + req.ConfigCode)
	}

	// Parse JSON from config (from JSON column)
	var configJSON models.RunningConfigJSON
	if err := json.Unmarshal([]byte(systemConfig.JSON), &configJSON); err != nil {
		return nil, errors.New("failed to parse config JSON: " + err.Error())
	}

	// Get current year and month
	now := time.Now()
	currentYear := now.Format("2006") // Full year format (2025)
	currentMonth := now.Format("01")  // MM format (11 for November)

	// Check if year and month match current and reset if needed
	if configJSON.Year != currentYear || configJSON.Month != currentMonth {
		// Need to reset running number for new year/month
		configJSON.Year = currentYear
		configJSON.Month = currentMonth
		configJSON.CurrentRunning = 0
	}

	// Generate running codes
	var generatedCodes []string
	startRunning := configJSON.CurrentRunning + 1

	for i := 0; i < req.Count; i++ {
		runningNumber := startRunning + i

		// Format: PFYYYY-NNNN (เปลี่ยนเป็นปีเต็ม)
		code := fmt.Sprintf("%s%s%s-%s",
			configJSON.Prefix,
			configJSON.Year,
			configJSON.Month,
			fmt.Sprintf("%0*d", configJSON.RunningDigit, runningNumber))

		generatedCodes = append(generatedCodes, code)
	}

	response := GetRunningSystemConfigResponse{
		ConfigCode: req.ConfigCode,
		Data:       generatedCodes,
	}

	return response, nil
}
