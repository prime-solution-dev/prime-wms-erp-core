package middleware

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
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
)

func RequestLogMiddleware() gin.HandlerFunc {
	log.Println("[ ====================== SERVICE LOG DEBUG] RequestLogMiddleware REGISTERED")
	cfg := servicelog.LoadFromEnv()

	cfg.ServiceName = "wms-service-warehouse"
	log.Printf(
		"[ ====================== SERVICE LOG DEBUG] CONFIG enabled=%t service=%s database=%s collection=%s",
		cfg.Enabled,
		cfg.ServiceName,
		cfg.DatabaseName,
		cfg.CollectionName,
	)

	enabled := cfg.Enabled
	log.Println("[====================== SERVICE LOG DEBUG] MongoDB Init START")

	if err := servicelog.Init(cfg); err != nil {
		log.Printf(
			"[SERVICE LOG DEBUG] MongoDB Init ERROR: %v",
			err,
		)

		enabled = false
	} else {
		log.Println("[SERVICE LOG DEBUG] MongoDB Init SUCCESS")

	}

	return func(c *gin.Context) {

		log.Printf(
			"[SERVICE LOG DEBUG] Middleware ENTER method=%s path=%s enabled=%t",
			c.Request.Method,
			c.Request.URL.Path,
			enabled,
		)

		startTime := time.Now()

		requestID := uuid.NewString()

		c.Set(
			"request_id",
			requestID,
		)

		c.Header(
			"X-Request-ID",
			requestID,
		)

		traceID := c.GetHeader(
			"X-Trace-ID",
		)

		if traceID == "" {
			traceID = requestID
		}

		ctx := requestcontext.WithTraceID(
			c.Request.Context(),
			traceID,
		)

		c.Request = c.Request.WithContext(
			ctx,
		)

		if !enabled {

			log.Printf(
				"[=================== SERVICE LOG DEBUG] SKIP MongoDB logging because enabled=false path=%s",
				c.Request.URL.Path,
			)

			c.Next()
			return
		}

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

		user, _ := requestcontext.GetUser(
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

		log.Printf(
			"[================================= SERVICE LOG] CreateInboundLog UTILS:",
		)

		logCtx, cancel := context.WithTimeout(
			context.WithoutCancel(
				c.Request.Context(),
			),
			logTimeout,
		)

		log.Printf(
			"[SERVICE LOG DEBUG] CreateInboundLog START requestID=%s traceID=%s endpoint=%s",
			requestID,
			traceID,
			endpoint,
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
		// ส่ง Request ไป Middleware / Handler ถัดไป
		//
		// ตัวอย่าง:
		//
		// Auth Middleware
		//      ↓
		// Handler
		//      ↓
		// ProcessContextGinRequest
		//      ↓
		// Business Service
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
