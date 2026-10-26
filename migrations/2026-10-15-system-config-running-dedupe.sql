-- ล้างแถว system_config ที่ซ้ำ (จาก 2026-10-01-system-config-seed-running-reset.sql) โดยไม่แตะเลข running
-- ปลอดภัยกับทุก env รวม production, รันซ้ำได้

BEGIN;

-- 1) RUNNING: เหลือ 1 แถวต่อ config_code (period ล่าสุด แล้วเลขมากสุด เพื่อไม่ให้ออกเลขซ้ำ)
DELETE FROM system_config t
USING (
  SELECT ctid, row_number() OVER (
    PARTITION BY config_code
    ORDER BY "json"->>'year' DESC, "json"->>'month' DESC,
             ("json"->>'current_running')::int DESC
  ) rn
  FROM system_config WHERE topic_code = 'RUNNING'
) d
WHERE t.ctid = d.ctid AND d.rn > 1;

-- 2) config อื่นที่ซ้ำกันทุกคอลัมน์ ให้เหลือแถวเดียว
DELETE FROM system_config t
USING (
  SELECT ctid, row_number() OVER (
    PARTITION BY topic_code, config_code, tenant_id, config_name,
                 cond1, cond2, value, "sequence", remark, "json"::text
  ) rn
  FROM system_config
) d
WHERE t.ctid = d.ctid AND d.rn > 1;

-- กันไม่ให้ running ซ้ำอีก
CREATE UNIQUE INDEX IF NOT EXISTS ux_system_config_running
  ON system_config (config_code) WHERE topic_code = 'RUNNING';

COMMIT;
