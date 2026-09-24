# Context user propagation — 5 repos

วันที่: 2026-09-24
Branch: `refactor/context-user-propagation` (ทุก repo แตกจาก `Develop`)

## ปัญหา

เวลาคนกดปุ่มบนเว็บ ข้อมูลว่า "ใครกด" ติดมากับ request ในรูป JWT ที่ header `Authorization`
service ตัวแรกรู้ว่าใครกด แต่เมื่อมันยิงต่อไป service ถัดไป ไม่ได้ส่ง token ต่อไปด้วย
service ปลายทางจึงไม่รู้ว่าใครกด แล้วเขียน `create_by` / `update_by` เป็นค่าคงที่ เช่น `SYSTEM`

```
เว็บ ──(token สมชาย)──▶ picking ──(ไม่มี token)──▶ outbound
                        create_by = สมชาย          create_by = SYSTEM   ← ผิด
```

## สิ่งที่ต้องมีครบ 3 อย่าง จึงจะแก้ปัญหาได้

1. **middleware เก็บ user และ token ลง `c.Request.Context()`**
   ของเดิมเก็บด้วย `c.Set("user", ...)` ซึ่งอยู่ใน map ของ gin คนละที่กับ `c.Request.Context()`
   service ที่รับ `context.Context` จึงมองไม่เห็น (เกิดขึ้นจริงแล้วที่ `wms-product-service` บน Develop:
   `UpsertUnit` ตอบ `user not found in context` เพราะ route ใช้ `ProcessContextRequest` แต่ middleware ยังเป็นตัวเก่า)

2. **ตอนยิงออกต้องแปะ token เป็น header**
   `context.Context` เป็นตัวแปรใน process ไม่ได้วิ่งข้าม HTTP
   `http.NewRequestWithContext(ctx, ...)` ส่งไปแค่ deadline/cancel ไม่ได้ส่งค่าใน context
   มีแต่ header ที่ข้ามไปถึง Kong → auth → service ปลายทาง

3. **service ต้องอ่าน user จาก context** ไม่ใช่ `ctx.GetString("user")` หรือค่าคงที่

## ขอบเขต

ทำ 5 repo เท่านั้น repo อื่นมีคนอื่นทำ

| repo | โฟลเดอร์ในเครื่อง | module |
|---|---|---|
| crossmax-custom | `C:\work-prime\crossmax-custom` | `crossmax-custom` |
| prime-wms-erp-core | `C:\work-prime\erp-core` | `prime-erp-core` |
| prime-wms-outbound-core | `C:\work-prime\wms-outbound-service` | `wms-service-outbound` |
| prime-wms-packing-core | `C:\work-prime\wms-pack-service` | `wms-service-packing` |
| prime-wms-picking-core | `C:\work-prime\wms-picking-service` | `wms-service-picking` |

**ไม่อยู่ในขอบเขต:** service ที่เหลือใน repo ที่ไม่ได้ระบุ, การเปลี่ยน path/request/response ของ API,
การตรวจสอบลายเซ็น JWT (ยังใช้ `ParseUnverified` เหมือนเดิม เพราะ Kong/auth ตรวจให้แล้ว)

## สภาพปัจจุบันของแต่ละ repo

| repo | AuthMiddleware | service ที่รับ `*gin.Context` | `ctx.JSON` ใน service | `http.NewRequest` ไม่มี ctx | route ที่ต้องเปลี่ยน |
|---|---|---|---|---|---|
| crossmax-custom | ตัวเปล่า (no-op), ไม่มี jwt ใน go.mod | 1 | 0 | 1 | 3 (อีก 28 ใช้ `ProcessRequestV2` อยู่แล้ว) |
| erp-core | มีแล้ว (แก้เป็นตัวใหม่แล้วใน pilot) | 120 | 21 | 21 (เหลือหลัง pilot จาก 35) | 100 (แปลงแล้ว 2) |
| wms-outbound-service | ตัวเปล่า (no-op), ไม่มี jwt ใน go.mod | 24 | 37 | 29 | 22 |
| wms-pack-service | มีแล้ว (เก็บด้วย `c.Set`) | 30 | 43 | 23 | 28 |
| wms-picking-service | มีแล้ว (เก็บด้วย `c.Set`) | 33 | 34 | 35 | 30 |

crossmax ส่ง `ctx` ครบเกือบทุกจุดแล้ว แต่ client แปะ `SERVICE_TOKEN` จาก env แทน token ของคนกด
จึงยังส่ง user ไม่ได้

## สถาปัตยกรรม

### ไฟล์กลาง (เหมือนกันทุก repo)

```
internal/requestcontext/context.go     WithUser/GetUser/GetUserOrDefault
                                       WithToken/GetToken, WithTraceID/GetTraceID
internal/middleware/set-user-context.go AuthMiddleware แกะ JWT แล้วเก็บลง c.Request.Context()
internal/apperr/apperr.go              AppError{HTTPStatus,Code,Message} + BadRequest/NotFound/Conflict/Unauthorized
internal/utils/request-handler-gin.go  ProcessContextRequest (+ ProcessContextRequestMultipart เฉพาะ erp-core)
internal/utils/http-client.go          NewRequest(ctx, method, url, body)
```

`internal/requestcontext/context.go` ต้องตรงกับตัวที่เจ้าของ pattern (ทีม product-service) ใช้ทุกบรรทัด
ส่วนที่เราเพิ่มคือ `DefaultUser` กับ `GetUserOrDefault` เท่านั้น

### การไหลของข้อมูล

```
เว็บ
 │ Authorization: Bearer <jwt ของสมชาย>
 ▼
AuthMiddleware ─ แกะ JWT ─ c.Set("user")                 (ให้ service เดิมที่ยังเป็น gin)
                         └ WithUser(ctx, "somchai")      (ให้ service ที่แปลงแล้ว)
                         └ WithToken(ctx, authHeader)    (ไว้ส่งต่อ)
 ▼
ProcessContextRequest ─ อ่าน body ─▶ service(ctx, payload)
 │                                      │
 │                                      ├ requestcontext.GetUserOrDefault(ctx) → create_by/update_by
 │                                      └ utils.NewRequest(ctx, ...) ─ แปะ Authorization ─▶ Kong ─▶ service ปลายทาง
 ▼
ตอบ: AppError → status ตามที่กำหนด / TypedError → body ของ error นั้น / อื่นๆ → 500
```

### กติกาที่ต้องทำเหมือนกันทุกที่

1. ทุกจุดที่ยิงออกใช้ `utils.NewRequest(ctx, ...)` ห้ามใช้ `http.NewRequest` ตรงๆ
2. `utils.NewRequest` ไม่มี fallback ไป `SERVICE_TOKEN` เส้นที่ต้องยิงในนามระบบ
   ต้องใส่ token ลง context เองที่จุดเริ่มงาน
3. ทุกจุดที่เขียน `create_by` / `update_by` ใช้ `requestcontext.GetUserOrDefault(ctx)` ไม่ใส่ค่าคงที่
4. **เส้นที่ทำงานหลัง `tx.Commit()` หรือที่ต้องทำต่อแม้ caller ตัดสาย ให้ใช้ `context.WithoutCancel(ctx)`**
   ไม่งั้นพอ caller หมดเวลา งานหลัง commit จะตายเงียบ (เจอจริงที่ `closeSalesOfDeliveries` ใน erp-core)
5. `ctx.JSON(4xx, ...)` ใน service เปลี่ยนเป็น `return nil, apperr.BadRequest(...)` ตาม status เดิม
6. service ที่ยังไม่แปลง เรียกฟังก์ชันที่ต้องการ ctx ได้โดยส่ง `ctx.Request.Context()` ต่อไป
   (ใช้ร่วมกันได้ระหว่างที่ยังแปลงไม่ครบ)

### ค่า default เมื่อไม่มี user

`GetUserOrDefault` คืน **ค่าว่าง** (`""`) พร้อม log `[WARN]` หนึ่งบรรทัด
ไม่ใส่ชื่อสมมติแทนคน เพราะ "ไม่รู้ว่าใคร" ต้องอ่านออกจากข้อมูลได้ตรงๆ
งานที่ไม่มีคนกดแต่รู้ตัวตน (cron) ต้องใส่ชื่อของตัวเองลง context เช่น `"CRON"`
เพื่อให้แยกออกจากค่าว่างที่แปลว่า "ไม่รู้ว่าใคร"

## เคสพิเศษต่อ repo

- **crossmax-custom**: ต้องเพิ่ม `github.com/golang-jwt/jwt/v5` ใน go.mod และสร้าง `internal/models` ที่มี `AuthenJWTClaims`
  เปลี่ยน client ทั้ง 14 ตัวจาก `getServiceToken()` เป็น token จาก ctx
  เปลี่ยนชื่อ `ProcessRequestV2` เป็น `ProcessContextRequest` เพื่อให้ตรงกับ repo อื่น
- **wms-outbound-service**: ต้องเพิ่ม jwt ใน go.mod และสร้าง `AuthenJWTClaims` เหมือนกัน
- **wms-pack-service / wms-picking-service**: มี `ProcessRequestTyped` ที่คืน body ของ error เอง (9 route รวม create picking / confirm pack)
  `ProcessContextRequest` ต้องรองรับ `TypedError` ด้วย ไม่งั้นหน้าเว็บที่อ่าน body นั้นจะพัง
  picking มี `internal/utils/context_user.go` (`GetUserFromCtx`) ที่ต้องเลิกใช้
- **erp-core**: มี 2 route ที่รับไฟล์ ใช้ `ProcessContextRequestMultipart`
  อีก 3 route ที่เดิมใช้ `ProcessRequestWithBinding` เป็น JSON ใช้ `ProcessContextRequest` ได้
  cron `wms-kernal` ต้องใส่ `WithUser(ctx, "CRON")` + `WithToken` จาก `SERVICE_TOKEN`

## ผลข้างเคียงที่ยอมรับ

1. `update_by` ของเส้นที่ไม่มี token เปลี่ยนจาก `system` / `SYSTEM` เป็น **ค่าว่าง** ทุก repo
   แถวที่เขียนโดยไม่รู้ว่าใครกด จะมองเห็นได้ทันทีจากข้อมูล ไม่ถูกกลบด้วยชื่อสมมติ
2. error บางตัวที่เคยตอบ 500 จะตอบ 400 ตามความหมายที่ถูกต้อง
3. จะมี log `[WARN] no user in context` ทุกครั้งที่ถูกเรียกโดยไม่มี token เช่น hook ระหว่าง service
   ใช้เป็นตัวชี้ว่าเส้นไหนยังไม่ได้ส่ง token ต่อ

## การทดสอบ

ต่อ repo:
1. unit test ของไฟล์กลาง (แบบที่ทำไว้แล้วใน erp-core)
   - middleware ใส่ user/token ลง ctx จริง ผ่าน route จำลองพร้อม JWT
   - `ProcessContextRequest` map status ของ `AppError` และรับ user จาก middleware ตัวเก่าได้ด้วย
   - `utils.NewRequest` แปะ Authorization จาก ctx และไม่แปะเมื่อไม่มี token
   - external client 1 ตัวยิงไป `httptest` server จริง แล้วยืนยันว่า server ได้ header
2. `go build ./...` + `go vet` + test เดิมของ repo ผ่านเท่าเดิม
3. grep ยืนยันว่าไม่เหลือ `http.NewRequest(`, `ctx.GetString("user")`, และค่าคงที่ใน create_by
4. e2e: กดจากเว็บ 1 เส้นที่ข้าม service แล้วเช็ค `create_by` ทั้ง 2 ฝั่งเป็นชื่อคนกด พร้อมแคปหน้าจอ

## สิ่งที่ยังค้าง

- `SERVICE_TOKEN` ยังไม่มีในไฟล์ env ของ erp-core ต้องให้คนดูแลระบบใส่ก่อนขึ้นจริง ไม่งั้น cron ยิงออกแบบไม่มี token
- middleware ตัวที่เจ้าของ pattern เขียนยังไม่ final ถ้าเค้าแก้ ให้ copy ทับไฟล์เดียว
- ยังไม่ได้เสนอ `GetUserOrDefault` ให้เจ้าของ pattern ใส่ในตัวกลาง
