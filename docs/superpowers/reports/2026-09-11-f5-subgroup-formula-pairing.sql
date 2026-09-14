-- รายงาน F5: subgroup ที่ผูกสูตรราคาไม่ครบคู่
--
-- แต่ละ subgroup ควรผูกสูตร uom 'kg' หนึ่งตัวและ uom 'pcs' หนึ่งตัว
-- เพื่อให้ทั้ง total_net_price_weight และ total_net_price_unit ถูกคำนวณ
--
-- พบว่ามี 147 subgroup ที่ผูกสูตร uom 'pcs' ทั้งสองตัว ไม่มีสูตร 'kg' เลย
-- ในจำนวนนั้น 131 ตัวผูกสูตรเดียวกันซ้ำสองครั้ง
-- ผลคือ total_net_price_weight ไม่เคยถูกคำนวณใหม่ และคำนวณสิ่งเดิมซ้ำลง total_net_price_unit
--
-- เป็นปัญหาข้อมูล ไม่ใช่โค้ด ต้องให้ธุรกิจตัดสินว่าแต่ละกลุ่มควรผูกสูตร kg ตัวไหน
--
-- READ-ONLY ไม่แก้ข้อมูลใด ๆ · ใช้กับ database prime_erp

\echo '=== 1) สรุปภาพรวมการจับคู่ uom ของสูตรต่อ subgroup ==='
SELECT
    uom_combo,
    formula_names,
    count(*) AS subgroups
FROM (
    SELECT
        m.price_list_subgroup_code,
        string_agg(f.uom, '+' ORDER BY f.uom) AS uom_combo,
        string_agg(f.name, '  ||  ' ORDER BY f.uom, f.name) AS formula_names
    FROM price_list_subgroup_formulas_map m
    JOIN price_list_formulas f ON f.formula_code = m.price_list_formulas_code
    GROUP BY m.price_list_subgroup_code
) t
GROUP BY uom_combo, formula_names
ORDER BY subgroups DESC;

\echo ''
\echo '=== 2) subgroup ที่ไม่มีสูตร uom kg เลย (total_net_price_weight ไม่ถูกคำนวณใหม่) ==='
SELECT
    m.price_list_subgroup_code,
    string_agg(DISTINCT f.name, '  ||  ' ORDER BY f.name) AS formulas_bound,
    s.total_net_price_weight,
    s.total_net_price_unit
FROM price_list_subgroup_formulas_map m
JOIN price_list_formulas f ON f.formula_code = m.price_list_formulas_code
LEFT JOIN price_list_sub_group s ON s.subgroup_code = m.price_list_subgroup_code
GROUP BY m.price_list_subgroup_code, s.total_net_price_weight, s.total_net_price_unit
HAVING count(*) FILTER (WHERE f.uom = 'kg') = 0
ORDER BY m.price_list_subgroup_code;

\echo ''
\echo '=== 3) subgroup ที่ผูกสูตรเดียวกันซ้ำมากกว่าหนึ่งครั้ง ==='
SELECT
    m.price_list_subgroup_code,
    f.name AS duplicated_formula,
    f.uom,
    count(*) AS times_bound
FROM price_list_subgroup_formulas_map m
JOIN price_list_formulas f ON f.formula_code = m.price_list_formulas_code
GROUP BY m.price_list_subgroup_code, f.name, f.uom
HAVING count(*) > 1
ORDER BY times_bound DESC, m.price_list_subgroup_code;
