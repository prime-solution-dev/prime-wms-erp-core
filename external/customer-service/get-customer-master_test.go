package externalService

import (
	"encoding/json"
	"testing"
)

// customer service เปลี่ยนรูปแบบ address จาก array เป็นสตริง แล้วทำให้ทุก endpoint
// ที่แปลงชื่อลูกค้าเป็นรหัสตอบ 500 เทสนี้กันไม่ให้ประกาศ type กลับไปผูกกับรูปแบบใดรูปแบบหนึ่ง
func TestGetCustomerResponseAcceptsBothAddressShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "address เป็นสตริงว่าง (รูปแบบที่ service ส่งอยู่ตอนนี้)",
			body: `{"customers":[{"customer_code":"CK0332","customer_name":"TMI","address":""}]}`,
		},
		{
			name: "address เป็นสตริงที่มีค่า",
			body: `{"customers":[{"customer_code":"CK0332","customer_name":"TMI","address":"48 กระทุ่มราย"}]}`,
		},
		{
			name: "address เป็น array ของที่อยู่ (รูปแบบเดิม)",
			body: `{"customers":[{"customer_code":"CK0332","customer_name":"TMI","address":[{"address_code":"A1","address":"48 กระทุ่มราย"}]}]}`,
		},
		{
			name: "ไม่มี key address เลย",
			body: `{"customers":[{"customer_code":"CK0332","customer_name":"TMI"}]}`,
		},
		{
			name: "address เป็น null",
			body: `{"customers":[{"customer_code":"CK0332","customer_name":"TMI","address":null}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var res ResultCustomerResponse
			if err := json.Unmarshal([]byte(tc.body), &res); err != nil {
				t.Fatalf("unmarshal ล้ม: %v", err)
			}
			if len(res.Customers) != 1 {
				t.Fatalf("want 1 customer, got %d", len(res.Customers))
			}
			if got := res.Customers[0].CustomerCode; got != "CK0332" {
				t.Fatalf("want customer_code CK0332, got %q", got)
			}
		})
	}
}
