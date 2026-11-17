-- ผู้ติดต่อที่ผู้ใช้กรอกเองบนใบเสนอราคา (TMI issue log #40/#42)
-- ว่างได้: PDF ใบเสนอราคาจะใช้ผู้ติดต่อจาก customer master แทน
-- +goose Up
-- +goose StatementBegin
ALTER TABLE quotation ADD COLUMN IF NOT EXISTS contact_person VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE quotation DROP COLUMN IF EXISTS contact_person;
-- +goose StatementEnd
