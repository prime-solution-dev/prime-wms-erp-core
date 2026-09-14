-- แก้ expression ของสูตร kg = [Pcs] / [Weight Spec]
--
-- expression เดิมเป็น pcs*weight_spec ซึ่งเป็นการคูณ ไม่ตรงกับชื่อสูตรที่ระบุว่าหาร
-- คู่ผกผันของสูตรนี้คือ Pcs = [kg] x [Weight Spec] (kg*weight_spec)
-- weight_spec มีหน่วย kg ต่อชิ้น ดังนั้น ราคาต่อกิโล = ราคาต่อชิ้น / kg ต่อชิ้น
--
-- ระบุแถวด้วย expression ที่ผิดอย่างเดียว ไม่ใช้ name ไม่ใช้ id ไม่ใช้ formula_code
-- เพราะ name ในระบบนี้ไม่ถูก normalize ช่องว่าง (สูตรคู่ของมันชื่อ
-- "Pcs = [kg]  x [Weight Spec]" มีช่องว่างซ้อนสองตัว) การ match ด้วย name
-- จึงเสี่ยงไม่แมตช์เงียบ ๆ ส่วน id และ formula_code ต่างกันในแต่ละ environment
--
-- ไม่มีสูตรอื่นในระบบที่ใช้ expression pcs*weight_spec อย่างถูกต้อง
-- ยืนยันจาก internal/scripts/price_list_formulas/price-list-formulas.json
--
-- ต้องรันทุก environment ที่ seed ไปแล้ว — seed script ใช้
-- ON CONFLICT (formula_code) DO NOTHING จึงไม่ซ่อมข้อมูลเดิมให้
--
-- idempotent: รันซ้ำได้ เพราะ WHERE จะไม่แมตช์อะไรเมื่อแก้ไปแล้ว

BEGIN;

UPDATE price_list_formulas
SET expression = 'pcs/weight_spec'
WHERE expression = 'pcs*weight_spec';

-- ตรวจผลด้วย predicate เดียวกับ UPDATE เพื่อให้จับได้จริงว่าไม่เหลือแถวที่ผิด
-- (ถ้าใช้ name เป็นเงื่อนไข การไม่แมตช์จะทำให้ count เป็น 0 แล้วรายงานสำเร็จทั้งที่ไม่ได้แก้)
DO $$
DECLARE
    wrong_count integer;
BEGIN
    SELECT count(*) INTO wrong_count
    FROM price_list_formulas
    WHERE expression = 'pcs*weight_spec';

    IF wrong_count > 0 THEN
        RAISE EXCEPTION 'ยังเหลือสูตรที่ expression เป็น pcs*weight_spec อยู่ % แถว', wrong_count;
    END IF;
END $$;

COMMIT;
