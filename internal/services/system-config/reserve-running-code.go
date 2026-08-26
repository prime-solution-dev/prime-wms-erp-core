package systemConfigService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prime-erp-core/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RunningPeriod คือปี/เดือนที่ใช้ทั้งประกอบเป็นเลขที่เอกสารและใช้ตัดสินว่าต้องรีเซ็ต running หรือยัง
// แยกออกมาเพราะเอกสารคนละกลุ่มใช้ปีคนละแบบ (ดู StandardRunningPeriod / InvoiceRunningPeriod)
type RunningPeriod struct {
	Year  string
	Month string
}

// StandardRunningPeriod ใช้ปี ค.ศ. เต็ม เช่น 202608 — ใช้กับ quotation, sale, delivery, purchase
func StandardRunningPeriod() RunningPeriod {
	now := time.Now()

	return RunningPeriod{
		Year:  now.Format("2006"),
		Month: now.Format("01"),
	}
}

// InvoiceRunningPeriod ใช้ปี พ.ศ. 2 หลัก ยกเว้น RUNNING_AP ที่ใช้ ค.ศ.
func InvoiceRunningPeriod(configCode string) RunningPeriod {
	now := time.Now()

	year := now.Year() + 543
	if configCode == "RUNNING_AP" {
		year = now.Year()
	}

	return RunningPeriod{
		Year:  fmt.Sprintf("%02d", year%100),
		Month: now.Format("01"),
	}
}

// ReserveRunningCodes จองเลขที่เอกสาร count ตัวแบบ atomic แล้วเดิน current_running ให้ในทีเดียว
//
// เดิมเป็น GetRunningSystemConfig (SELECT เฉยๆ ไม่มี lock) แล้วค่อยเรียก
// UpdateRunningSystemConfig ทีหลัง คนละ connection คนละ transaction และเรียก "หลัง" insert เอกสาร
// สองคนกดพร้อมกันจึงอ่าน current_running ค่าเดียวกันแล้วได้เลขซ้ำ ซึ่งเลขนั้นคือ
// delivery_code ที่ WMS ใช้เป็น document_ref ในการ join กลับ
//
// ที่นี่ล็อกแถว config ด้วย SELECT ... FOR UPDATE จนกว่าจะเขียน current_running เสร็จ
// คนที่สองจะรอแล้วได้เลขถัดไปเสมอ
//
// แลกมาด้วยการที่เลขถูกใช้ไปแล้วจริงตั้งแต่ตอนจอง ถ้าเอกสารสร้างไม่สำเร็จจะเกิดเลขกระโดด
// ซึ่งยอมรับได้ ดีกว่าเลขซ้ำ
func ReserveRunningCodes(gormx *gorm.DB, configCode string, count int, prefix string, period RunningPeriod) ([]string, error) {
	if configCode == "" {
		return nil, errors.New("config_code is required")
	}

	if count <= 0 {
		return nil, errors.New("count must be greater than 0")
	}

	codes := []string{}

	err := gormx.Transaction(func(tx *gorm.DB) error {
		var systemConfig models.SystemConfig

		// Take ไม่ใช่ First เพราะ system_config ไม่มี primary key ให้ GORM เรียงตาม
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("config_code = ?", configCode).
			Take(&systemConfig).Error; err != nil {
			return fmt.Errorf("config not found: %s", configCode)
		}

		var configJSON models.RunningConfigJSON
		if err := json.Unmarshal([]byte(systemConfig.JSON), &configJSON); err != nil {
			return fmt.Errorf("failed to parse config JSON: %v", err)
		}

		// ขึ้นเดือน/ปีใหม่ให้เริ่มนับหนึ่งใหม่
		if configJSON.Year != period.Year || configJSON.Month != period.Month {
			configJSON.Year = period.Year
			configJSON.Month = period.Month
			configJSON.CurrentRunning = 0
		}

		if prefix != "" {
			configJSON.Prefix = prefix
		}

		codes = buildRunningCodes(configJSON, count)
		configJSON.CurrentRunning += count

		updatedJSON, err := json.Marshal(configJSON)
		if err != nil {
			return fmt.Errorf("failed to marshal updated config JSON: %v", err)
		}

		return tx.Model(&models.SystemConfig{}).
			Where("config_code = ?", configCode).
			Update("json", string(updatedJSON)).Error
	})
	if err != nil {
		return nil, err
	}

	return codes, nil
}

// buildRunningCodes สร้างเลขรูปแบบ <prefix><year><month>-<running ที่ pad แล้ว>
func buildRunningCodes(configJSON models.RunningConfigJSON, count int) []string {
	codes := make([]string, 0, count)
	startRunning := configJSON.CurrentRunning + 1

	for i := 0; i < count; i++ {
		codes = append(codes, fmt.Sprintf("%s%s%s-%s",
			configJSON.Prefix,
			configJSON.Year,
			configJSON.Month,
			fmt.Sprintf("%0*d", configJSON.RunningDigit, startRunning+i),
		))
	}

	return codes
}
