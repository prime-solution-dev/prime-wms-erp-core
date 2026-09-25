package utils

import (
	"github.com/prime-solution-dev/prime-service-x/apilog"
)

// =========================================================
// SEAM สำหรับเทส — เหตุผลเดียวกับ ServiceLogSink ใน service-log.go
//
// apilog.Init ต่อ MongoDB จริง (ping ตอน Init) เทสที่ไม่ได้ตั้งใจทดสอบเรื่อง Mongo (เช่น เทส path
// exclude ของ RequestLogMiddleware) ไม่ควรไปรอ/ล้มเพราะ apilog พยายามต่อ URI ปลอมที่เทสตั้งไว้
// สำหรับ servicelog ต้องสลับปลายทางได้โดยไม่พึ่ง MongoDB จริงเหมือนกัน — ทำที่ชั้นนี้เท่านั้น ห้าม
// แก้ library เพื่อเทส
// =========================================================

// APILogInitiator คือส่วนของ apilog ที่ InitAPILog เรียกใช้จริง แยกเป็น interface เพื่อให้เทส
// ส่งของปลอมเข้ามาแทนของจริงได้ (ไม่ต่อ MongoDB)
type APILogInitiator interface {
	Init(cfg apilog.Config) error
}

type realAPILogInitiator struct{}

func (realAPILogInitiator) Init(cfg apilog.Config) error {
	return apilog.Init(cfg)
}

// activeAPILogInitiator ปลายทางจริงที่ InitAPILog เรียก ค่าเริ่มต้นคือของจริง (ต่อ MongoDB ผ่าน
// prime-service-x/apilog) เทสค่อยสลับด้วย SetAPILogInitiatorForTest เป็นของปลอมชั่วคราว
var activeAPILogInitiator APILogInitiator = realAPILogInitiator{}

// SetAPILogInitiatorForTest สลับปลายทางของ apilog init เป็นของปลอมสำหรับเทส คืนฟังก์ชันไว้เรียก
// (defer/t.Cleanup) เพื่อสลับกลับเป็นของจริงเสมอ ไม่ใช่ของ production code — เรียกได้จาก _test.go
// เท่านั้น และไม่ปลอดภัยกับเทสที่รันแบบ parallel เพราะ activeAPILogInitiator เป็น package-level var
func SetAPILogInitiatorForTest(initiator APILogInitiator) func() {
	previous := activeAPILogInitiator
	activeAPILogInitiator = initiator

	return func() {
		activeAPILogInitiator = previous
	}
}

// InitAPILog สั่ง init ปลายทางของ apilog (บันทึก outbound request ที่ NewOutboundLogTransport
// สร้างให้) ผ่าน activeAPILogInitiator เดียวกับที่เทสสลับได้
//
// เรียกจาก RequestLogMiddleware() จุดเดียว ติดกับ utils.InitServiceLog(cfg) เพราะทั้งสองอ่าน
// API_LOG_* ชุดเดียวกันจาก .env/Vault (ServiceName/MongoDBURI/Database) จึงต่อ MongoDB ปลายทาง
// เดียวกันโดยอัตโนมัติ ไม่ต้องตั้งค่าซ้ำ — ผู้เรียกต้อง log แล้วปล่อยผ่านเมื่อ error ไม่ใช่ทำให้
// service เริ่มไม่ได้ (ดู RequestLogMiddleware)
func InitAPILog(cfg apilog.Config) error {
	return activeAPILogInitiator.Init(cfg)
}
