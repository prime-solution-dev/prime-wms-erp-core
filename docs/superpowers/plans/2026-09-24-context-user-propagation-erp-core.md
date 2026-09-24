# Context User Propagation — erp-core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ให้ทุกเส้นใน erp-core ส่ง user และ token ของคนที่ยิงเข้ามาผ่าน `context.Context` ไปจนถึง service ปลายทาง และเลิกเขียน `create_by` / `update_by` ด้วยค่าคงที่

**Architecture:** middleware เก็บ user + token ลง `c.Request.Context()` → `ProcessContextRequest` ส่ง ctx ให้ service → service อ่าน user ด้วย `requestcontext.GetUserOrDefault(ctx)` และยิงออกด้วย `utils.NewRequest(ctx, ...)` ซึ่งแปะ `Authorization` จาก ctx ให้จุดเดียว

งานแบ่งเป็น 2 เฟสเพื่อให้ build เขียวตลอดทาง: **เฟส A** เติม `ctx` ให้ external client ทุกตัวก่อน (service เดิมที่ยังเป็น gin ส่ง `ctx.Request.Context()` เข้าไปได้) — จบเฟสนี้ token เริ่มไหลข้าม service แล้ว **เฟส B** ค่อยแปลง service ทีละ package

**Tech Stack:** Go 1.24, gin, gorm, `github.com/golang-jwt/jwt/v5`, testing + httptest

**Spec:** `docs/superpowers/specs/2026-09-24-context-user-propagation-design.md`

## Global Constraints

- Go 1.24 (`go.mod`) — `context.WithoutCancel` ใช้ได้
- ห้ามเปลี่ยน path / request / response ของ route ใดๆ (ยกเว้น HTTP status ที่ระบุไว้ในสเปก)
- ห้ามเพิ่ม dependency ใหม่ใน repo นี้ (jwt v5 มีอยู่แล้ว)
- commit message ภาษาอังกฤษเสมอ
- `git add` เฉพาะไฟล์ที่แก้ ห้าม `git add -A`
- ทุกจุดที่ยิงออกใช้ `utils.NewRequest(ctx, ...)` ห้าม `http.NewRequest` ตรงๆ
- ทุกจุดที่เขียน `create_by` / `update_by` ใช้ `requestcontext.GetUserOrDefault(ctx)`
- เส้นที่ทำงานหลัง `tx.Commit()` หรือที่ต้องทำต่อแม้ caller ตัดสาย ใช้ `context.WithoutCancel(ctx)`
- branch: `refactor/context-user-propagation` (มีอยู่แล้ว มี pilot commit ค้างอยู่ใน working tree)

## Review Focus

- **เส้นที่ถูกเรียกโดยไม่มี token** (hook จาก wms-order-service, cron, เครื่องมือภายใน) ต้องได้ `SYSTEM` และทำงานต่อ ไม่ใช่ error — ทดสอบใน Task 13
- **งานหลัง commit ถูกยกเลิกตาม caller** เมื่อ caller timeout: ต้องยังทำงานจนจบด้วย `WithoutCancel` — ทดสอบใน Task 4
- **goroutine ที่ยังทำงานอยู่ตอน handler คืนค่าแล้ว** ctx ถูกยกเลิก งานในนั้นจะตายเงียบ — ทดสอบใน Task 12
- **รูป response ของ error** ที่เปลี่ยนจาก `{"error":...}` เป็น `{"code":...,"error":...}` ต้องยังมี key `error` เดิมอยู่เสมอ — ทดสอบใน Task 9
- **route multipart** ต้องยังอ่านไฟล์และ form ได้ครบหลังเลิกรับ `*gin.Context` — ทดสอบใน Task 9

---

## สถานะเริ่มต้น (pilot ที่ทำไปแล้ว ยังไม่ commit)

ไฟล์กลางสร้างครบแล้ว: `internal/requestcontext/context.go`, `internal/apperr/apperr.go`,
`internal/utils/request-handler-gin.go`, `internal/utils/http-client.go` พร้อม test
middleware แก้แล้ว, `UpdateStatusDelivery` + สายที่เกี่ยวข้องแปลงแล้ว, cron ใส่ `CRON` แล้ว

**Task 0 (ทำก่อนเริ่ม Task 1):** commit ของที่ค้างอยู่

- [ ] **Step 1: ตรวจว่า build และ test ผ่าน**

Run: `go build ./... && go test ./external/... ./internal/utils/... ./internal/services/delivery-service/... ./internal/services/cronjob-service/...`
Expected: BUILD ผ่าน, test ผ่านทั้งหมด (111 passed)

- [ ] **Step 2: Commit ของที่ค้าง**

```bash
git add internal/requestcontext internal/apperr internal/utils/http-client.go internal/utils/http-client_test.go internal/utils/request-handler-gin.go internal/utils/request-handler-gin_test.go internal/middleware/set-user-context.go internal/routes/routes.go internal/services/delivery-service internal/services/cronjob-service external/order-service
git commit -m "feat(context): carry caller user and token through context.Context"
```

---

## เฟส A — external client ทุกตัวรับ ctx

### Task 1: external/order-service — CreateOrder และ UpdateOrderByDelivery

**Files:**
- Modify: `external/order-service/create-order.go`
- Modify: `external/order-service/update-order-by-delivery.go`
- Modify: caller ทุกไฟล์ (หาได้ด้วย grep ใน Step 3)
- Test: `external/order-service/create-order_test.go`

**Interfaces:**
- Consumes: `utils.NewRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error)`, `requestcontext.WithToken(ctx, token)`
- Produces: `CreateOrder(ctx context.Context, jsonPayload CreateOrderRequest) (CreateOrderResponse, error)`, `UpdateOrderByDelivery(ctx context.Context, jsonPayload UpdateOrderByDeliveryRequest) (UpdateOrderByDeliveryResponse, error)`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `external/order-service/create-order_test.go`:

```go
package externalService

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/config"
	"prime-erp-core/internal/requestcontext"
)

func TestCreateOrderForwardsCallerToken(t *testing.T) {
	gotAuth := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer downstream.Close()

	original := config.CREATE_ORDER_ENDPOINT
	config.CREATE_ORDER_ENDPOINT = downstream.URL
	defer func() { config.CREATE_ORDER_ENDPOINT = original }()

	ctx := requestcontext.WithToken(context.Background(), "Bearer token-test")

	if _, err := CreateOrder(ctx, CreateOrderRequest{}); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if gotAuth != "Bearer token-test" {
		t.Fatalf("downstream got Authorization = %q", gotAuth)
	}
}
```

หมายเหตุ: ชื่อ type ของ request/response ให้ยึดตามที่ประกาศจริงใน `create-order.go`

- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**

Run: `go test ./external/order-service/ -run TestCreateOrderForwardsCallerToken`
Expected: FAIL — compile error `not enough arguments in call to CreateOrder`

- [ ] **Step 3: แก้ client 2 ตัว**

ใน `create-order.go` และ `update-order-by-delivery.go` ทำแบบเดียวกับ `cancel-order.go` ที่แปลงไปแล้ว:

```go
func CreateOrder(ctx context.Context, jsonPayload CreateOrderRequest) (CreateOrderResponse, error) {
	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return CreateOrderResponse{}, errors.New("Error marshaling struct to JSON: " + err.Error())
	}

	// utils.NewRequest แปะ token ของคนที่ยิงเข้ามาไปกับ header ให้ ปลายทางจะได้รู้ว่าใครสั่ง
	req, err := utils.NewRequest(ctx, "POST", config.CREATE_ORDER_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return CreateOrderResponse{}, errors.New("Error creating request: " + err.Error())
	}
	// บรรทัด req.Header.Set("Content-Type", ...) เดิม ลบทิ้ง utils.NewRequest ตั้งให้แล้ว
```

เพิ่ม import `"context"` และ `"prime-erp-core/internal/utils"` ในทั้ง 2 ไฟล์

- [ ] **Step 4: แก้ caller ทุกจุด**

หา caller: `grep -rn "\.CreateOrder(\|\.UpdateOrderByDelivery(" internal --include=*.go`

caller ที่เป็น service แบบ `*gin.Context` ให้ส่ง `ctx.Request.Context()`:

```go
res, err := orderExternalService.CreateOrder(ctx.Request.Context(), createOrderRequest)
```

caller ที่เป็นฟังก์ชันช่วยที่ไม่มี ctx ให้เพิ่ม parameter `ctx context.Context` เป็นตัวแรก แล้วไล่ส่งต่อขึ้นไปจนถึง service

- [ ] **Step 5: รันเทสและ build**

Run: `go build ./... && go test ./external/order-service/`
Expected: BUILD ผ่าน, เทสใหม่ PASS

- [ ] **Step 6: Commit**

```bash
git add external/order-service internal
git commit -m "feat(external): pass caller context to order-service client"
```

### Task 2: external/warehouse-service — 3 ฟังก์ชัน

**Files:**
- Modify: `external/warehouse-service/*.go` (3 ไฟล์)
- Modify: caller (GetInventoryWeightByKey มี 4 ไฟล์, GetInventoryATP 1, GetSystemConfigWarehouse 1)
- Test: `external/warehouse-service/get-inventory-atp_test.go`

**Interfaces:**
- Consumes: `utils.NewRequest`
- Produces: `GetInventoryATP(ctx context.Context, ...)`, `GetInventoryWeightByKey(ctx context.Context, ...)`, `GetSystemConfigWarehouse(ctx context.Context, ...)` — argument เดิมทั้งหมดเลื่อนไปต่อท้าย ctx

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

```go
package externalService

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/config"
	"prime-erp-core/internal/requestcontext"
)

func TestGetInventoryATPForwardsCallerToken(t *testing.T) {
	gotAuth := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
	}))
	defer downstream.Close()

	original := config.GET_INVENTORY_ATP_ENDPOINT
	config.GET_INVENTORY_ATP_ENDPOINT = downstream.URL
	defer func() { config.GET_INVENTORY_ATP_ENDPOINT = original }()

	ctx := requestcontext.WithToken(context.Background(), "Bearer token-test")

	if _, err := GetInventoryATP(ctx, GetInventoryATPRequest{}); err != nil {
		t.Fatalf("GetInventoryATP: %v", err)
	}

	if gotAuth != "Bearer token-test" {
		t.Fatalf("downstream got Authorization = %q", gotAuth)
	}
}
```

ชื่อ package และ type ให้ยึดตามที่ประกาศจริงในโฟลเดอร์นั้น

- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**

Run: `go test ./external/warehouse-service/ -run TestGetInventoryATPForwardsCallerToken`
Expected: FAIL — compile error

- [ ] **Step 3: แก้ client ทั้ง 3 ตัว** — รูปแบบเดียวกับ Task 1 Step 3

- [ ] **Step 4: แก้ caller ทุกจุด** — รูปแบบเดียวกับ Task 1 Step 4

- [ ] **Step 5: รันเทสและ build**

Run: `go build ./... && go test ./external/warehouse-service/`
Expected: BUILD ผ่าน, เทสใหม่ PASS

- [ ] **Step 6: Commit**

```bash
git add external/warehouse-service internal
git commit -m "feat(external): pass caller context to warehouse-service client"
```

### Task 3: external ที่เหลือ 4 package

**Files:**
- Modify: `external/customer-service/get-customer.go`, `external/product-service/get-product.go`, `external/pack-service/get-pack-so.go`, `external/goods-receive-service/*.go`
- Modify: caller ของแต่ละฟังก์ชัน (GetCustomer 4 ไฟล์, GetProduct 2, GetPackSo 1, GetGoodsReceives 2, GetInbounds 2)
- Test: `external/customer-service/get-customer_forward_test.go`

**Interfaces:**
- Produces: `GetCustomer(ctx context.Context, ...)`, `GetProduct(ctx context.Context, ...)`, `GetPackSo(ctx context.Context, ...)`, `GetGoodsReceives(ctx context.Context, ...)`, `GetInbounds(ctx context.Context, ...)`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `external/customer-service/get-customer_forward_test.go`:

```go
package externalService

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/config"
	"prime-erp-core/internal/requestcontext"
)

func TestGetCustomerForwardsCallerToken(t *testing.T) {
	gotAuth := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
	}))
	defer downstream.Close()

	original := config.GET_CUSTOMER_MASTER_ENDPOINT
	config.GET_CUSTOMER_MASTER_ENDPOINT = downstream.URL
	defer func() { config.GET_CUSTOMER_MASTER_ENDPOINT = original }()

	ctx := requestcontext.WithToken(context.Background(), "Bearer token-test")

	if _, err := GetCustomer(ctx, GetCustomerRequest{}); err != nil {
		t.Fatalf("GetCustomer: %q", err)
	}

	if gotAuth != "Bearer token-test" {
		t.Fatalf("downstream got Authorization = %q", gotAuth)
	}
}
```

ชื่อ package และ type ของ request ให้ยึดตามที่ประกาศจริงใน `external/customer-service`

- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**

Run: `go test ./external/customer-service/ -run TestGetCustomerForwardsCallerToken`
Expected: FAIL — compile error

- [ ] **Step 3: แก้ client ทั้ง 5 ฟังก์ชัน** — รูปแบบเดียวกับ Task 1 Step 3

- [ ] **Step 4: แก้ caller ทุกจุด** — รูปแบบเดียวกับ Task 1 Step 4

- [ ] **Step 5: ยืนยันว่าไม่เหลือ http.NewRequest ใน external**

Run: `grep -rn "http.NewRequest(" external --include=*.go`
Expected: ไม่มีผลลัพธ์

Run: `go build ./... && go test ./external/...`
Expected: BUILD ผ่าน, test ผ่าน

- [ ] **Step 6: Commit**

```bash
git add external internal
git commit -m "feat(external): pass caller context to remaining service clients"
```

---

## เฟส B — แปลง service ทีละ package

ทุก task ในเฟสนี้ทำ 4 อย่างเหมือนกัน:
1. `func X(ctx *gin.Context, jsonPayload string)` → `func X(ctx context.Context, jsonPayload string)`
2. `ctx.GetString("user")` + fallback → `requestcontext.GetUserOrDefault(ctx)`
3. `ctx.JSON(4xx, ...)` → `return nil, apperr.BadRequest(...)` / `apperr.NotFound(...)` ตาม status เดิม
4. route ของ package นั้นใน `internal/routes/routes.go`: `utils.ProcessRequest` → `utils.ProcessContextRequest`

และที่จุดเรียก external ให้เปลี่ยน `ctx.Request.Context()` ที่ใส่ไว้ในเฟส A กลับเป็น `ctx` ตรงๆ

### Task 4: delivery-service ที่เหลือ

**Files:**
- Modify: `internal/services/delivery-service/get-delivery.go`, `get-delivery-co.go`, `create-delivery.go`, `update-delivery.go`, และไฟล์อื่นใน package ที่ยังรับ `*gin.Context`
- Modify: `internal/routes/routes.go` (route ของ delivery)
- Test: `internal/services/delivery-service/close-sale-context_test.go`

**Interfaces:**
- Consumes: `requestcontext.GetUserOrDefault(ctx) string`, `apperr.BadRequest(message string) *apperr.AppError`
- Produces: service ทุกตัวใน package รับ `context.Context`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน — งานหลัง commit ต้องไม่ถูกยกเลิกตาม caller**

สร้าง `internal/services/delivery-service/close-sale-context_test.go`:

```go
package deliveryService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// เส้นที่ทำงานหลัง commit ต้องเก็บ user/token ไว้ แต่ไม่ตายตาม caller ที่ตัดสายไปแล้ว
func TestPostCommitContextSurvivesCallerCancel(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")
	ctx = requestcontext.WithToken(ctx, "Bearer token-test")

	canceled, cancel := context.WithCancel(ctx)
	cancel()

	postCommit := postCommitContext(canceled)

	if err := postCommit.Err(); err != nil {
		t.Fatalf("post-commit context ถูกยกเลิกไปด้วย: %v", err)
	}

	if user, _ := requestcontext.GetUser(postCommit); user != "somchai" {
		t.Fatalf("user หาย: %q", user)
	}

	if token, _ := requestcontext.GetToken(postCommit); token != "Bearer token-test" {
		t.Fatalf("token หาย: %q", token)
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**

Run: `go test ./internal/services/delivery-service/ -run TestPostCommitContextSurvivesCallerCancel`
Expected: FAIL — `undefined: postCommitContext`

- [ ] **Step 3: เพิ่มฟังก์ชัน `postCommitContext` แล้วใช้แทนโค้ดเดิม**

ใน `internal/services/delivery-service/update-status-delivery.go`:

```go
// postCommitContext คืน context สำหรับงานที่ทำหลัง commit
// เก็บ user/token ไว้ครบ แต่ตัดการยกเลิกทิ้ง ไม่งั้นพอ caller หมดเวลาแล้วตัดสาย
// งานที่เหลือจะไม่เกิดขึ้นเลยและเงียบด้วย
func postCommitContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}
```

แล้วใน `closeSalesOfDeliveries` เปลี่ยน `ctx = context.WithoutCancel(ctx)` เป็น `ctx = postCommitContext(ctx)`

- [ ] **Step 4: รันเทสให้ผ่าน**

Run: `go test ./internal/services/delivery-service/ -run TestPostCommitContextSurvivesCallerCancel`
Expected: PASS

- [ ] **Step 5: แปลง service ที่เหลือใน package**

ไล่ทีละไฟล์ตาม 4 ข้อที่หัวเฟส B แล้วแก้ route ของ delivery ใน `internal/routes/routes.go`

- [ ] **Step 6: ยืนยัน**

Run: `go build ./... && go test ./internal/services/delivery-service/`
Expected: BUILD ผ่าน, test เดิม 92 ตัว + เทสใหม่ PASS

Run: `grep -rn "gin.Context" internal/services/delivery-service --include=*.go`
Expected: ไม่มีผลลัพธ์

- [ ] **Step 7: Commit**

```bash
git add internal/services/delivery-service internal/routes/routes.go
git commit -m "refactor(delivery): take context.Context in delivery services"
```

### Task 5: sale-service (12 ไฟล์)

**Files:**
- Modify: `internal/services/sale-service/*.go`
- Modify: `internal/routes/routes.go` (route ของ sale)
- Test: `internal/services/sale-service/user_from_context_test.go`

**Interfaces:**
- Consumes: `requestcontext.GetUserOrDefault`, `apperr.*`
- Produces: service ทุกตัวใน package รับ `context.Context`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

```go
package saleService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// ยืนยันว่า package นี้อ่าน user จาก context ไม่ใช่จาก gin
func TestUserFromContext(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := requestcontext.GetUserOrDefault(ctx); got != "somchai" {
		t.Fatalf("user = %q", got)
	}

	if got := requestcontext.GetUserOrDefault(context.Background()); got != requestcontext.DefaultUser {
		t.Fatalf("fallback = %q", got)
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**

Run: `go test ./internal/services/sale-service/ -run TestUserFromContext`
Expected: FAIL — compile error เพราะ package ยังไม่ได้ import `requestcontext` (ถ้า PASS ทันทีให้ข้ามไป Step 3 ได้เลย)

- [ ] **Step 3: แปลงทุกไฟล์ใน package** ตาม 4 ข้อที่หัวเฟส B

- [ ] **Step 4: แก้ route ของ sale ใน `internal/routes/routes.go`**

- [ ] **Step 5: ยืนยัน**

Run: `go build ./... && go test ./internal/services/sale-service/`
Expected: BUILD ผ่าน, test PASS

Run: `grep -rn "gin.Context" internal/services/sale-service --include=*.go`
Expected: ไม่มีผลลัพธ์

- [ ] **Step 6: Commit**

```bash
git add internal/services/sale-service internal/routes/routes.go
git commit -m "refactor(sale): take context.Context in sale services"
```

### Task 6: quotation-service (8 ไฟล์)

ทำตามขั้นตอนเดียวกับ Task 5 ทุกข้อ เปลี่ยนเป็น package `quotationService` โฟลเดอร์ `internal/services/quotation-service`
commit message: `refactor(quotation): take context.Context in quotation services`

**Files:**
- Modify: `internal/services/quotation-service/*.go`
- Modify: `internal/routes/routes.go` (route ของ quotation)
- Test: `internal/services/quotation-service/user_from_context_test.go`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `internal/services/quotation-service/user_from_context_test.go`:

```go
package quotationService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// ยืนยันว่า package นี้อ่าน user จาก context ไม่ใช่จาก gin
func TestUserFromContext(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := requestcontext.GetUserOrDefault(ctx); got != "somchai" {
		t.Fatalf("user = %q", got)
	}

	if got := requestcontext.GetUserOrDefault(context.Background()); got != requestcontext.DefaultUser {
		t.Fatalf("fallback = %q", got)
	}
}
```

ชื่อ package ให้ยึดตามที่ประกาศจริงในไฟล์ของโฟลเดอร์ `internal/services/quotation-service`
- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน** — `go test ./internal/services/quotation-service/ -run TestUserFromContext`
- [ ] **Step 3: แปลงทุกไฟล์ใน package** ตาม 4 ข้อที่หัวเฟส B
- [ ] **Step 4: แก้ route ของ quotation**
- [ ] **Step 5: ยืนยัน** — `go build ./... && go test ./internal/services/quotation-service/` และ grep `gin.Context` ต้องไม่เหลือ
- [ ] **Step 6: Commit**

### Task 7: credit-service + summary-credit (16 ไฟล์)

ทำตามขั้นตอนเดียวกับ Task 5 ทุกข้อ กับ 2 โฟลเดอร์: `internal/services/credit-service` และ `internal/services/summary-credit`
commit message: `refactor(credit): take context.Context in credit services`

**Files:**
- Modify: `internal/services/credit-service/*.go`, `internal/services/summary-credit/*.go`
- Modify: `internal/routes/routes.go` (route ของ credit และ summary-credit)
- Test: `internal/services/credit-service/user_from_context_test.go`

**หมายเหตุเฉพาะ task นี้:** cron `credit-request.go` / `credit-extra.go` เรียกฟังก์ชันใน package นี้ผ่าน HTTP ไม่ได้เรียกตรง จึงไม่ต้องแก้ cron ซ้ำ

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `internal/services/credit-service/user_from_context_test.go`:

```go
package creditService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// ยืนยันว่า package นี้อ่าน user จาก context ไม่ใช่จาก gin
func TestUserFromContext(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := requestcontext.GetUserOrDefault(ctx); got != "somchai" {
		t.Fatalf("user = %q", got)
	}

	if got := requestcontext.GetUserOrDefault(context.Background()); got != requestcontext.DefaultUser {
		t.Fatalf("fallback = %q", got)
	}
}
```

ชื่อ package ให้ยึดตามที่ประกาศจริงในไฟล์ของโฟลเดอร์ `internal/services/credit-service`
- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**
- [ ] **Step 3: แปลงทุกไฟล์ใน 2 package** ตาม 4 ข้อที่หัวเฟส B
- [ ] **Step 4: แก้ route ของ credit และ summary-credit**
- [ ] **Step 5: ยืนยัน** — build + test + grep `gin.Context` ต้องไม่เหลือใน 2 โฟลเดอร์นี้
- [ ] **Step 6: Commit**

### Task 8: invoice-service (15 ไฟล์)

ทำตามขั้นตอนเดียวกับ Task 5 ทุกข้อ โฟลเดอร์ `internal/services/invoice-service`
commit message: `refactor(invoice): take context.Context in invoice services`

**Files:**
- Modify: `internal/services/invoice-service/*.go`
- Modify: `internal/routes/routes.go` (route ของ invoice)
- Test: `internal/services/invoice-service/user_from_context_test.go`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `internal/services/invoice-service/user_from_context_test.go`:

```go
package invoiceService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// ยืนยันว่า package นี้อ่าน user จาก context ไม่ใช่จาก gin
func TestUserFromContext(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := requestcontext.GetUserOrDefault(ctx); got != "somchai" {
		t.Fatalf("user = %q", got)
	}

	if got := requestcontext.GetUserOrDefault(context.Background()); got != requestcontext.DefaultUser {
		t.Fatalf("fallback = %q", got)
	}
}
```

ชื่อ package ให้ยึดตามที่ประกาศจริงในไฟล์ของโฟลเดอร์ `internal/services/invoice-service`
- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**
- [ ] **Step 3: แปลงทุกไฟล์ใน package** ตาม 4 ข้อที่หัวเฟส B
- [ ] **Step 4: แก้ route ของ invoice**
- [ ] **Step 5: ยืนยัน** — build + test + grep
- [ ] **Step 6: Commit**

### Task 9: price-service รวม route ที่รับไฟล์

**Files:**
- Modify: `internal/services/price-service/*.go` (13 ไฟล์ที่ยังรับ `*gin.Context`)
- Modify: `internal/routes/routes.go` (route ของ price รวม 2 route multipart และ 3 route binding)
- Test: `internal/utils/request-handler-multipart_test.go`, `internal/services/price-service/user_from_context_test.go`

**Interfaces:**
- Consumes: `utils.ProcessContextRequestMultipart(c *gin.Context, fn func(context.Context, utils.MultipartInput) (interface{}, error))`, `utils.MultipartInput{Files map[string][]*multipart.FileHeader; Form map[string][]string}`
- Produces: `UploadPricelistMultipart(ctx context.Context, input utils.MultipartInput) (interface{}, error)`, `UploadPricelistTemplateMultipart(ctx context.Context, input utils.MultipartInput) (interface{}, error)`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน — multipart ต้องส่งไฟล์และ user ถึง service ครบ**

สร้าง `internal/utils/request-handler-multipart_test.go`:

```go
package utils

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/requestcontext"

	"github.com/gin-gonic/gin"
)

func TestProcessContextRequestMultipartPassesFileAndUser(t *testing.T) {
	gotFileName := ""
	gotForm := ""
	gotUser := ""

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(requestcontext.WithUser(c.Request.Context(), "somchai"))
		c.Next()
	})
	router.POST("/upload", func(c *gin.Context) {
		ProcessContextRequestMultipart(c, func(ctx context.Context, input MultipartInput) (interface{}, error) {
			gotUser = requestcontext.GetUserOrDefault(ctx)
			if files := input.Files["file"]; len(files) > 0 {
				gotFileName = files[0].Filename
			}
			if values := input.Form["company_code"]; len(values) > 0 {
				gotForm = values[0]
			}

			return gin.H{"ok": true}, nil
		})
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "pricelist.xlsx")
	part.Write([]byte("dummy"))
	writer.WriteField("company_code", "CM")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	if gotFileName != "pricelist.xlsx" {
		t.Fatalf("filename = %q", gotFileName)
	}

	if gotForm != "CM" {
		t.Fatalf("form company_code = %q", gotForm)
	}

	if gotUser != "somchai" {
		t.Fatalf("user = %q", gotUser)
	}
}
```

- [ ] **Step 2: เขียนเทสรูป response ของ error**

เพิ่มใน `internal/utils/request-handler-gin_test.go`:

```go
// หน้าเว็บเดิมอ่าน key "error" รูปนี้ต้องไม่หายไปแม้จะเพิ่ม code เข้ามา
func TestAppErrorResponseKeepsErrorKey(t *testing.T) {
	router := newTestRouter()
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			return nil, apperr.NotFound("pricelist not found")
		})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`)))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}

	body := map[string]string{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("อ่าน body ไม่ได้: %v", err)
	}

	if body["error"] != "pricelist not found" {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
```

- [ ] **Step 3: รันเทสให้เห็นว่าไม่ผ่าน**

Run: `go test ./internal/utils/ -run "TestProcessContextRequestMultipartPassesFileAndUser|TestAppErrorResponseKeepsErrorKey"`
Expected: multipart FAIL (ยังไม่มี service ที่ใช้) หรือ PASS ถ้า helper พร้อมแล้ว — ถ้า PASS ทั้งคู่ให้บันทึกไว้แล้วไปต่อ

- [ ] **Step 4: แปลง service ใน package**

2 ตัวที่รับไฟล์ใช้ลายเซ็นใหม่:

```go
func UploadPricelistMultipart(ctx context.Context, input utils.MultipartInput) (interface{}, error) {
	user := requestcontext.GetUserOrDefault(ctx)

	files := input.Files["file"]
	if len(files) == 0 {
		return nil, apperr.BadRequest("file is required")
	}

	fileHeader := files[0]
	// จากนี้ใช้ fileHeader.Open() แทน c.FormFile
```

3 ตัวที่เดิมใช้ `ProcessRequestWithBinding` เปลี่ยนเป็นรับ payload เป็น string แล้ว `json.Unmarshal` เอง
ส่วนที่เหลือทำตาม 4 ข้อที่หัวเฟส B

- [ ] **Step 5: แก้ route ของ price**

```go
price.POST("/UploadPricelist", func(c *gin.Context) {
	utils.ProcessContextRequestMultipart(c, priceService.UploadPricelistMultipart)
})
```

- [ ] **Step 6: ยืนยัน**

Run: `go build ./... && go test ./internal/utils/ ./internal/services/price-service/`
Expected: BUILD ผ่าน, test PASS (price-service มี test เดิมอยู่หลายตัว ต้องผ่านเท่าเดิม)

- [ ] **Step 7: Commit**

```bash
git add internal/services/price-service internal/utils internal/routes/routes.go
git commit -m "refactor(price): take context.Context in price services including uploads"
```

### Task 10: purchase-service + pre-purchase-service (8 ไฟล์)

ทำตามขั้นตอนเดียวกับ Task 5 ทุกข้อ กับ 2 โฟลเดอร์
commit message: `refactor(purchase): take context.Context in purchase services`

**Files:**
- Modify: `internal/services/purchase-service/*.go`, `internal/services/pre-purchase-service/*.go`
- Modify: `internal/routes/routes.go`
- Test: `internal/services/purchase-service/user_from_context_test.go`

**หมายเหตุ:** 2 package นี้ใช้ตัวแปรชื่อ `userCode` แทน `user` ให้เปลี่ยนเป็น `userCode := requestcontext.GetUserOrDefault(ctx)`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `internal/services/purchase-service/user_from_context_test.go`:

```go
package purchaseService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// ยืนยันว่า package นี้อ่าน user จาก context ไม่ใช่จาก gin
func TestUserFromContext(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := requestcontext.GetUserOrDefault(ctx); got != "somchai" {
		t.Fatalf("user = %q", got)
	}

	if got := requestcontext.GetUserOrDefault(context.Background()); got != requestcontext.DefaultUser {
		t.Fatalf("fallback = %q", got)
	}
}
```

ชื่อ package ให้ยึดตามที่ประกาศจริงในไฟล์ของโฟลเดอร์ `internal/services/purchase-service`
- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**
- [ ] **Step 3: แปลงทุกไฟล์ใน 2 package** ตาม 4 ข้อที่หัวเฟส B
- [ ] **Step 4: แก้ route**
- [ ] **Step 5: ยืนยัน** — build + test + grep
- [ ] **Step 6: Commit**

### Task 11: approval + deposit + payment (10 ไฟล์)

ทำตามขั้นตอนเดียวกับ Task 5 ทุกข้อ กับ 3 โฟลเดอร์
commit message: `refactor(approval): take context.Context in approval, deposit and payment services`

**Files:**
- Modify: `internal/services/approval-service/*.go`, `internal/services/deposit-service/*.go`, `internal/services/payment-service/*.go`
- Modify: `internal/routes/routes.go`
- Test: `internal/services/approval-service/user_from_context_test.go`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `internal/services/approval-service/user_from_context_test.go`:

```go
package approvalService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// ยืนยันว่า package นี้อ่าน user จาก context ไม่ใช่จาก gin
func TestUserFromContext(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := requestcontext.GetUserOrDefault(ctx); got != "somchai" {
		t.Fatalf("user = %q", got)
	}

	if got := requestcontext.GetUserOrDefault(context.Background()); got != requestcontext.DefaultUser {
		t.Fatalf("fallback = %q", got)
	}
}
```

ชื่อ package ให้ยึดตามที่ประกาศจริงในไฟล์ของโฟลเดอร์ `internal/services/approval-service`
- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**
- [ ] **Step 3: แปลงทุกไฟล์ใน 3 package** ตาม 4 ข้อที่หัวเฟส B
- [ ] **Step 4: แก้ route**
- [ ] **Step 5: ยืนยัน** — build + test + grep
- [ ] **Step 6: Commit**

### Task 12: package ที่เหลือทั้งหมด

**Files:**
- Modify: `internal/services/group-service/*.go`, `system-config/*.go`, `time-service/*.go`, `unit-service/*.go`, `verify-service/*.go`, `x-service/*.go`, `email-service/*.go`, `customer-service/*.go`, `interface-service/*.go`, `authentication-service/*.go`, `cronjob-service/*.go`
- Modify: `internal/routes/routes.go` (route ที่เหลือทั้งหมด)
- Test: `internal/services/interface-service/goroutine_context_test.go`

**Interfaces:**
- Consumes: `requestcontext.WithUser`, `requestcontext.GetToken`, `context.WithoutCancel`
- Produces: service ทุกตัวในทุก package รับ `context.Context`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน — งานใน goroutine ต้องไม่ตายตอน handler คืนค่า**

สร้าง `internal/services/interface-service/goroutine_context_test.go`:

```go
package interfaceService

import (
	"context"
	"testing"
	"time"

	"prime-erp-core/internal/requestcontext"
)

// goroutine ที่ยังทำงานต่อหลัง handler คืนค่า ต้องใช้ context ที่ไม่ถูกยกเลิกตาม
func TestBackgroundContextOutlivesRequest(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	requestCtx, done := context.WithCancel(ctx)
	background := backgroundContext(requestCtx)

	done() // handler คืนค่าแล้ว

	select {
	case <-background.Done():
		t.Fatal("งานเบื้องหลังถูกยกเลิกตาม request")
	case <-time.After(10 * time.Millisecond):
	}

	if user, _ := requestcontext.GetUser(background); user != "somchai" {
		t.Fatalf("user หาย: %q", user)
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าไม่ผ่าน**

Run: `go test ./internal/services/interface-service/ -run TestBackgroundContextOutlivesRequest`
Expected: FAIL — `undefined: backgroundContext`

- [ ] **Step 3: เพิ่ม `backgroundContext` แล้วใช้กับทุก goroutine ที่ไม่ได้ `wg.Wait()`**

```go
// backgroundContext ใช้กับงานที่ปล่อยไว้ใน goroutine แล้วไม่รอ
// เก็บ user/token ไว้ แต่ไม่ถูกยกเลิกตอน handler คืนค่า
func backgroundContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}
```

หา goroutine ทั้งหมดด้วย `grep -rn "go func()" internal/services --include=*.go`
ตัวที่มี `wg.Wait()` ตามหลังไม่ต้องแก้ ตัวที่ไม่มีให้ส่ง `backgroundContext(ctx)` เข้าไปใช้แทน `ctx`

- [ ] **Step 4: รันเทสให้ผ่าน**

Run: `go test ./internal/services/interface-service/ -run TestBackgroundContextOutlivesRequest`
Expected: PASS

- [ ] **Step 5: แปลง package ที่เหลือทั้งหมด** ตาม 4 ข้อที่หัวเฟส B แล้วแก้ route ที่เหลือ

- [ ] **Step 6: ยืนยัน**

Run: `go build ./... && go test ./internal/...`
Expected: BUILD ผ่าน, test ผ่าน (ยกเว้น 3 ชุดที่ต้องใช้ docker: price_list_formulas, price_list_sub_group, priceList)

Run: `grep -rn "utils.ProcessRequest(" internal/routes --include=*.go`
Expected: ไม่มีผลลัพธ์

- [ ] **Step 7: Commit**

```bash
git add internal
git commit -m "refactor(services): take context.Context in remaining services"
```

### Task 13: ด่านกันของเก่ากลับมา + เก็บกวาด

**Files:**
- Create: `internal/guard/no_legacy_context_test.go`
- Modify: `internal/utils/request-handler.go` (ลบตัวที่ไม่มีคนใช้แล้ว)
- Modify: `internal/utils/print.go` ถ้ามีการอ้าง `*gin.Context`

**Interfaces:**
- Consumes: ไม่มี (เทสอ่านไฟล์ในโปรเจกต์ตรงๆ)

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `internal/guard/no_legacy_context_test.go`:

```go
package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ห้ามกลับไปใช้ของเดิมที่ทำให้ user หายระหว่างทาง
func TestNoLegacyContextUsage(t *testing.T) {
	banned := map[string]string{
		"http.NewRequest(":       "ใช้ utils.NewRequest(ctx, ...) แทน ไม่งั้น token ไม่ถูกส่งต่อ",
		`ctx.GetString("user")`:  "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
		`c.GetString("user")`:    "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
	}

	roots := []string{"../services", "../../external"}

	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			for pattern, reason := range banned {
				if strings.Contains(string(content), pattern) {
					t.Errorf("%s: เจอ %q — %s", path, pattern, reason)
				}
			}

			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นผล**

Run: `go test ./internal/guard/`
Expected: PASS ถ้า Task 1-12 ครบ ถ้า FAIL ให้ไล่แก้จุดที่เทสฟ้องจนหมด

- [ ] **Step 3: ลบ helper ที่ไม่มีคนใช้แล้ว**

Run: `grep -rn "ProcessRequest\b\|ProcessRequestWithBinding\|ProcessRequestMultiPart" internal --include=*.go`
ถ้าไม่มีใครเรียกแล้ว ให้ลบออกจาก `internal/utils/request-handler.go` พร้อมเทสของมัน

- [ ] **Step 4: ยืนยันทั้ง repo**

Run: `go build ./... && go vet ./internal/... ./external/... && go test ./...`
Expected: BUILD ผ่าน, vet ไม่มี error ใหม่, test ผ่าน (ยกเว้น 3 ชุดที่ต้องใช้ docker)

- [ ] **Step 5: Commit**

```bash
git add internal
git commit -m "test(guard): block legacy gin context and user lookups"
```

---

## หลังจบทุก task

1. เพิ่ม `SERVICE_TOKEN` ในไฟล์ env ของทุก environment (cron ยิงออกแบบไม่มี token ถ้ายังไม่มีค่านี้)
2. e2e: กดจากเว็บ 1 เส้นที่ข้าม service (เช่น cancel delivery ที่ยิงไป order-service) แล้วเช็คใน DB ว่า `update_by` ทั้ง 2 ฝั่งเป็นชื่อคนกด พร้อมแคปหน้าจอ ทั้งเคสปกติและเคส error
3. หยุดที่ merge `Develop` รอบ deploy ขึ้น UAT/prod เจ้าของกำหนดเอง
