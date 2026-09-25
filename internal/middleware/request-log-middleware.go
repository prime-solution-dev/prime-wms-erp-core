package middleware

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"prime-erp-core/internal/requestcontext"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prime-solution-dev/prime-service-x/servicelog"
)

const (
	logBodyLimit = 64 * 1024
	logTimeout   = 2 * time.Second

	// defaultServiceLogServiceName ใช้เมื่อไม่ได้ตั้ง API_LOG_SERVICE ไว้ (เช่น .env ที่ยังไม่ได้
	// เติมค่าจริงจาก Vault) ต่างจาก reference ที่ hardcode ชื่อ service ไว้ตรงๆ เพราะที่นี่ให้
	// ตั้งค่าผ่าน ENV ได้ แล้วมีค่า default ไว้กันพลาดเท่านั้น
	defaultServiceLogServiceName = "prime-erp-core"

	// apiLogExcludeEnv ไม่ได้เป็นส่วนหนึ่งของ servicelog.LoadFromEnv() (library ไม่รู้จักตัวนี้)
	// erp-core เป็นคนอ่านเองเพื่อกันไม่ให้ cron endpoint (ยิงทุกนาที) เขียนลง MongoDB รัวๆ
	apiLogExcludeEnv = "API_LOG_EXCLUDE"
)

// RequestLogMiddleware บันทึก request/response ทุกเส้นที่ไม่ถูก API_LOG_EXCLUDE กันไว้ ลง
// service log (MongoDB ผ่าน prime-service-x/servicelog)
//
// ต้องลงทะเบียนหลัง AuthMiddleware (ดู RegisterMiddlewares) เพื่อให้ inbound log เห็น user ที่
// แกะจาก JWT แล้วจริงๆ ไม่ใช่ค่าว่างเหมือน reference (reference วาง middleware นี้ไว้ก่อน Auth
// แล้วคอมเมนต์ยอมรับเองว่า user ยังว่างอยู่ตอนนั้น)
//
// ส่วน request id / trace id ต้องมาก่อน enabled/excluded check เสมอ เพราะทุก route (รวมเส้นที่
// AuthMiddleware ปฏิเสธ) ต้องมี trace id ติดตัวไปด้วย ไม่ใช่แค่เส้นที่ถูกบันทึก log
func RequestLogMiddleware() gin.HandlerFunc {

	// =========================================================
	// INITIALIZE SERVICE LOG
	// ทำครั้งเดียวตอน Register Middleware
	// =========================================================

	cfg := servicelog.LoadFromEnv()

	if cfg.ServiceName == "" {
		cfg.ServiceName = defaultServiceLogServiceName
	}

	enabled := cfg.Enabled

	if err := utils.InitServiceLog(cfg); err != nil {
		log.Printf(
			"[SERVICE LOG] DISABLED: %v",
			err,
		)

		enabled = false
	}

	excludedPrefixes := parseAPILogExcludePrefixes(
		os.Getenv(apiLogExcludeEnv),
	)

	return func(c *gin.Context) {

		// =========================================================
		// STEP 1: START TIME
		// ใช้สำหรับคำนวณ Duration ของ Request
		// =========================================================

		startTime := time.Now()

		// =========================================================
		// STEP 2: REQUEST ID
		//
		// Request ID:
		// สร้างใหม่ทุก Request
		// ใช้ระบุ Request ปัจจุบันของ Service นี้
		// =========================================================

		requestID := uuid.NewString()

		c.Set(
			"request_id",
			requestID,
		)

		c.Header(
			"X-Request-ID",
			requestID,
		)

		// =========================================================
		// STEP 3: TRACE ID
		//
		// ถ้า Service ก่อนหน้าส่ง X-Trace-ID มา
		// → ใช้ Trace ID เดิม
		//
		// ถ้าไม่มี
		// → ออก Trace ID ใหม่ (เดิมจุดนี้อยู่ใน utils.buildContext ซึ่งครอบคลุมแค่ route ที่ผ่าน
		// ProcessContextRequest — ย้ายมาที่นี่เพื่อให้ทุก route มี trace id รวมถึงเส้นที่
		// AuthMiddleware ปฏิเสธด้วย utils.buildContext ยังคงอ่านค่าที่มีอยู่แล้วใน context ต่อไป
		// ไม่ทับ — เป็นตาข่ายรองเหมือนที่ทำกับ user/token)
		// =========================================================

		traceID := strings.TrimSpace(
			c.GetHeader(utils.TraceIDHeader),
		)

		if traceID == "" {
			traceID = uuid.NewString()
		}

		// =========================================================
		// STEP 4: ADD TRACE ID TO CONTEXT
		//
		// Business Service และ External Service
		// สามารถอ่าน Trace ID ผ่าน:
		//
		// requestcontext.GetTraceID(ctx)
		// =========================================================

		ctx := requestcontext.WithTraceID(
			c.Request.Context(),
			traceID,
		)

		c.Request = c.Request.WithContext(
			ctx,
		)

		// =========================================================
		// SERVICE LOG DISABLED หรือ PATH ถูกยกเว้น (API_LOG_EXCLUDE)
		//
		// ถึงแม้ปิด Log หรือ path ถูกยกเว้น
		// Request ID / Trace ID ยังคงถูกส่งผ่าน Context ตามปกติ
		// =========================================================

		if !enabled || isExcludedPath(c.Request.URL.Path, excludedPrefixes) {
			c.Next()
			return
		}

		// =========================================================
		// STEP 5: READ REQUEST BODY
		//
		// อ่านเฉพาะสำเนาสำหรับ Logging
		// จำกัดขนาดไม่เกิน 64 KiB
		//
		// Handler ด้านหลังยังสามารถอ่าน Body เดิมได้
		// =========================================================

		var requestBody interface{}

		if c.Request.Body != nil {

			originalBody := c.Request.Body

			reader := bufio.NewReaderSize(
				originalBody,
				logBodyLimit+1,
			)

			data, err := reader.Peek(
				logBodyLimit + 1,
			)

			// คืน Reader กลับเข้า Request
			// เพื่อให้ Handler อ่าน Body ต่อได้
			c.Request.Body = &logRequestBody{
				Reader: reader,
				Closer: originalBody,
			}

			if err != nil && err != io.EOF {
				requestBody = "[body omitted: read error]"
			} else {
				requestBody = safeLogBody(data)
			}
		}

		// =========================================================
		// STEP 6: REQUEST INFORMATION
		// =========================================================

		logID := uuid.NewString()

		serviceName := cfg.ServiceName

		user := requestcontext.GetUserOrDefault(
			c.Request.Context(),
		)

		scheme := "http"

		if c.Request.TLS != nil {
			scheme = "https"
		}

		baseURL := scheme +
			"://" +
			c.Request.Host

		endpoint := c.Request.URL.Path
		method := c.Request.Method

		// =========================================================
		// STEP 7: CREATE INBOUND LOG
		//
		// Request เข้ามายัง Service
		//
		// Status เริ่มต้น:
		// PROCESS
		//
		// หมายเหตุ: RequestLogMiddleware ลงทะเบียนหลัง AuthMiddleware (ดู RegisterMiddlewares)
		// user ตรงนี้จึงเป็น user จริงของคนเรียก ไม่ใช่ค่าว่างเหมือน reference
		// =========================================================

		logCtx, cancel := context.WithTimeout(
			context.WithoutCancel(
				c.Request.Context(),
			),
			logTimeout,
		)

		utils.CreateInboundLog(
			logCtx,
			utils.InboundLog{
				ID:          logID,
				TraceID:     traceID,
				RequestID:   requestID,
				Service:     serviceName,
				BaseURL:     baseURL,
				Endpoint:    endpoint,
				Method:      method,
				RequestBody: requestBody,
				User:        user,
			},
		)

		cancel()

		// =========================================================
		// STEP 8: RESPONSE WRITER
		//
		// ครอบ Gin ResponseWriter
		// เพื่อเก็บสำเนา Response Body สำหรับ Logging
		// =========================================================

		writer := &logResponseWriter{
			ResponseWriter: c.Writer,
		}

		c.Writer = writer

		// =========================================================
		// STEP 9: PANIC HANDLING
		//
		// ถ้า Handler / Service panic
		// ให้บันทึก Outbound Log เป็น 500 ก่อน
		//
		// จากนั้น panic ต่อ
		// เพื่อให้ Gin Recovery Middleware จัดการ
		// =========================================================

		defer func() {

			panicValue := recover()

			if panicValue != nil {

				writeOutboundLog(
					c,
					requestID,
					startTime,
					writer,
					http.StatusInternalServerError,
				)

				panic(panicValue)
			}
		}()

		// =========================================================
		// STEP 10: NEXT
		//
		// ส่ง Request ไป Handler ถัดไป
		// =========================================================

		c.Next()

		// =========================================================
		// STEP 11: CREATE OUTBOUND LOG
		//
		// Handler ทำงานเสร็จแล้ว
		// นำ Response / HTTP Status / Duration
		// ไป Update Log เดิมด้วย Request ID
		// =========================================================

		writeOutboundLog(
			c,
			requestID,
			startTime,
			writer,
			writer.Status(),
		)
	}
}

// =========================================================
// API_LOG_EXCLUDE
//
// รายการ path prefix (คั่นด้วย comma) ที่ไม่ต้องบันทึก log เช่น "/cronjob/,/health"
// erp-core เป็นคนเพิ่มเอง — ไม่มีใน servicelog library และ reference ก็ไม่ได้ทำไว้
// =========================================================
func parseAPILogExcludePrefixes(raw string) []string {

	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")

	prefixes := make([]string, 0, len(parts))

	for _, part := range parts {

		prefix := strings.TrimSpace(part)

		if prefix == "" {
			continue
		}

		prefixes = append(prefixes, prefix)
	}

	return prefixes
}

func isExcludedPath(path string, prefixes []string) bool {

	for _, prefix := range prefixes {

		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// =========================================================
// OUTBOUND LOG
//
// ใช้สำหรับ Update Log หลัง Request ทำงานเสร็จ
//
// SUCCESS:
// HTTP Status < 400
//
// ERROR:
// HTTP Status >= 400
// =========================================================
func writeOutboundLog(
	c *gin.Context,
	requestID string,
	startTime time.Time,
	writer *logResponseWriter,
	httpStatus int,
) {

	// =========================================================
	// RESPONSE BODY
	// =========================================================

	responseBody := safeLogBody(
		writer.body.Bytes(),
	)

	// =========================================================
	// DURATION
	// =========================================================

	durationMs := time.Since(
		startTime,
	).Milliseconds()

	// =========================================================
	// STATUS
	// =========================================================

	status := "SUCCESS"
	errorMessage := ""

	if httpStatus >= http.StatusBadRequest {

		status = "ERROR"

		// ไม่เก็บ Error จริง
		// เพื่อป้องกันข้อมูล Sensitive ลง Log
		errorMessage = http.StatusText(
			httpStatus,
		)
	}

	// =========================================================
	// LOG CONTEXT
	//
	// แยก Context สำหรับ Logging
	// ไม่ให้ Request cancellation ยกเลิกการเขียน Log
	// =========================================================

	ctx, cancel := context.WithTimeout(
		context.WithoutCancel(
			c.Request.Context(),
		),
		logTimeout,
	)

	defer cancel()

	// =========================================================
	// CREATE OUTBOUND LOG
	// =========================================================

	utils.CreateOutboundLog(
		ctx,
		utils.OutboundLog{
			RequestID:  requestID,
			Response:   responseBody,
			HTTPStatus: httpStatus,
			Status:     status,
			Error:      errorMessage,
			DurationMs: durationMs,
		},
	)
}

// =========================================================
// REQUEST BODY WRAPPER
//
// ทำให้ Middleware อ่าน Request Body ได้
// โดย Handler ด้านหลังยังสามารถอ่าน Body ต่อได้
// =========================================================
type logRequestBody struct {
	io.Reader
	io.Closer
}

// =========================================================
// RESPONSE WRITER
//
// ครอบ gin.ResponseWriter
// เพื่อเก็บสำเนา Response Body สำหรับ Logging
// =========================================================
type logResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

// =========================================================
// WRITE RESPONSE
// =========================================================
func (w *logResponseWriter) Write(
	data []byte,
) (int, error) {

	// ส่ง Response จริงไป Client
	n, err := w.ResponseWriter.Write(
		data,
	)

	// =========================================================
	// เก็บสำเนาสำหรับ Log
	//
	// เก็บสูงสุด logBodyLimit + 1
	// เพื่อให้รู้ว่า Response ใหญ่เกิน Limit หรือไม่
	// =========================================================

	remaining := logBodyLimit +
		1 -
		w.body.Len()

	if remaining > 0 {

		w.body.Write(
			data[:min(n, remaining)],
		)
	}

	return n, err
}

// =========================================================
// WRITE STRING RESPONSE
// =========================================================
func (w *logResponseWriter) WriteString(
	data string,
) (int, error) {

	return w.Write(
		[]byte(data),
	)
}

// =========================================================
// SAFE LOG BODY
//
// ตรวจสอบ Body ก่อนนำลง MongoDB
//
// - Empty       → nil
// - > 64 KiB    → ไม่เก็บ
// - Non JSON    → ไม่เก็บ
// - JSON        → Parse เป็น Object
// - Sensitive   → Redact
// =========================================================
func safeLogBody(
	data []byte,
) interface{} {

	if len(data) == 0 {
		return nil
	}

	// =========================================================
	// BODY TOO LARGE
	// =========================================================

	if len(data) > logBodyLimit {
		return "[body omitted: exceeds 64 KiB]"
	}

	// =========================================================
	// VALIDATE JSON
	// =========================================================

	if !json.Valid(data) {
		return "[body omitted: non-JSON]"
	}

	// =========================================================
	// DECODE JSON
	// =========================================================

	var value interface{}

	decoder := json.NewDecoder(
		bytes.NewReader(data),
	)

	decoder.UseNumber()

	if err := decoder.Decode(&value); err != nil {
		return "[body omitted: non-JSON]"
	}

	// =========================================================
	// REDACT SENSITIVE DATA
	// =========================================================

	redactLogValue(
		value,
	)

	return value
}

// =========================================================
// REDACT LOG VALUE
//
// ป้องกันข้อมูล Sensitive ลง MongoDB
//
// ตัวอย่าง:
// password
// token
// secret
// authorization
// api_key
// cookie
// set-cookie
// =========================================================
func redactLogValue(
	value interface{},
) {

	switch value := value.(type) {

	// =========================================================
	// JSON OBJECT
	// =========================================================

	case map[string]interface{}:

		for key, child := range value {

			normalized := strings.NewReplacer(
				"_", "",
				"-", "",
				" ", "",
			).Replace(
				strings.ToLower(key),
			)

			if isSensitiveLogKey(normalized) {

				value[key] = "[REDACTED]"

				continue
			}

			redactLogValue(
				child,
			)
		}

	// =========================================================
	// JSON ARRAY
	// =========================================================

	case []interface{}:

		for _, child := range value {

			redactLogValue(
				child,
			)
		}
	}
}

// =========================================================
// SENSITIVE LOG KEY
// =========================================================
func isSensitiveLogKey(
	key string,
) bool {

	return strings.Contains(key, "password") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "secret") ||
		key == "authorization" ||
		key == "apikey" ||
		key == "cookie" ||
		key == "setcookie"
}
