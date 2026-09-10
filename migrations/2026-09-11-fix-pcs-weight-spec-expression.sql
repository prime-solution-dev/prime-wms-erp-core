-- แก้ expression ของสูตร kg = [Pcs] / [Weight Spec]
--
-- expression เดิมเป็น pcs*weight_spec ซึ่งเป็นการคูณ ไม่ตรงกับชื่อสูตรที่ระบุว่าหาร
-- คู่ผกผันของสูตรนี้คือ Pcs = [kg] x [Weight Spec] (kg*weight_spec)
-- weight_spec มีหน่วย kg ต่อชิ้น ดังนั้น ราคาต่อกิโล = ราคาต่อชิ้น / kg ต่อชิ้น
--
-- ระบุแถวด้วย name คู่กับ expression ที่ผิด ไม่ใช้ id หรือ formula_code
-- เพราะทั้งสองค่าอาจต่างกันในแต่ละ environment
--
-- idempotent: รันซ้ำได้ เพราะ WHERE จะไม่แมตช์อะไรเมื่อแก้ไปแล้ว
UPDATE price_list_formulas
SET expression = 'pcs/weight_spec'
WHERE name = 'kg = [Pcs] / [Weight Spec]'
  AND expression = 'pcs*weight_spec';

-- ตรวจผล: ต้องไม่เหลือแถวที่ชื่อบอกหารแต่ expression ไม่ใช่การหาร
DO $$
DECLARE
    wrong_count integer;
BEGIN
    SELECT count(*) INTO wrong_count
    FROM price_list_formulas
    WHERE name = 'kg = [Pcs] / [Weight Spec]'
      AND expression <> 'pcs/weight_spec';

    IF wrong_count > 0 THEN
        RAISE EXCEPTION 'ยังเหลือสูตร kg = [Pcs] / [Weight Spec] ที่ expression ไม่ถูกต้อง % แถว', wrong_count;
    END IF;
END $$;
