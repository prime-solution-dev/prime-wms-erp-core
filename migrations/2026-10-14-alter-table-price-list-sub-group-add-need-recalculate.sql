-- flag บอกว่า subgroup ต้องคำนวณราคาใหม่ตอนกด Edit ในหน้า detail หรือไม่
-- set เป็น true ตอน update ราคา group (UpdatePriceListGroupBase) ที่ราคาเปลี่ยนจริง
-- ล้างเป็น false ตอน save หน้า detail (UpdatePriceListSubGroup)
--
-- แถวเดิมได้ false ทั้งหมด กด Edit จะไม่คำนวณใหม่จนกว่าจะแก้ราคา group ครั้งถัดไป
ALTER TABLE price_list_sub_group
ADD COLUMN IF NOT EXISTS need_recalculate BOOLEAN NOT NULL DEFAULT false;
