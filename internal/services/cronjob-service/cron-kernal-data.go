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

// runKernal คือ seam สำหรับเทส — เทสสลับตัวนี้เพื่อดักว่า context ที่ส่งเข้า GetKernal
// เป็นตัวไหน โดยไม่ต้องยิง HTTP จริงออกไป (GetKernal ยิง 3 เส้นตาม base_url_erp)
var runKernal = GetKernal

// GetKernalManual เป็นเส้นที่ยิงจากหน้าจอ ใช้ user และ token ของคนกดตามปกติ
//
// ต้องใช้ context.WithoutCancel ไม่งั้นถ้า client/gateway ตัดการเชื่อมต่อกลางทาง
// (เช่น timeout) request context จะถูกยกเลิก แล้ว utils.NewRequest ใน
// credit-request.go / credit-extra.go จะได้ context.Canceled กลับมา ทำให้งาน credit
// ตายกลางทาง — หลุดระหว่าง UpdateCreditRequest กับ CreateCreditTransaction แล้วค้าง
// เป็น state ครึ่งๆ กลางๆ เส้นนี้ยัง block รอ GetKernal จบเหมือนเดิม แค่ทำให้ตัวงานเอง
// ยกเลิกไม่ได้จากฝั่ง caller
func GetKernalManual(ctx context.Context, jsonPayload string) (interface{}, error) {
	runKernal(context.WithoutCancel(ctx))

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
	// route ที่เรียกฟังก์ชันนี้ (เช่น GetKernalManual) ต้องรอผลจริงก่อนตอบ response
	// ถ้าไม่ wait ผู้เรียกจะเห็น response ว่า "จบแล้ว" ทั้งที่งานยังทำอยู่เบื้องหลัง
	wg.Wait()
}
