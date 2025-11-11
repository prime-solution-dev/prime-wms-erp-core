package models

import "github.com/google/uuid"

type SystemConfig struct {
	TopicCode  string     `json:"topic_code"`
	ConfigCode string     `json:"config_code"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty"`
	ConfigName string     `json:"config_name"`
	Cond1      string     `json:"cond1"`
	Cond2      string     `json:"cond2"`
	Value      string     `json:"value"`
	Sequence   int        `json:"sequence"`
	Remark     string     `json:"remark"`
	JSON       string     `json:"json"`
}

func (SystemConfig) TableName() string {
	return "system_config"
}

// RunningConfigJSON represents the JSON structure for running number configuration
type RunningConfigJSON struct {
	Year           string `json:"year"`
	Month          string `json:"month"`
	Prefix         string `json:"prefix"`
	RunningDigit   int    `json:"running_digit"`
	CurrentRunning int    `json:"current_running"`
}
