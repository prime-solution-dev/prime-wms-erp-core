-- รีเซ็ต running ทุกตัวเป็น 0
-- !!! ใช้เฉพาะ env ที่ล้างเอกสาร (transaction) แล้ว เช่น UAT ห้ามรันบน production ที่มีเอกสารอยู่ จะออกเลขซ้ำ
-- ต้องรัน 2026-10-15-system-config-running-dedupe.sql ก่อน

UPDATE system_config
SET "json" = ("json"::jsonb || '{"current_running":0}')::json
WHERE topic_code = 'RUNNING';
