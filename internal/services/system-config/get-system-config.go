package systemConfigService

import (
	"encoding/json"
	"errors"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"

	"github.com/gin-gonic/gin"
)

type GetSystemConfigRequest struct {
	TopicCodes  []string `json:"topic_codes"`
	ConfigCodes []string `json:"config_codes"`
}

type GetSystemConfigResponse struct {
	SystemConfigs []SystemConfigDto `json:"system_configs"`
}

type SystemConfigDto struct {
	TopicCode  string `json:"topic_code"`
	ConfigCode string `json:"config_code"`
	Value      string `json:"value"`
	JSON       string `json:"json,omitempty"`
}

func GetSystemConfig(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetSystemConfigRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Validate request
	if len(req.TopicCodes) < 1 {
		return nil, errors.New("topic_codes is required")
	}

	// Call repository function
	systemConfigs, err := systemConfigRepository.GetSystemConfig(req.TopicCodes, req.ConfigCodes)
	if err != nil {
		return nil, err
	}

	// Map to response DTOs
	var dtos []SystemConfigDto
	for _, config := range systemConfigs {
		dto := SystemConfigDto{
			TopicCode:  config.TopicCode,
			ConfigCode: config.ConfigCode,
			Value:      config.Value,
			JSON:       config.JSON,
		}
		dtos = append(dtos, dto)
	}

	response := GetSystemConfigResponse{
		SystemConfigs: dtos,
	}

	return response, nil
}
