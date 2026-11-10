package utils

import (
	"context"
	"log"

	"github.com/prime-solution-dev/prime-service-x/servicelog"
)

// =========================================================
// INBOUND LOG
// Request ที่เข้ามายัง Service
// =========================================================
type InboundLog struct {
	ID          string
	TraceID     string
	RequestID   string
	Service     string
	BaseURL     string
	Endpoint    string
	Method      string
	RequestBody interface{}
	User        string
}

func CreateInboundLog(
	ctx context.Context,
	data InboundLog,
) {
	logEntry := servicelog.ServiceLog{
		ID:        data.ID,
		TraceID:   data.TraceID,
		RequestID: data.RequestID,
		Service:   data.Service,
		BaseURL:   data.BaseURL,
		Endpoint:  data.Endpoint,
		Method:    data.Method,
		Request:   data.RequestBody,
		User:      data.User,
	}

	if err := activeServiceLogSink.Create(ctx, logEntry); err != nil {
		log.Printf(
			"[SERVICE LOG] CREATE INBOUND ERROR: %v",
			err,
		)
		return
	}

	log.Printf(
		"[SERVICE LOG] CREATE INBOUND SUCCESS requestID=%s",
		data.RequestID,
	)
}

// =========================================================
// OUTBOUND LOG
// Response ที่ออกจาก Service
// =========================================================
type OutboundLog struct {
	RequestID  string
	Response   interface{}
	HTTPStatus int
	Status     string
	Error      string
	DurationMs int64
}

func CreateOutboundLog(
	ctx context.Context,
	data OutboundLog,
) {
	if err := activeServiceLogSink.Update(
		ctx,
		data.RequestID,
		data.Response,
		data.HTTPStatus,
		data.Status,
		data.Error,
		data.DurationMs,
	); err != nil {
		log.Printf(
			"[SERVICE LOG] CREATE OUTBOUND ERROR: %v",
			err,
		)
		return
	}

	log.Printf(
		"[SERVICE LOG] CREATE OUTBOUND SUCCESS requestID=%s status=%s",
		data.RequestID,
		data.Status,
	)
}

// =========================================================
// SEAM สำหรับเทส
//
// servicelog.Init/Create/Update เป็นฟังก์ชัน package-level ที่ต่อ MongoDB จริง ต่อให้ปิด
// logging (Enabled=false) Init ก็ยังถูกเรียกเสมอตอน RequestLogMiddleware() สร้างตัวเอง เทส
// จึงต้องสลับปลายทางได้โดยไม่พึ่ง MongoDB จริง — ทำที่ชั้นนี้ (wrapper ของเรา) เท่านั้น
// ห้ามแก้ library เพื่อเทส
// =========================================================

// ServiceLogSink คือส่วนของ servicelog ที่ไฟล์นี้เรียกใช้จริง แยกเป็น interface เพื่อให้เทส
// (ของ package นี้เองและของ internal/middleware) ส่งของปลอมเข้ามาแทนของจริงได้
type ServiceLogSink interface {
	Init(cfg servicelog.Config) error
	Create(ctx context.Context, entry servicelog.ServiceLog) error
	Update(ctx context.Context, requestID string, response interface{}, httpStatus int, status string, errMessage string, durationMs int64) error
}

type realServiceLogSink struct{}

func (realServiceLogSink) Init(cfg servicelog.Config) error {
	return servicelog.Init(cfg)
}

func (realServiceLogSink) Create(ctx context.Context, entry servicelog.ServiceLog) error {
	return servicelog.Create(ctx, entry)
}

func (realServiceLogSink) Update(
	ctx context.Context,
	requestID string,
	response interface{},
	httpStatus int,
	status string,
	errMessage string,
	durationMs int64,
) error {
	return servicelog.Update(ctx, requestID, response, httpStatus, status, errMessage, durationMs)
}

// activeServiceLogSink ปลายทางจริงที่ CreateInboundLog/CreateOutboundLog/InitServiceLog เรียก
// ค่าเริ่มต้นคือของจริง (ต่อ MongoDB ผ่าน prime-service-x/servicelog) เทสค่อยสลับด้วย
// SetServiceLogSinkForTest เป็นของปลอมชั่วคราว
var activeServiceLogSink ServiceLogSink = realServiceLogSink{}

// SetServiceLogSinkForTest สลับปลายทางของ service log เป็นของปลอมสำหรับเทส คืนฟังก์ชันไว้
// เรียก (defer) เพื่อสลับกลับเป็นของจริงเสมอ ไม่ใช่ของ production code — เรียกได้จาก _test.go
// เท่านั้น และไม่ปลอดภัยกับเทสที่รันแบบ parallel เพราะ activeServiceLogSink เป็น package-level var
func SetServiceLogSinkForTest(sink ServiceLogSink) func() {
	previous := activeServiceLogSink
	activeServiceLogSink = sink

	return func() {
		activeServiceLogSink = previous
	}
}

// InitServiceLog สั่ง init ปลายทาง service log (ต่อ MongoDB ถ้าเปิดใช้งาน) ผ่าน
// activeServiceLogSink เดียวกับที่ Create/Update ใช้ เพื่อให้เทสสลับเป็นของปลอมได้จุดเดียว
func InitServiceLog(cfg servicelog.Config) error {
	return activeServiceLogSink.Init(cfg)
}
