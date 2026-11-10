package middleware

import "github.com/gin-gonic/gin"

// ลำดับนี้ตั้งใจ: CORS ก่อน (ตัด preflight OPTIONS ทิ้งไม่ให้ไหลลง log) ตามด้วย Auth (แกะ JWT
// ใส่ user/token ลง context) แล้วค่อยเป็น RequestLogMiddleware ตัวสุดท้าย ต่างจาก reference
// (wms-warehouse-service) ที่วาง log middleware ไว้เป็นตัวแรกสุด — reference เองก็คอมเมนต์ยอมรับ
// ว่า user ยังว่างอยู่ตอนนั้น ที่นี่ต้องให้ Auth ทำงานก่อน เพื่อให้ inbound log บันทึก user จริงของ
// คนเรียก ไม่ใช่ค่าว่าง (AuthMiddleware ไม่เคย abort ทุก route จึงยังได้ log เสมอ)
func RegisterMiddlewares(ctx *gin.Engine) {
	ctx.Use(CORSMiddleware())
	ctx.Use(AuthMiddleware())
	ctx.Use(RequestLogMiddleware())
}
