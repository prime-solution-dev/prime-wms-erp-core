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

type UpdateRunningSystemConfigRequest struct {
	ConfigCode string `json:"config_code"`
	Count      int    `json:"count"`
}

type UpdateRunningSystemConfigResponse struct {
	ConfigCode     string `json:"config_code"`
	CurrentRunning int    `json:"current_running"`
	Message        string `json:"message"`
}

func UpdateRunningSystemConfig(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req UpdateRunningSystemConfigRequest

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

	// Parse JSON from config
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
		// Reset running number for new year/month
		configJSON.Year = currentYear
		configJSON.Month = currentMonth
		configJSON.CurrentRunning = req.Count
	} else {
		// Update current_running by adding count
		configJSON.CurrentRunning += req.Count
	}

	// Update database with WHERE condition
	updatedJSON, err := json.Marshal(configJSON)
	if err != nil {
		return nil, errors.New("failed to marshal updated config JSON: " + err.Error())
	}

	if err := gormx.Model(&systemConfig).Where("config_code = ?", req.ConfigCode).Update("json", string(updatedJSON)).Error; err != nil {
		return nil, errors.New("failed to update config: " + err.Error())
	}

	response := UpdateRunningSystemConfigResponse{
		ConfigCode:     req.ConfigCode,
		CurrentRunning: configJSON.CurrentRunning,
		Message:        fmt.Sprintf("Successfully updated current_running to %d", configJSON.CurrentRunning),
	}

	return response, nil
}
func UpdateRunningSystemConfigInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req UpdateRunningSystemConfigRequest

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

	// Parse JSON from config
	var configJSON models.RunningConfigJSON
	if err := json.Unmarshal([]byte(systemConfig.JSON), &configJSON); err != nil {
		return nil, errors.New("failed to parse config JSON: " + err.Error())
	}

	// Get current year and month
	now := time.Now()
	currentMonth := now.Format("01") // MM format (11 for November)
	currentYear := now.Year() + 543  // Full year format (2025)
	if req.ConfigCode == "RUNNING_AP" {
		currentYear = now.Year()
	}
	shortYearBE := fmt.Sprintf("%02d", currentYear%100)

	// Check if year and month match current and reset if needed
	if configJSON.Year != shortYearBE || configJSON.Month != currentMonth {
		// Reset running number for new year/month
		configJSON.Year = shortYearBE
		configJSON.Month = currentMonth
		configJSON.CurrentRunning = req.Count
	} else {
		// Update current_running by adding count
		configJSON.CurrentRunning += req.Count
	}

	// Update database with WHERE condition
	updatedJSON, err := json.Marshal(configJSON)
	if err != nil {
		return nil, errors.New("failed to marshal updated config JSON: " + err.Error())
	}

	if err := gormx.Model(&systemConfig).Where("config_code = ?", req.ConfigCode).Update("json", string(updatedJSON)).Error; err != nil {
		return nil, errors.New("failed to update config: " + err.Error())
	}

	response := UpdateRunningSystemConfigResponse{
		ConfigCode:     req.ConfigCode,
		CurrentRunning: configJSON.CurrentRunning,
		Message:        fmt.Sprintf("Successfully updated current_running to %d", configJSON.CurrentRunning),
	}

	return response, nil
}
