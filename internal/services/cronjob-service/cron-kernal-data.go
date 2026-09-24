package CronjobService

import (
	"context"
	"os"
	"strings"
	"sync"

	"prime-erp-core/internal/cronjob"
	"prime-erp-core/internal/requestcontext"
)

// CronUser คือชื่อที่ถูกเขียนลง create_by / update_by ของงานที่ cron เป็นคนสั่ง
// ไม่ใช่ค่า mock แต่เป็นการบอกว่าแถวนี้ไม่ได้เกิดจากคนกด
const CronUser = "CRON"

func init() {
	cronjob.RegisterJob("wms-kernal", GetKernalFromCron, "*/1 * * * *")
}

// cronContext สร้าง context ของงานที่ไม่มีคนกด
//
// ต้องประกาศตัวตนที่นี่จุดเดียว เพราะ utils.NewRequest ไม่มี fallback ให้แล้ว
// เส้นไหนไม่ใส่ token ลง context ก็จะยิงออกไปแบบไม่มี Authorization
func cronContext() context.Context {
	ctx := requestcontext.WithUser(context.Background(), CronUser)

	if serviceToken := strings.TrimSpace(os.Getenv("SERVICE_TOKEN")); serviceToken != "" {
		if !strings.HasPrefix(serviceToken, "Bearer ") {
			serviceToken = "Bearer " + serviceToken
		}

		ctx = requestcontext.WithToken(ctx, serviceToken)
	}

	return ctx
}

// GetKernalFromCron คือ entry point ของ cron ที่ RegisterJob เรียก (รับ func() ไม่มี argument)
func GetKernalFromCron() {
	GetKernal(cronContext())
}

// GetKernalManual เป็นเส้นที่ยิงจากหน้าจอ ใช้ user และ token ของคนกดตามปกติ
func GetKernalManual(ctx context.Context, jsonPayload string) (interface{}, error) {
	GetKernal(ctx)

	return nil, nil
}

func GetKernal(ctx context.Context) {
	println("start kernal service")

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		CreditRequestEffectiveDtmPending(ctx)
	}()

	go func() {
		defer wg.Done()
		CreditRequestEffectiveDtm(ctx)
	}()

	go func() {
		defer wg.Done()
		CreditExtra(ctx)
	}()

	// เดิมไม่ได้ wait ทำให้ฟังก์ชันจบก่อนงานทั้งสามเสร็จ
	// ตอนนี้ทั้งสามใช้ context ร่วมกัน ถ้าปล่อยไว้แบบเดิม ctx ของเส้นที่ยิงจากหน้าจอ
	// จะถูกยกเลิกตั้งแต่ตอบ response ไปแล้ว งานที่ยังค้างอยู่จะถูกตัดกลางคัน
	wg.Wait()
}
