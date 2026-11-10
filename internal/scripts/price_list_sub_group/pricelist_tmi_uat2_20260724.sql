-- =============================================================================
-- price_list_sub_group loader  (TMI_UAT2 Pricelist template, V0 2026-07-24)
-- Generated 2026-07-24T09:18:42Z  |  52 subgroups
--
-- Source : TMI_UAT2_Pricelist upload_20260724_V0.xlsx  (sheet 'Pricelist')
-- company_code = 09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3
-- site_code    = TMI_WH
--
-- Convention (matches upload-pricelist.go genKeyFromCols & seed-price-list.go):
--   subgroup_code  <- 'Product code'
--   subgroup_key   <- non-empty 'Product Group 1..10' cells joined with '|'
--   price_list_sub_group_key: code=PG01..PG10, value=full cell, seq=absolute col pos
--
-- Assumptions (edit if needed): is_trading=false, effective_date=2026-01-01.
--   Excel Price_unit/Price_weight -> total_net_price_unit/total_net_price_weight
--   (and before_total_net_*); price_unit/price_weight/extra kept 0.
--   price_list_group_id resolved by group_code so it works even if target ids
--   differ from the pasted JSON.
-- Dedup: match existing rows by (price_list_group_id, subgroup_key); DELETE any
--   match (placeholder SGxx and prior loads) then INSERT fresh -> exactly one
--   row per (group, subgroup_key). Re-runnable (each run deletes then reinserts).
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- 0) Resolve target (group_id, subgroup_key) pairs from group_code
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE _tmi_targets (group_id uuid NOT NULL, subgroup_key text NOT NULL) ON COMMIT DROP;
INSERT INTO _tmi_targets (group_id, subgroup_key)
SELECT g.id, v.subgroup_key
FROM (VALUES
  ('GROUP_1_ITEM_1', 'PG01_3|PG02_6|PG03_8|PG05_3|PG06_33'),
  ('GROUP_1_ITEM_1', 'PG01_3|PG02_6|PG03_8|PG05_3|PG06_45'),
  ('GROUP_1_ITEM_1', 'PG01_3|PG02_6|PG03_8|PG05_4|PG06_45'),
  ('GROUP_1_ITEM_1', 'PG01_3|PG02_6|PG03_8|PG05_5|PG06_45'),
  ('GROUP_1_ITEM_1', 'PG01_3|PG02_6|PG03_8|PG05_5|PG06_58'),
  ('GROUP_1_ITEM_1', 'PG01_3|PG02_9|PG03_21|PG05_3|PG06_33'),
  ('GROUP_1_ITEM_1', 'PG01_3|PG02_9|PG03_21|PG05_3|PG06_45'),
  ('GROUP_1_ITEM_6', 'PG01_6|PG02_8|PG03_1|PG04_83|PG07_1'),
  ('GROUP_1_ITEM_6', 'PG01_6|PG02_8|PG03_1|PG04_90|PG07_1'),
  ('GROUP_1_ITEM_12', 'PG01_11|PG02_2|PG03_1|PG04_92|PG07_1|PG08_2|PG09_3'),
  ('GROUP_1_ITEM_12', 'PG01_11|PG02_2|PG03_1|PG04_100|PG07_1|PG08_2|PG09_3'),
  ('GROUP_1_ITEM_12', 'PG01_11|PG02_2|PG03_1|PG04_113|PG07_1|PG08_2|PG09_3'),
  ('GROUP_1_ITEM_12', 'PG01_11|PG02_2|PG03_1|PG04_119|PG07_1|PG08_2|PG09_3'),
  ('GROUP_1_ITEM_12', 'PG01_11|PG02_1|PG03_1|PG04_99|PG07_1|PG08_2|PG09_3'),
  ('GROUP_1_ITEM_12', 'PG01_11|PG02_1|PG03_1|PG04_109|PG07_1|PG08_2|PG09_3'),
  ('GROUP_1_ITEM_12', 'PG01_11|PG02_1|PG03_1|PG04_118|PG07_1|PG08_2|PG09_3'),
  ('GROUP_1_ITEM_2', 'PG01_7|PG03_5|PG04_59|PG06_18|PG07_1|PG09_1'),
  ('GROUP_1_ITEM_2', 'PG01_7|PG03_5|PG04_63|PG06_18|PG07_1|PG09_1'),
  ('GROUP_1_ITEM_2', 'PG01_7|PG03_5|PG04_59|PG06_29|PG07_1|PG09_1'),
  ('GROUP_1_ITEM_2', 'PG01_7|PG03_5|PG04_63|PG06_29|PG07_1|PG09_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_11|PG03_2|PG04_18|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_11|PG03_2|PG04_20|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_11|PG03_2|PG04_24|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_11|PG03_2|PG04_18|PG06_29|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_11|PG03_2|PG04_20|PG06_29|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_11|PG03_2|PG04_24|PG06_29|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_13|PG03_2|PG04_21|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_13|PG03_2|PG04_23|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_13|PG03_2|PG04_21|PG06_29|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_13|PG03_2|PG04_23|PG06_29|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_16|PG03_2|PG04_2|PG06_29|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_16|PG03_2|PG04_3|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_16|PG03_2|PG04_4|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_16|PG03_2|PG04_6|PG06_18|PG07_1'),
  ('GROUP_1_ITEM_4', 'PG01_4|PG02_16|PG03_2|PG04_8|PG06_29|PG07_1'),
  ('GROUP_1_ITEM_5', 'PG01_4|PG02_14|PG03_1|PG04_17|PG06_5|PG07_1|PG08_5'),
  ('GROUP_1_ITEM_5', 'PG01_4|PG02_14|PG03_1|PG04_17|PG06_8|PG07_1|PG08_5'),
  ('GROUP_1_ITEM_5', 'PG01_4|PG02_14|PG03_1|PG04_21|PG06_5|PG07_1|PG08_5'),
  ('GROUP_1_ITEM_3', 'PG01_4|PG02_15|PG03_1|PG04_51|PG06_5|PG07_1'),
  ('GROUP_1_ITEM_3', 'PG01_4|PG02_15|PG03_1|PG04_51|PG06_10|PG07_1'),
  ('GROUP_1_ITEM_3', 'PG01_4|PG02_15|PG03_1|PG04_62|PG06_13|PG07_1'),
  ('GROUP_1_ITEM_3', 'PG01_4|PG02_15|PG03_1|PG04_59|PG06_13|PG07_1'),
  ('GROUP_1_ITEM_9', 'PG01_1|PG02_3|PG03_23|PG06_46|PG07_3|PG09_7'),
  ('GROUP_1_ITEM_9', 'PG01_1|PG02_3|PG03_6|PG06_46|PG07_3|PG09_7'),
  ('GROUP_1_ITEM_9', 'PG01_1|PG02_10|PG03_23|PG06_58|PG07_3|PG09_7'),
  ('GROUP_1_ITEM_9', 'PG01_1|PG02_10|PG03_23|PG06_64|PG07_3|PG09_7'),
  ('GROUP_1_ITEM_9', 'PG01_1|PG02_10|PG03_6|PG06_58|PG07_3|PG09_7'),
  ('GROUP_1_ITEM_9', 'PG01_1|PG02_10|PG03_6|PG06_64|PG07_3|PG09_7'),
  ('GROUP_1_ITEM_9', 'PG01_1|PG02_10|PG03_23|PG06_58|PG07_4|PG09_7'),
  ('GROUP_1_ITEM_8', 'PG01_9|PG03_15|PG05_16|PG06_3|PG08_13|PG09_8'),
  ('GROUP_1_ITEM_8', 'PG01_9|PG03_15|PG05_16|PG06_5|PG08_13|PG09_8'),
  ('GROUP_1_ITEM_8', 'PG01_9|PG03_15|PG05_16|PG06_10|PG08_13|PG09_8')
) AS v(group_code, subgroup_key)
JOIN public.price_list_group g ON g.group_code = v.group_code
  AND g.site_code = 'TMI_WH' AND g.company_code = '09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3';

-- ---------------------------------------------------------------------------
-- 1) DELETE dependents, then matching existing subgroups, by (group_id, subgroup_key)
-- ---------------------------------------------------------------------------
-- 1a) formulas_map FK references price_list_sub_group.subgroup_code
DELETE FROM public.price_list_subgroup_formulas_map m
USING public.price_list_sub_group sg, _tmi_targets t
WHERE m.price_list_subgroup_code = sg.subgroup_code
  AND sg.price_list_group_id = t.group_id
  AND sg.subgroup_key = t.subgroup_key;

-- 1b) sub_group_key FK references price_list_sub_group.id
DELETE FROM public.price_list_sub_group_key k
USING public.price_list_sub_group sg, _tmi_targets t
WHERE k.sub_group_id = sg.id
  AND sg.price_list_group_id = t.group_id
  AND sg.subgroup_key = t.subgroup_key;

-- 1c) the subgroups themselves
DELETE FROM public.price_list_sub_group sg
USING _tmi_targets t
WHERE sg.price_list_group_id = t.group_id
  AND sg.subgroup_key = t.subgroup_key;

-- ---------------------------------------------------------------------------
-- 2) INSERT fresh price_list_sub_group rows
-- ---------------------------------------------------------------------------
INSERT INTO public.price_list_sub_group (
  id, price_list_group_id, subgroup_code, subgroup_key, is_trading,
  price_unit, extra_price_unit, total_net_price_unit,
  price_weight, extra_price_weight, term_price_weight, total_net_price_weight,
  before_price_unit, before_extra_price_unit, before_total_net_price_unit,
  before_price_weight, before_extra_price_weight, before_term_price_weight, before_total_net_price_weight,
  effective_date, remark, create_by, create_dtm, update_by, update_dtm
) VALUES
  ('44a0042a-5d56-5acb-89c5-5a06f742f893', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_1' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PTB3.848', 'PG01_3|PG02_6|PG03_8|PG05_3|PG06_33', false, 0, 0, 0, 0, 0, 0, 18.6, 0, 0, 0, 0, 0, 0, 18.6, '2026-01-01 00:00:00+00', 'เหล็กแผ่น   3.8x4''x8''', 'system', now(), 'system', now()),
  ('7f9fd279-bea4-5d67-9a18-a1c40e10a305', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_1' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PTB5.848', 'PG01_3|PG02_6|PG03_8|PG05_3|PG06_45', false, 0, 0, 0, 0, 0, 0, 18.6, 0, 0, 0, 0, 0, 0, 18.6, '2026-01-01 00:00:00+00', 'เหล็กแผ่น   5.8x4''x8''', 'system', now(), 'system', now()),
  ('fca84583-78bb-5ca9-9b64-50e26b6ab7af', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_1' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PTB5.8510', 'PG01_3|PG02_6|PG03_8|PG05_4|PG06_45', false, 0, 0, 0, 0, 0, 0, 18.6, 0, 0, 0, 0, 0, 0, 18.6, '2026-01-01 00:00:00+00', 'เหล็กแผ่น   5.8x5''x10''', 'system', now(), 'system', now()),
  ('f2d2ba71-869d-57e8-bb54-7869589324dc', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_1' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PTB5.8520', 'PG01_3|PG02_6|PG03_8|PG05_5|PG06_45', false, 0, 0, 0, 0, 0, 0, 18.6, 0, 0, 0, 0, 0, 0, 18.6, '2026-01-01 00:00:00+00', 'เหล็กแผ่น   5.8x5''x20''', 'system', now(), 'system', now()),
  ('bb5b08dc-ca69-526d-a54e-4f3354242af3', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_1' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PTB11.8520', 'PG01_3|PG02_6|PG03_8|PG05_5|PG06_58', false, 0, 0, 0, 0, 0, 0, 18.6, 0, 0, 0, 0, 0, 0, 18.6, '2026-01-01 00:00:00+00', 'เหล็กแผ่น   11.8x5''x20''', 'system', now(), 'system', now()),
  ('49b7d724-1b56-5519-b574-550baf3c9d72', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_1' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PTH3.848I', 'PG01_3|PG02_9|PG03_21|PG05_3|PG06_33', false, 0, 0, 2800.0, 0, 0, 0, 0, 0, 0, 2800.0, 0, 0, 0, 0, '2026-01-01 00:00:00+00', 'เหล็กแผ่นลาย   3.8x4''x8'' LT', 'system', now(), 'system', now()),
  ('8fa09169-c333-5262-a40a-887039c3f0c4', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_1' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PTH5.848I', 'PG01_3|PG02_9|PG03_21|PG05_3|PG06_45', false, 0, 0, 4350.0, 0, 0, 0, 0, 0, 0, 4350.0, 0, 0, 0, 0, '2026-01-01 00:00:00+00', 'เหล็กแผ่นลาย   5.8x4''x8'' LT', 'system', now(), 'system', now()),
  ('9ea3b212-29db-5c33-858e-3a81b0d3d134', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_6' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'UBB7556', 'PG01_6|PG02_8|PG03_1|PG04_83|PG07_1', false, 0, 0, 775.0, 0, 0, 0, 0, 0, 0, 775.0, 0, 0, 0, 0, '2026-01-01 00:00:00+00', 'เหล็กรางน้ำ 75x40x5x7x6ม.', 'system', now(), 'system', now()),
  ('6fd3cf3a-d7bc-5b9b-bb18-7e975a25d4fd', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_6' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'UBB1506.56', 'PG01_6|PG02_8|PG03_1|PG04_90|PG07_1', false, 0, 0, 2230.0, 0, 0, 0, 0, 0, 0, 2230.0, 0, 0, 0, 0, '2026-01-01 00:00:00+00', 'เหล็กรางน้ำ 150x75x6.5x10x6ม.', 'system', now(), 'system', now()),
  ('fb17a2c5-17c2-5366-94ee-babdee4a71a6', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_12' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'WFB2005.506', 'PG01_11|PG02_2|PG03_1|PG04_92|PG07_1|PG08_2|PG09_3', false, 0, 0, 0, 0, 0, 0, 25.8, 0, 0, 0, 0, 0, 0, 25.8, '2026-01-01 00:00:00+00', 'เหล็กไวแฟรงค์ 200x100x5.5x8x6ม.', 'system', now(), 'system', now()),
  ('6bef1849-896d-5874-b7ab-2e4c79af74b5', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_12' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'WFB250606', 'PG01_11|PG02_2|PG03_1|PG04_100|PG07_1|PG08_2|PG09_3', false, 0, 0, 0, 0, 0, 0, 25.8, 0, 0, 0, 0, 0, 0, 25.8, '2026-01-01 00:00:00+00', 'เหล็กไวแฟรงค์ 250x125x6x9x6ม.', 'system', now(), 'system', now()),
  ('636e4226-4d5f-5a5a-8b74-c7b0fd77f741', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_12' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'WFB350706', 'PG01_11|PG02_2|PG03_1|PG04_113|PG07_1|PG08_2|PG09_3', false, 0, 0, 0, 0, 0, 0, 25.8, 0, 0, 0, 0, 0, 0, 25.8, '2026-01-01 00:00:00+00', 'เหล็กไวแฟรงค์ 350x175x7x11x6ม.', 'system', now(), 'system', now()),
  ('a035ba5f-92b2-5f8c-b44b-51ffb4512524', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_12' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'WFB400806', 'PG01_11|PG02_2|PG03_1|PG04_119|PG07_1|PG08_2|PG09_3', false, 0, 0, 0, 0, 0, 0, 25.8, 0, 0, 0, 0, 0, 0, 25.8, '2026-01-01 00:00:00+00', 'เหล็กไวแฟรงค์ 400x200x8x13x6ม.', 'system', now(), 'system', now()),
  ('39c60728-3a69-5a5b-88cb-319f2c19908e', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_12' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'HBB150706', 'PG01_11|PG02_1|PG03_1|PG04_99|PG07_1|PG08_2|PG09_3', false, 0, 0, 0, 0, 0, 0, 25.8, 0, 0, 0, 0, 0, 0, 25.8, '2026-01-01 00:00:00+00', 'เหล็กเอชบีม 150x150x7x10x6ม.', 'system', now(), 'system', now()),
  ('5767b391-9d4c-5664-9dc6-0747ff379acd', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_12' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'HBB200806', 'PG01_11|PG02_1|PG03_1|PG04_109|PG07_1|PG08_2|PG09_3', false, 0, 0, 0, 0, 0, 0, 25.8, 0, 0, 0, 0, 0, 0, 25.8, '2026-01-01 00:00:00+00', 'เหล็กเอชบีม 200x200x8x12x6ม.', 'system', now(), 'system', now()),
  ('df5610b0-4965-5073-817d-0e761d6664f9', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_12' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'HBB250906', 'PG01_11|PG02_1|PG03_1|PG04_118|PG07_1|PG08_2|PG09_3', false, 0, 0, 0, 0, 0, 0, 25.8, 0, 0, 0, 0, 0, 0, 25.8, '2026-01-01 00:00:00+00', 'เหล็กเอชบีม 250x250x9x14x6ม.', 'system', now(), 'system', now()),
  ('46b3758c-5c21-5161-a3d2-091ff0a52f11', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_2' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLB10050202.3TIS', 'PG01_7|PG03_5|PG04_59|PG06_18|PG07_1|PG09_1', false, 0, 0, 0, 0, 0, 0, 18.8, 0, 0, 0, 0, 0, 0, 18.8, '2026-01-01 00:00:00+00', 'เหล็กตัวซี (มอก.) 100x50x20x2.3x6ม.', 'system', now(), 'system', now()),
  ('1903c66c-59d3-526e-b886-d5e793687ee4', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_2' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLB15050202.3TIS', 'PG01_7|PG03_5|PG04_63|PG06_18|PG07_1|PG09_1', false, 0, 0, 0, 0, 0, 0, 18.8, 0, 0, 0, 0, 0, 0, 18.8, '2026-01-01 00:00:00+00', 'เหล็กตัวซี (มอก.) 150x50x20x2.3x6ม', 'system', now(), 'system', now()),
  ('a0ff3759-63b7-5fde-9e9b-d898dc83fd7e', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_2' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLB10050203.2TIS', 'PG01_7|PG03_5|PG04_59|PG06_29|PG07_1|PG09_1', false, 0, 0, 0, 0, 0, 0, 18.8, 0, 0, 0, 0, 0, 0, 18.8, '2026-01-01 00:00:00+00', 'เหล็กตัวซี (มอก.)  100x50x20x3.2x6ม.', 'system', now(), 'system', now()),
  ('a47724a9-7536-5658-834f-0264b037528f', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_2' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLB15050203.2TIS', 'PG01_7|PG03_5|PG04_63|PG06_29|PG07_1|PG09_1', false, 0, 0, 0, 0, 0, 0, 18.8, 0, 0, 0, 0, 0, 0, 18.8, '2026-01-01 00:00:00+00', 'เหล็กตัวซี (มอก.)  150x50x20x3.2x6ม.', 'system', now(), 'system', now()),
  ('2ef73090-2583-586a-b3c7-8c0dd91e3d4a', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PSB382.3TIS', 'PG01_4|PG02_11|PG03_2|PG04_18|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อสี่เหลี่ยม (มอก.) 38x38x2.3x6ม.', 'system', now(), 'system', now()),
  ('a7fd55ec-b173-5c42-bee0-90bd8da54cda', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PSB502.3TIS', 'PG01_4|PG02_11|PG03_2|PG04_20|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อสี่เหลี่ยม (มอก.) 50x50x2.3x6ม.', 'system', now(), 'system', now()),
  ('0eacec0b-d47b-548b-94fe-8303001e6dca', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PSB752.3TIS', 'PG01_4|PG02_11|PG03_2|PG04_24|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อสี่เหลี่ยม (มอก.) 75x75x2.3x6ม.', 'system', now(), 'system', now()),
  ('32d0781f-88a0-550b-913f-eef58ea185bd', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PSB383.2TIS', 'PG01_4|PG02_11|PG03_2|PG04_18|PG06_29|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อสี่เหลี่ยม (มอก.) 38x38x3.2x6ม.', 'system', now(), 'system', now()),
  ('284eed97-776e-5fa3-9b5b-8235d90d8052', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PSB503.2TIS', 'PG01_4|PG02_11|PG03_2|PG04_20|PG06_29|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อสี่เหลี่ยม (มอก.) 50x50x3.2x6ม.', 'system', now(), 'system', now()),
  ('e966cfe7-32fa-5b6f-befa-ccfae87adf1f', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PSB753.2TIS', 'PG01_4|PG02_11|PG03_2|PG04_24|PG06_29|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อสี่เหลี่ยม (มอก.) 75x75x3.2x6ม.', 'system', now(), 'system', now()),
  ('828bb7ae-20a2-5ed4-bea0-4d0600b745c9', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PRB75382.3TIS', 'PG01_4|PG02_13|PG03_2|PG04_21|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อแบน (มอก.) 75x38x2.3x6ม.', 'system', now(), 'system', now()),
  ('976285f4-d9c4-5fce-ac1d-f25ddecd132d', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PRB100502.3TIS', 'PG01_4|PG02_13|PG03_2|PG04_23|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อแบน (มอก.) 100x50x2.3x6ม.', 'system', now(), 'system', now()),
  ('747c8d3f-df64-555c-90bf-a1fc690cec0f', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PRB75383.2TIS', 'PG01_4|PG02_13|PG03_2|PG04_21|PG06_29|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อแบน (มอก.) 75x38x3.2x6ม.', 'system', now(), 'system', now()),
  ('93eb30db-9f9e-533a-9c7b-9caefe75820c', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PRB100503.2TIS', 'PG01_4|PG02_13|PG03_2|PG04_23|PG06_29|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อแบน (มอก.) 100x50x3.2x6ม.', 'system', now(), 'system', now()),
  ('ee910ece-9199-5d5e-ac27-9d3f25af5086', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'POB343.2TIS', 'PG01_4|PG02_16|PG03_2|PG04_2|PG06_29|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อดำ (มอก.)  34x3.2x6 ม.', 'system', now(), 'system', now()),
  ('ebc468d9-3305-5644-a1c9-894f929aadd6', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'POB42.72.3TIS', 'PG01_4|PG02_16|PG03_2|PG04_3|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อดำ (มอก.)  42.7x2.3x6ม.', 'system', now(), 'system', now()),
  ('5e9b8b5e-626f-5790-943a-3c0e47c595f3', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'POB48.62.3TIS', 'PG01_4|PG02_16|PG03_2|PG04_4|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อดำ (มอก.)  48.6x2.3x6ม.', 'system', now(), 'system', now()),
  ('970f9e8a-43bb-5f9d-bb24-ab5f3639f355', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'POB76.32.3TIS', 'PG01_4|PG02_16|PG03_2|PG04_6|PG06_18|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อดำ (มอก.)  76.3x2.3x6ม.', 'system', now(), 'system', now()),
  ('c22e7f79-153c-572e-826c-be2717b88033', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_4' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'POB101.63.2TIS', 'PG01_4|PG02_16|PG03_2|PG04_8|PG06_29|PG07_1', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'ท่อดำ (มอก.) 101.6x3.2x6ม.', 'system', now(), 'system', now()),
  ('fedf9d63-780b-5575-aafe-5c7bdf624348', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_5' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PRZ50251.3', 'PG01_4|PG02_14|PG03_1|PG04_17|PG06_5|PG07_1|PG08_5', false, 0, 0, 0, 0, 0, 0, 21.6, 0, 0, 0, 0, 0, 0, 21.6, '2026-01-01 00:00:00+00', 'ท่อแบน  50x25x1.3x6ม. GI', 'system', now(), 'system', now()),
  ('1bb189b0-8eed-5cdc-8c15-7eef51631ce0', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_5' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PRZ50251.5', 'PG01_4|PG02_14|PG03_1|PG04_17|PG06_8|PG07_1|PG08_5', false, 0, 0, 0, 0, 0, 0, 21.6, 0, 0, 0, 0, 0, 0, 21.6, '2026-01-01 00:00:00+00', 'ท่อแบน  50x25x1.5x6ม. GI', 'system', now(), 'system', now()),
  ('b689ca21-8de7-5f14-87f4-e7634f8c04df', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_5' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'PRZ75381.3', 'PG01_4|PG02_14|PG03_1|PG04_21|PG06_5|PG07_1|PG08_5', false, 0, 0, 0, 0, 0, 0, 21.6, 0, 0, 0, 0, 0, 0, 21.6, '2026-01-01 00:00:00+00', 'ท่อแบน  75x38x1.3x6ม. GI', 'system', now(), 'system', now()),
  ('c8550606-cc2d-56f6-a9cd-f01268895e9b', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_3' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLZ751.3', 'PG01_4|PG02_15|PG03_1|PG04_51|PG06_5|PG07_1', false, 0, 0, 0, 0, 0, 0, 20.4, 0, 0, 0, 0, 0, 0, 20.4, '2026-01-01 00:00:00+00', 'เหล็กตัวซี 75x45x15x1.3x6ม  GI', 'system', now(), 'system', now()),
  ('80542bea-d323-5dc7-bc28-97e0dec7652f', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_3' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLZ751.7', 'PG01_4|PG02_15|PG03_1|PG04_51|PG06_10|PG07_1', false, 0, 0, 0, 0, 0, 0, 20.4, 0, 0, 0, 0, 0, 0, 20.4, '2026-01-01 00:00:00+00', 'เหล็กตัวซี 75x45x15x1.7x6ม. GI', 'system', now(), 'system', now()),
  ('d485df5b-e32f-5d8b-b8a3-9eadbce9a137', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_3' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLZ1252.0', 'PG01_4|PG02_15|PG03_1|PG04_62|PG06_13|PG07_1', false, 0, 0, 0, 0, 0, 0, 20.4, 0, 0, 0, 0, 0, 0, 20.4, '2026-01-01 00:00:00+00', 'เหล็กตัวซี 125x50x20x2.0x6ม. GI', 'system', now(), 'system', now()),
  ('f81cd376-f99d-57af-b935-a2e822972fe2', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_3' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'CLZ1002.0', 'PG01_4|PG02_15|PG03_1|PG04_59|PG06_13|PG07_1', false, 0, 0, 0, 0, 0, 0, 20.4, 0, 0, 0, 0, 0, 0, 20.4, '2026-01-01 00:00:00+00', 'เหล็กตัวซี 100x50x20x2.0x6ม.GI', 'system', now(), 'system', now()),
  ('f7bc5629-03e4-5817-acb5-e969f897904a', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_9' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'RBB610SR24IF', 'PG01_1|PG02_3|PG03_23|PG06_46|PG07_3|PG09_7', false, 0, 0, 0, 0, 0, 0, 18.8, 0, 0, 0, 0, 0, 0, 18.8, '2026-01-01 00:00:00+00', 'เหล็กเส้นกลม  6 มม.x10 ม. SR24 IF', 'system', now(), 'system', now()),
  ('47213b3b-aa2c-52bb-b7ab-3623cb1e9c38', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_9' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'RBB610SR24TIF', 'PG01_1|PG02_3|PG03_6|PG06_46|PG07_3|PG09_7', false, 0, 0, 0, 0, 0, 0, 18.8, 0, 0, 0, 0, 0, 0, 18.8, '2026-01-01 00:00:00+00', 'เหล็กเส้นกลม  6 มม.x10 ม. SR24 ตรง IF', 'system', now(), 'system', now()),
  ('8ec762a8-0314-5930-8620-5adcbff94c16', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_9' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'DBB1210SD40IF', 'PG01_1|PG02_10|PG03_23|PG06_58|PG07_3|PG09_7', false, 0, 0, 0, 0, 0, 0, 17.7, 0, 0, 0, 0, 0, 0, 17.7, '2026-01-01 00:00:00+00', 'เหล็กข้ออ้อย 12มมx10ม. SD40 IF', 'system', now(), 'system', now()),
  ('825a04fc-dced-5cf6-a2de-fa4022313aff', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_9' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'DBB2510SD40IF', 'PG01_1|PG02_10|PG03_23|PG06_64|PG07_3|PG09_7', false, 0, 0, 0, 0, 0, 0, 17.7, 0, 0, 0, 0, 0, 0, 17.7, '2026-01-01 00:00:00+00', 'เหล็กข้ออ้อย 25มมx10ม. SD40 IF', 'system', now(), 'system', now()),
  ('0799edf5-c870-5938-b3c0-090299ea7f9c', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_9' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'DBB1210SD40TIF', 'PG01_1|PG02_10|PG03_6|PG06_58|PG07_3|PG09_7', false, 0, 0, 0, 0, 0, 0, 17.7, 0, 0, 0, 0, 0, 0, 17.7, '2026-01-01 00:00:00+00', 'เหล็กข้ออ้อย 12มมx10ม.SD40 ตรง IF', 'system', now(), 'system', now()),
  ('d274eadd-51cb-5865-84d3-deb237c1a3a0', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_9' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'DBB2510SD40TIF', 'PG01_1|PG02_10|PG03_6|PG06_64|PG07_3|PG09_7', false, 0, 0, 0, 0, 0, 0, 17.7, 0, 0, 0, 0, 0, 0, 17.7, '2026-01-01 00:00:00+00', 'เหล็กข้ออ้อย 25มมx10ม.SD40 ตรง IF', 'system', now(), 'system', now()),
  ('739f3c9b-1c07-5db6-8a79-d6ea9c17fc77', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_9' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'DBB1212SD40IF', 'PG01_1|PG02_10|PG03_23|PG06_58|PG07_4|PG09_7', false, 0, 0, 0, 0, 0, 0, 17.7, 0, 0, 0, 0, 0, 0, 17.7, '2026-01-01 00:00:00+00', 'เหล็กข้ออ้อย 12มมx12ม. SD40 IF', 'system', now(), 'system', now()),
  ('c716c2c5-3a8d-5f84-9c7d-48a3728559a3', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_8' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'COZ1.14', 'PG01_9|PG03_15|PG05_16|PG06_3|PG08_13|PG09_8', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'เหล็กม้วนซิงค์  1.10มมx4''xCoil', 'system', now(), 'system', now()),
  ('6f9ee63e-9b82-5336-95be-818d45cfbab9', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_8' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'COZ1.34', 'PG01_9|PG03_15|PG05_16|PG06_5|PG08_13|PG09_8', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'เหล็กม้วนซิงค์  1.30มมx4''xCoil', 'system', now(), 'system', now()),
  ('74d8d258-b512-5b4b-8e34-f1cf25cb89cf', (SELECT id FROM public.price_list_group WHERE group_code='GROUP_1_ITEM_8' AND site_code='TMI_WH' AND company_code='09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3' LIMIT 1), 'COZ1.74', 'PG01_9|PG03_15|PG05_16|PG06_10|PG08_13|PG09_8', false, 0, 0, 0, 0, 0, 0, 19.7, 0, 0, 0, 0, 0, 0, 19.7, '2026-01-01 00:00:00+00', 'เหล็กม้วนซิงค์  1.7มมx4''xCoil', 'system', now(), 'system', now());

-- ---------------------------------------------------------------------------
-- 3) INSERT price_list_sub_group_key (sub_group_id resolved from fresh rows)
-- ---------------------------------------------------------------------------
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'eb1c98b6-9304-5271-a9d4-9a71d66e3805', sg.id, 'PG01', 'PG01_3', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB3.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5e4be221-2b62-50d8-85d4-74279cc277c0', sg.id, 'PG02', 'PG02_6', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB3.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a9ab4d53-cba0-5d04-8a1a-491b9770b06a', sg.id, 'PG03', 'PG03_8', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB3.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6ac756ad-de3b-5991-b1aa-291435d390e1', sg.id, 'PG05', 'PG05_3', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB3.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '848fea4b-7043-5e27-9fde-7be1ea49cc0d', sg.id, 'PG06', 'PG06_33', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB3.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0244d6c3-22bb-52cb-8139-7f004ce504c0', sg.id, 'PG01', 'PG01_3', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3c27648e-462d-5405-a6af-952dc3885bbf', sg.id, 'PG02', 'PG02_6', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '72f6ce28-9385-5e5b-aeb3-584f7fbbfe5f', sg.id, 'PG03', 'PG03_8', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '9e8b5342-1fed-50af-8bc2-5791f8acd9dd', sg.id, 'PG05', 'PG05_3', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'bbc76175-48da-5118-b6df-433b3f64dc8f', sg.id, 'PG06', 'PG06_45', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.848';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ddf3bbf2-3613-5013-9b34-7d4534fe282e', sg.id, 'PG01', 'PG01_3', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8510';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5b6f537d-3a57-5617-a6de-f7c0c61871cd', sg.id, 'PG02', 'PG02_6', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8510';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0b82d375-b052-5ffd-b390-7031d0c12266', sg.id, 'PG03', 'PG03_8', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8510';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '58144c7c-9bcd-5898-862f-903a937e8249', sg.id, 'PG05', 'PG05_4', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8510';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '72d5b402-69ac-5be5-ac25-24574096346c', sg.id, 'PG06', 'PG06_45', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8510';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3845732d-6730-55ef-9faa-2c428d703423', sg.id, 'PG01', 'PG01_3', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c6455691-c744-5f60-b171-53ed78e8394c', sg.id, 'PG02', 'PG02_6', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '83ef5204-c4a4-5f20-ba4a-df12202a03e0', sg.id, 'PG03', 'PG03_8', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'cfdec86a-69ff-5264-8a89-46507c0b66ec', sg.id, 'PG05', 'PG05_5', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '48ef1735-f230-5473-b332-c5f55cf117e4', sg.id, 'PG06', 'PG06_45', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB5.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0f901eec-dfa5-52c0-9532-a75b4000ea52', sg.id, 'PG01', 'PG01_3', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB11.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '660217d0-2bcb-5fe6-b633-cd3ea4997ff1', sg.id, 'PG02', 'PG02_6', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB11.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c29132ef-2f51-54f1-b42c-2b971a87d0d8', sg.id, 'PG03', 'PG03_8', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB11.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'bc203ba8-87e7-5e2c-b035-99112b83ea2a', sg.id, 'PG05', 'PG05_5', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB11.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ad1ac551-fdee-59de-adfc-1a6306adc226', sg.id, 'PG06', 'PG06_58', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTB11.8520';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '647a7ce1-8dab-5dde-ad9d-2eae1ff2ab89', sg.id, 'PG01', 'PG01_3', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH3.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4535c8cc-78e8-541b-9ac1-4d783c459653', sg.id, 'PG02', 'PG02_9', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH3.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3ef23d8a-efed-5a1f-8332-3ea13906a26f', sg.id, 'PG03', 'PG03_21', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH3.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1c112b2b-5a67-5852-81c7-cd6e697b5b68', sg.id, 'PG05', 'PG05_3', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH3.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '15c34688-deec-5c61-b165-e99a8ea75065', sg.id, 'PG06', 'PG06_33', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH3.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '56a919c4-1dfd-537e-b6cc-da48af36d9de', sg.id, 'PG01', 'PG01_3', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH5.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '54de9bc2-4df3-504d-a53e-cce830fe42da', sg.id, 'PG02', 'PG02_9', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH5.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '77fb7aef-4970-58e2-a115-556c38e81200', sg.id, 'PG03', 'PG03_21', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH5.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0fca6989-e85c-57bf-b92e-6cd7040ae843', sg.id, 'PG05', 'PG05_3', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH5.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5dc5a5e2-8faf-589a-95da-48b8bf14f40f', sg.id, 'PG06', 'PG06_45', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PTH5.848I';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'cb58d289-2c5f-5c27-9a8e-5f68c25f5154', sg.id, 'PG01', 'PG01_6', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB7556';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '8796ce48-5b81-538c-bed1-7fcd5e7592c6', sg.id, 'PG02', 'PG02_8', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB7556';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2021b591-bea2-54a8-b2b5-b8fad625c2c7', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB7556';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5bf58c04-a645-50f4-8149-e172c6ba5f46', sg.id, 'PG04', 'PG04_83', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB7556';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5683ab06-c0a4-5d15-af5c-eff9cb8a1ddf', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB7556';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '50160004-197f-53e6-84f0-dfd102c8d197', sg.id, 'PG01', 'PG01_6', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB1506.56';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '9ffc887e-dca1-502a-bb7e-be818c39a29d', sg.id, 'PG02', 'PG02_8', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB1506.56';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3dc3243b-9a4c-5074-9874-6448f7588961', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB1506.56';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '941caf5e-719a-5b25-8255-c7d91f85f003', sg.id, 'PG04', 'PG04_90', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB1506.56';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '932a7dbc-96f5-571d-bf86-fdf2ccc9b5b9', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'UBB1506.56';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ba380faa-51a5-5773-b3fe-d3167e6a80ec', sg.id, 'PG01', 'PG01_11', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB2005.506';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'fad87a2b-7ed9-5cdb-9694-e8c6b7148bac', sg.id, 'PG02', 'PG02_2', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB2005.506';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3d9e7f2e-caee-560f-a13f-ecb1aac04787', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB2005.506';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0febec87-2b9e-5af9-92f2-3716db01f381', sg.id, 'PG04', 'PG04_92', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB2005.506';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ee1714a7-1947-54cb-8872-6a0767eef8e0', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB2005.506';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '715ff430-992d-56a0-bdaa-957a299ac49f', sg.id, 'PG08', 'PG08_2', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB2005.506';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '8b9ced37-40bd-5bc4-b0aa-64d0808a1602', sg.id, 'PG09', 'PG09_3', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB2005.506';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'cf0b82f5-7901-5cd9-ab6d-653f7190b3e3', sg.id, 'PG01', 'PG01_11', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB250606';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'eb8f621f-c79f-5c3c-acf6-befc3c992aca', sg.id, 'PG02', 'PG02_2', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB250606';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'caf0c113-743c-571b-81ad-044c527c13d3', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB250606';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'eef4c247-3933-5615-81da-1c8292ec7d75', sg.id, 'PG04', 'PG04_100', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB250606';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '9271ef0b-237b-50a5-93ad-dc83806f602f', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB250606';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '423761d7-be91-592c-954c-6a73d3d83adb', sg.id, 'PG08', 'PG08_2', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB250606';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '28d8ab99-a638-5bea-b5bb-3b8dcf2d257d', sg.id, 'PG09', 'PG09_3', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB250606';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c046656d-3eff-5679-af8e-6e960a897133', sg.id, 'PG01', 'PG01_11', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB350706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c0780ac8-03db-5c39-ba73-4ec0d85a9f82', sg.id, 'PG02', 'PG02_2', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB350706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3727ce0b-86f4-58ce-bc64-03f924bbe204', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB350706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'fa0b0010-18fd-5b89-aade-063207f270ef', sg.id, 'PG04', 'PG04_113', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB350706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f4520c2c-75d8-5d48-a135-26457db229a4', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB350706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4a0ec209-efd8-5a47-999b-ec360334b2c6', sg.id, 'PG08', 'PG08_2', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB350706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3f4ba560-26c3-5df9-99cd-6b10abb606b7', sg.id, 'PG09', 'PG09_3', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB350706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'cc67ea36-f6dd-5906-960c-43e3d3af632a', sg.id, 'PG01', 'PG01_11', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB400806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a69e57a8-db36-52ce-bf33-da5e915c13e1', sg.id, 'PG02', 'PG02_2', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB400806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4195bb86-1c12-5d3a-a515-46b1cd86c16a', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB400806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b109a2fd-cdf5-572c-b5cc-091f30245743', sg.id, 'PG04', 'PG04_119', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB400806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6a7f2a65-0f8d-5c33-b44f-73ca8ccc0803', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB400806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a03b8dd2-8295-5346-aef9-51d53d2d04f1', sg.id, 'PG08', 'PG08_2', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB400806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5700ee80-c9bd-5e71-addd-472988b830d7', sg.id, 'PG09', 'PG09_3', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'WFB400806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd437097b-0a46-502e-a103-2760b1b35c54', sg.id, 'PG01', 'PG01_11', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB150706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '243c356c-9d14-5310-9455-1442c299e06f', sg.id, 'PG02', 'PG02_1', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB150706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'db762116-4897-587b-992a-0f07bff80fe9', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB150706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ca747b3d-c950-5d58-a14f-e3f86a6197be', sg.id, 'PG04', 'PG04_99', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB150706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '55c372eb-98c8-5f4e-a661-7d21d85c55e3', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB150706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3296ee38-0d1a-5751-b553-29dd696bc728', sg.id, 'PG08', 'PG08_2', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB150706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6da499da-6273-5c65-bb6c-880dafc770ae', sg.id, 'PG09', 'PG09_3', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB150706';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '43cabdbf-4202-5ec3-97f1-aba1fc6286ef', sg.id, 'PG01', 'PG01_11', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB200806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3b8314ca-5645-50e3-97cd-c12f7389042c', sg.id, 'PG02', 'PG02_1', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB200806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f44e346d-34d5-5cdb-97df-0b9842916d54', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB200806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '75663a02-bc1a-57be-a6b2-224f1396e2f4', sg.id, 'PG04', 'PG04_109', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB200806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '69cf88df-5393-566c-a075-582feb00d9c9', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB200806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'fec56081-c148-517b-90b4-0dc12ebce5d7', sg.id, 'PG08', 'PG08_2', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB200806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '350dffb3-f4da-57d6-88f5-c5abf63c5842', sg.id, 'PG09', 'PG09_3', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB200806';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '13815ee7-320b-5364-957a-b7d2fde37423', sg.id, 'PG01', 'PG01_11', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB250906';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f2b9cd15-148a-54a1-b24f-70507399ba71', sg.id, 'PG02', 'PG02_1', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB250906';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0006cd0a-18f8-5825-a539-4b3387e104b6', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB250906';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ac2037b5-8b99-5605-921f-5bb798df5791', sg.id, 'PG04', 'PG04_118', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB250906';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'e424d75b-fcd2-56da-9697-0e7fc26b7f99', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB250906';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3b59c553-6515-555f-9f07-653db4f4457d', sg.id, 'PG08', 'PG08_2', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB250906';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'aab46538-97ca-5ce3-89d7-91730a42f72a', sg.id, 'PG09', 'PG09_3', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'HBB250906';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3db54198-7949-51d9-b0aa-ec822a1f4095', sg.id, 'PG01', 'PG01_7', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3d4110c3-7bd5-551e-98b7-631a34cbd7f4', sg.id, 'PG03', 'PG03_5', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'e4e36b57-3588-53f1-8ded-b348d2f5764a', sg.id, 'PG04', 'PG04_59', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4810a30d-bd07-515f-a816-5724c5123e29', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd1203f71-dde5-5783-a86a-916caeeca256', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f489565f-568c-5f0c-a506-fe98ee91aa45', sg.id, 'PG09', 'PG09_1', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6b5dddfe-b1c9-5d37-b0af-f85b096e0719', sg.id, 'PG01', 'PG01_7', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ddf7cda7-e3cb-5a0d-b349-3343cd29ffdc', sg.id, 'PG03', 'PG03_5', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c16014c7-38bb-55c8-a542-f01dc92f6083', sg.id, 'PG04', 'PG04_63', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1cf694c2-f8de-5e07-8773-47b36bf646df', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7e37b232-75ff-5bdf-855c-f0e093b71a07', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd637b5c0-215a-5d91-8ccb-9dd23e958cde', sg.id, 'PG09', 'PG09_1', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050202.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '733a4652-706b-5e40-ab6d-9cf50b8310ed', sg.id, 'PG01', 'PG01_7', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '9358a3f3-7e8e-519d-b02f-268b5f9917f8', sg.id, 'PG03', 'PG03_5', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a3b7fd6c-bb38-55d2-b6ad-5728cf52bfec', sg.id, 'PG04', 'PG04_59', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f3ecf2bc-b319-5aa2-b89b-1a5aacbadf42', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '77f893a6-89ef-572a-91e7-701312b4e90b', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'fd6c7c37-bf41-525d-97a8-7aa9e37219e9', sg.id, 'PG09', 'PG09_1', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB10050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0b8a5c2d-3c6e-55b0-ba4b-e17e8a3dae67', sg.id, 'PG01', 'PG01_7', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '519451ac-2503-5c21-83fc-6422633c0c5a', sg.id, 'PG03', 'PG03_5', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c59b75ce-7587-5412-af05-3f6b976d57fb', sg.id, 'PG04', 'PG04_63', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '664fc5f9-fa09-5713-a774-c0e075f06d2b', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ab166d60-5d02-5709-983e-caa70bb2647e', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5e68649a-a3eb-5d8a-90c4-3b6ebfea5c87', sg.id, 'PG09', 'PG09_1', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLB15050203.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6d1959d7-4e52-551e-ac23-b015260956b0', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7a1c60fa-5fac-5a60-8917-d05780985fe0', sg.id, 'PG02', 'PG02_11', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c76d5eca-3785-51b9-b988-30e5f1be3b5c', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'aa0f53e3-5e82-5cb5-b1eb-b4ad2c083bb3', sg.id, 'PG04', 'PG04_18', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1fab7d53-73b3-53f7-99c6-dc693f23f4b3', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '91459d2b-96d7-58e6-99aa-102ba0b0bbf2', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f4496897-485d-57c3-82b7-3aaa16168bb9', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '754fc9cf-49ac-59bb-b2ab-a66e77782f31', sg.id, 'PG02', 'PG02_11', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'df41ec7f-952f-5921-97c7-f0ea1a0c2f55', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '10d419b7-1d9e-5477-8f91-557d15a43b2a', sg.id, 'PG04', 'PG04_20', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0c21e798-08cf-5d1d-af36-9556de31ce29', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd63e77b0-4507-5ad5-84b4-a5e701092784', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '287c1e4c-d12f-5ada-8052-75515a8eeacc', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB752.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'e999e1da-6f96-55b9-88ff-cc78e2d3f1a2', sg.id, 'PG02', 'PG02_11', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB752.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0b1e42e1-d7f2-5759-a508-f02ec69c9e43', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB752.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd34425d2-50d8-53b8-9cbf-28ea6d653913', sg.id, 'PG04', 'PG04_24', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB752.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2a56ef76-9ddc-5089-825e-600f57d94bfd', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB752.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ec0932ff-030d-58eb-bc15-4212c59d37f5', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB752.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '04888646-4ddb-58fb-b07c-9df7a6bd3340', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5ff110c5-04da-5e09-a860-5c905905b8eb', sg.id, 'PG02', 'PG02_11', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5c6002cc-c5cb-5fc5-b496-7dc12da04cf6', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c420d9ca-5264-5729-bd88-17c1112674dd', sg.id, 'PG04', 'PG04_18', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6d0a543a-bfb1-542c-baa8-447abbd4a94b', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '27d05b9a-975f-561c-9d70-4cffcd065957', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '842d5158-26f2-5de8-9e65-c7ffed12057c', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5e7b15f9-43c0-5723-b6a3-0ffdd9bbe2d5', sg.id, 'PG02', 'PG02_11', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'caf918da-bc1c-5333-b737-f528c215aad2', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd225a93a-d62d-5d9d-9af1-59d7df0774b4', sg.id, 'PG04', 'PG04_20', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2e6c3849-e266-525b-91ba-c497ad09c367', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '58d4e276-c007-5440-8f3f-74c14687bfa2', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '90a6032b-35b2-5e3f-a75a-b8ef4f8d4824', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB753.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c9d1b23e-d3fa-577a-a7b0-571abdfd6eac', sg.id, 'PG02', 'PG02_11', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB753.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '432f0368-5eba-537c-a74a-5930823134db', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB753.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a15038aa-3285-51e9-91c3-ccac6f53bd41', sg.id, 'PG04', 'PG04_24', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB753.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b72b16e0-b23a-50ca-8727-5a1b9f29bee1', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB753.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2e68aaed-3fa6-5640-8822-40ad23793829', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PSB753.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3b5de90b-4e29-5d0d-aed8-781752c2ae5e', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '618861be-b951-50e4-9fc7-bdca1d3cc049', sg.id, 'PG02', 'PG02_13', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'fb9b2f09-edb0-55e3-ad2a-ee04693efb9c', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ca818e05-4241-5c18-b623-1259bbf0fab5', sg.id, 'PG04', 'PG04_21', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '904e7c2c-5b15-582d-9fb9-824406b25f08', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7fd9d0d7-4e99-581f-b2f2-92534fcc90c2', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75382.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6de18a73-3881-5142-bdca-8996dd679128', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '27bc773b-f16c-5d63-8938-904a0ccd4260', sg.id, 'PG02', 'PG02_13', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '517cc598-4d93-51a4-90c8-c3e4e6039ce0', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6b946940-7e20-56bd-b874-8170bc0f4cbd', sg.id, 'PG04', 'PG04_23', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '90201523-6626-52a3-99a5-28bd1c1664a9', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '20d8f604-6f5d-5c47-a7cb-c94a70654813', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100502.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4c7c0907-0013-57ae-8c8a-a0c7c8ee6365', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '59dc4f5b-0336-528f-90bc-6009cb4c65e3', sg.id, 'PG02', 'PG02_13', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1cb0d5de-608d-5014-bf00-a20c1eaad096', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3250faec-41ff-55aa-884c-a24300b0c9e8', sg.id, 'PG04', 'PG04_21', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '934021b6-9ced-529c-91e9-7b772e4d01d1', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '38941412-5fe2-5d41-8041-568d0b8560aa', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB75383.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ce5875b9-2078-5b5c-b99b-a690cd36615a', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4e78c78c-3a46-540e-ac50-a4b64b0841bd', sg.id, 'PG02', 'PG02_13', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3b90a569-fdd6-5763-a6e5-e31c165c37f9', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '973beee6-4417-5828-b647-9952703e7c41', sg.id, 'PG04', 'PG04_23', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2df4199c-b5d4-589e-82f5-9d5b84b8046d', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1aa13654-21ad-507e-a0e8-9408d353e89c', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRB100503.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ee5a215e-39fe-5ad5-a212-72d9ca176cc7', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB343.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b6cc857d-a939-5664-bbe0-1ece91f8c136', sg.id, 'PG02', 'PG02_16', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB343.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '733b0709-4a8f-5f23-bcbc-cd8debb48035', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB343.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '786143d7-162b-5d47-910d-1fb823e1b016', sg.id, 'PG04', 'PG04_2', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB343.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b818d4c0-8554-5396-a767-93a134385f91', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB343.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c42ba98a-9ce7-5040-b7da-78ea1a57cb62', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB343.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4062336e-f1f2-5d9f-83ab-aa32a33d6398', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB42.72.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '9a5cefc9-01a3-5225-aff5-7762bffd12c4', sg.id, 'PG02', 'PG02_16', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB42.72.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3085ee2c-1edf-5ceb-87e3-d0f2ddfbadb3', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB42.72.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '9484e8fe-61fd-5e76-9032-d191a45ef6c8', sg.id, 'PG04', 'PG04_3', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB42.72.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '310a2265-58e8-5e59-a3a2-ded392cc9998', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB42.72.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0409d3ff-7521-5b07-a27a-68141468d1c3', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB42.72.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f6174270-47b6-517d-ac31-8f1f87900390', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB48.62.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a007632b-b873-5f73-ae73-37a53b5ead2a', sg.id, 'PG02', 'PG02_16', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB48.62.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'cedeb890-9a16-5f44-a48f-ea8a4aedb1f0', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB48.62.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd7f69225-99ac-5923-af4b-62605283651e', sg.id, 'PG04', 'PG04_4', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB48.62.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '80558bb3-d140-5140-bcc5-459cf565ba70', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB48.62.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a982a866-957d-52f3-9e36-3b5f1bc9cae7', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB48.62.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '504f39a0-16e9-5fa7-841b-63d7e102807a', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB76.32.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7680e27a-1ba2-5525-8731-3ce1fce36048', sg.id, 'PG02', 'PG02_16', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB76.32.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd11ca30e-18ec-570f-9e6c-e0524164f16a', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB76.32.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7120bb48-944a-5b80-9ef4-e796764016f4', sg.id, 'PG04', 'PG04_6', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB76.32.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'afbb5e81-467a-5d00-a69c-8710221568ce', sg.id, 'PG06', 'PG06_18', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB76.32.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a651014b-deda-50fd-82ed-c1089d02a1f5', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB76.32.3TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '041245b3-7b13-54b3-917a-e2f3310ac860', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB101.63.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b28b1712-856c-553b-9c75-0abcfef1f5ad', sg.id, 'PG02', 'PG02_16', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB101.63.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '15b83f0f-68e2-59fe-9abe-d50b47d7f210', sg.id, 'PG03', 'PG03_2', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB101.63.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '03a852a4-f4b8-598f-b5ee-1af4aed0da92', sg.id, 'PG04', 'PG04_8', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB101.63.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '099cffe4-1319-5994-a8fc-3c75369a742f', sg.id, 'PG06', 'PG06_29', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB101.63.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0e32b455-c651-58b6-aeb1-77d4ede7fd43', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'POB101.63.2TIS';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4d4d4ad4-4759-5d0a-8694-1f7707352b57', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'adb0cf33-fc57-512c-8bcd-475498299b63', sg.id, 'PG02', 'PG02_14', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ffba8e0e-9036-5d4d-a5c9-5b05ca15c655', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '34cf1f20-1582-55fc-8eae-d6c683c4011e', sg.id, 'PG04', 'PG04_17', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0bf7feb8-e607-51a4-b2ad-76d7056555d4', sg.id, 'PG06', 'PG06_5', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3cb21268-158f-5920-8955-072dc08276bc', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3f45b54b-2d7a-5cd7-ba9d-6ac8f542a1a5', sg.id, 'PG08', 'PG08_5', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'fe301f42-0aaf-5a3e-83fb-9c4a0bb81ad8', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.5';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1be177a1-4067-5615-b0f9-a5dde08730f4', sg.id, 'PG02', 'PG02_14', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.5';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '34c2f25c-4d0d-5dc4-8161-144abe3581dc', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.5';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '13120c0c-1167-5e48-875c-95f5bac9ae27', sg.id, 'PG04', 'PG04_17', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.5';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'bf94b8fb-98be-532b-abcd-85b2b5111206', sg.id, 'PG06', 'PG06_8', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.5';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '922deda7-4a97-516a-814c-aedb5680629f', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.5';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '692e09d4-628e-5db9-99f8-336ebd49a315', sg.id, 'PG08', 'PG08_5', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ50251.5';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '45889ce9-73f9-5031-8261-2e3697e60bbd', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ75381.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '374e9716-f659-5a76-9279-022aba7dab54', sg.id, 'PG02', 'PG02_14', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ75381.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7c2f51d0-13c4-518c-a337-9087c282f793', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ75381.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '417dd984-9836-5197-a64b-5f360fc63f84', sg.id, 'PG04', 'PG04_21', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ75381.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7a4fc4a1-5646-537b-bf72-71e9c9f9e0c7', sg.id, 'PG06', 'PG06_5', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ75381.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '8a0aea50-5bd2-582e-9c43-bfb285237514', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ75381.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'e77f24a0-df86-5d76-b456-9e29279bd4a9', sg.id, 'PG08', 'PG08_5', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'PRZ75381.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b921a7f1-980d-544c-871b-84831aaa62d8', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3b01be33-d800-58c1-b7ef-d2ff96ae79ff', sg.id, 'PG02', 'PG02_15', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '10bfc1ab-f897-5f54-89c3-88c6244a8c5d', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '46dd0af6-97ac-5dfb-9140-d06b3cab3ea3', sg.id, 'PG04', 'PG04_51', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'dd6f320f-4b4d-5fe0-b39b-1e847edafd41', sg.id, 'PG06', 'PG06_5', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b1109a27-7477-50bc-b364-7955043635e6', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.3';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '967dc851-e862-5e03-a442-49db303145f6', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.7';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6243c212-fc20-53d0-a797-a9ea289989f2', sg.id, 'PG02', 'PG02_15', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.7';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5ebbd71b-0953-5262-928f-36b0ce49f15d', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.7';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4079d0de-1efd-5650-ab42-b82a37ef5be2', sg.id, 'PG04', 'PG04_51', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.7';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '08f924e5-fed1-5711-9721-5d0f8deec42e', sg.id, 'PG06', 'PG06_10', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.7';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'eebaa0f1-d5f7-5dc0-b7e2-d5a1cc5aa27a', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ751.7';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5621cebe-3604-549d-856c-767857062180', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1252.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0c6f1e17-9a39-5ada-8210-f14548e8cc8b', sg.id, 'PG02', 'PG02_15', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1252.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'df539d4b-7af6-5b91-9f3f-190005eac1c9', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1252.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2ac6ffc5-3a2f-5ca7-bde6-b6e7af989b3c', sg.id, 'PG04', 'PG04_62', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1252.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'bdab6195-f974-5437-9f09-eb83be9d0188', sg.id, 'PG06', 'PG06_13', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1252.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0ec078ef-37e2-5de3-b02f-7250ac46a080', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1252.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd1726754-0b49-5e46-abad-9dfe85f4d03b', sg.id, 'PG01', 'PG01_4', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1002.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f0366894-24d9-520a-a484-c6cbd66857a5', sg.id, 'PG02', 'PG02_15', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1002.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0b4e6d16-29c6-5f7b-a125-cfb913506cb7', sg.id, 'PG03', 'PG03_1', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1002.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '710b0e31-2e8e-54ec-b9eb-8c47ef7deaa8', sg.id, 'PG04', 'PG04_59', 4
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1002.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1b2b28d0-4dff-57d0-990b-48989a26ea18', sg.id, 'PG06', 'PG06_13', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1002.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'e8b157c9-8c00-5737-84a9-bb502504ce1b', sg.id, 'PG07', 'PG07_1', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'CLZ1002.0';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '47a55854-60aa-5b9e-b54e-90d8a575f404', sg.id, 'PG01', 'PG01_1', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f575d0a6-2188-573c-8a4f-5d6076706d71', sg.id, 'PG02', 'PG02_3', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0c8d6acc-c8b4-5530-9a87-43965ac87b8f', sg.id, 'PG03', 'PG03_23', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a56b2a79-e915-50a3-8cae-a42422fb1c97', sg.id, 'PG06', 'PG06_46', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f98e4115-2c2a-5619-b8ab-b7595000d828', sg.id, 'PG07', 'PG07_3', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '184659d0-be28-5455-8755-87808e340b71', sg.id, 'PG09', 'PG09_7', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '9905b03e-89f0-5a10-84d2-8a9820dea1a7', sg.id, 'PG01', 'PG01_1', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b4e25f04-797c-5912-a4b8-366a579d2139', sg.id, 'PG02', 'PG02_3', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '721146cf-3314-5aff-97a1-0559aaf62f52', sg.id, 'PG03', 'PG03_6', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3a14beca-95ff-5ef8-96b1-70045fef10e1', sg.id, 'PG06', 'PG06_46', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a6989b99-f48b-52ba-9a4e-0345a8a3aa40', sg.id, 'PG07', 'PG07_3', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a4aa8523-a486-5177-9013-2143d282a03a', sg.id, 'PG09', 'PG09_7', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'RBB610SR24TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c74f6706-924d-5314-9cc9-55d8dc159d91', sg.id, 'PG01', 'PG01_1', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0a030072-0d93-5376-b164-0456d035f3bc', sg.id, 'PG02', 'PG02_10', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5cf57e5d-a496-56b7-b592-14876c3293a3', sg.id, 'PG03', 'PG03_23', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '912528f1-080d-50e6-b35e-c79cfabb36a2', sg.id, 'PG06', 'PG06_58', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '638dd16e-6f57-54c0-86c9-eebc11738c44', sg.id, 'PG07', 'PG07_3', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '10b37fe0-e4f1-5774-bd07-895e483318da', sg.id, 'PG09', 'PG09_7', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '0c892af9-0fdf-505d-8424-103c25264f23', sg.id, 'PG01', 'PG01_1', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '7cd09cd3-92b6-57a3-8801-d426f9b3dbc5', sg.id, 'PG02', 'PG02_10', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'c94a38c2-57f5-558d-ae55-e856e2efcb35', sg.id, 'PG03', 'PG03_23', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'cdd09c5a-5a3b-54f7-b2dc-79c863dbc580', sg.id, 'PG06', 'PG06_64', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ab2e0a81-b901-5a2b-b7bf-c0fa2dce5812', sg.id, 'PG07', 'PG07_3', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a55757e5-f2de-5fbf-83e1-9a47ec1964a9', sg.id, 'PG09', 'PG09_7', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '846a59aa-8aa3-5fd6-a6bf-80166d83b29a', sg.id, 'PG01', 'PG01_1', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b0234014-fee0-59ec-9d27-2e47c5d90f2a', sg.id, 'PG02', 'PG02_10', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '4f7d6e28-24e0-5a02-aa12-085ce748d8f1', sg.id, 'PG03', 'PG03_6', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'fdb40ed7-ca8a-5a66-90af-c1b56dbaf848', sg.id, 'PG06', 'PG06_58', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '28b8d443-c8ac-5c87-8191-d36aede96df9', sg.id, 'PG07', 'PG07_3', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'f539d988-e69a-50ce-801f-26854ef10335', sg.id, 'PG09', 'PG09_7', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1210SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'decf80af-1ff2-55f9-8b15-b6d3dc49b642', sg.id, 'PG01', 'PG01_1', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '95eccb7d-0131-5aef-bfe8-b43fd2852aea', sg.id, 'PG02', 'PG02_10', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3b09f426-a8af-58a4-86c3-161ded3256ed', sg.id, 'PG03', 'PG03_6', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '95fd6c32-ce9a-5101-9e79-1d965f451410', sg.id, 'PG06', 'PG06_64', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '535c6ca5-5d53-54d8-9e74-01d8759ce979', sg.id, 'PG07', 'PG07_3', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2c8e2fe4-f6c1-5557-8119-4f634c4976a0', sg.id, 'PG09', 'PG09_7', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB2510SD40TIF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'bec9c0d3-c95b-556c-9b06-1a35108f549c', sg.id, 'PG01', 'PG01_1', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1212SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ae82cf2a-a11e-5b9c-b606-a46b6667d6be', sg.id, 'PG02', 'PG02_10', 2
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1212SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'a674293f-6eda-573b-97c2-6dbc635a05e2', sg.id, 'PG03', 'PG03_23', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1212SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '2dcf807d-53e8-56a0-b203-252ff0f7654b', sg.id, 'PG06', 'PG06_58', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1212SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '367ce3ed-e111-544a-9b0f-6a184359ea51', sg.id, 'PG07', 'PG07_4', 7
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1212SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd30616b5-5121-5c6e-a1f3-68f3f1d0a394', sg.id, 'PG09', 'PG09_7', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'DBB1212SD40IF';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '32f8858e-cb7f-5dcb-add5-812bf008582b', sg.id, 'PG01', 'PG01_9', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.14';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3466359b-9890-5a67-9ec1-993b0d3e9d5e', sg.id, 'PG03', 'PG03_15', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.14';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5bf7f514-718b-58fa-959d-f66e5038e3fe', sg.id, 'PG05', 'PG05_16', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.14';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '3756b9c1-7ecc-53c5-affd-6283a50e00f2', sg.id, 'PG06', 'PG06_3', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.14';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'd14d67c9-3787-59bd-92a4-b87fe5f4df74', sg.id, 'PG08', 'PG08_13', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.14';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'bdf94575-d00d-5b6d-9087-f651f38f0c7f', sg.id, 'PG09', 'PG09_8', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.14';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '89077926-7b64-5a53-914a-dfdf91f8e777', sg.id, 'PG01', 'PG01_9', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.34';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '57720b59-25bc-5d56-9f38-8fccb02fcb41', sg.id, 'PG03', 'PG03_15', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.34';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'cc5958bf-0928-59f2-817b-dc34966174fb', sg.id, 'PG05', 'PG05_16', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.34';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '66df4415-58eb-5f52-8f3b-544e6ef0abf5', sg.id, 'PG06', 'PG06_5', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.34';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'df658707-8214-56f9-bb42-62dcb7614d7f', sg.id, 'PG08', 'PG08_13', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.34';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '1bae2609-2298-5c29-89ba-749e64621d8c', sg.id, 'PG09', 'PG09_8', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.34';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5ba79c10-13d8-5128-8a7c-29bcaf15cf88', sg.id, 'PG01', 'PG01_9', 1
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.74';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'e2e708ea-aad3-5a96-817f-25f2c72fdce4', sg.id, 'PG03', 'PG03_15', 3
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.74';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'ee314841-31bd-5bed-9bbd-5f969f2d5995', sg.id, 'PG05', 'PG05_16', 5
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.74';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT 'b60baba8-038b-5011-847a-c57c760a0ac2', sg.id, 'PG06', 'PG06_10', 6
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.74';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '6d918e0a-7f51-5454-b4f4-e43eb232e15c', sg.id, 'PG08', 'PG08_13', 8
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.74';
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq)
SELECT '5d8b2006-a01f-546d-8abf-b2450980ed4a', sg.id, 'PG09', 'PG09_8', 9
  FROM public.price_list_sub_group sg WHERE sg.subgroup_code = 'COZ1.74';

COMMIT;

-- =============================================================================
-- 4) OPTIONAL: price_list_subgroup_formulas_map
--    Only inserts rows whose formula_code exists in price_list_formulas
--    (FK: price_list_formulas.formula_code). Run separately if you use formulas.
-- =============================================================================
-- BEGIN;
-- DELETE FROM public.price_list_subgroup_formulas_map WHERE price_list_subgroup_code IN ('PTB3.848', 'PTB5.848', 'PTB5.8510', 'PTB5.8520', 'PTB11.8520', 'PTH3.848I', 'PTH5.848I', 'UBB7556', 'UBB1506.56', 'WFB2005.506', 'WFB250606', 'WFB350706', 'WFB400806', 'HBB150706', 'HBB200806', 'HBB250906', 'CLB10050202.3TIS', 'CLB15050202.3TIS', 'CLB10050203.2TIS', 'CLB15050203.2TIS', 'PSB382.3TIS', 'PSB502.3TIS', 'PSB752.3TIS', 'PSB383.2TIS', 'PSB503.2TIS', 'PSB753.2TIS', 'PRB75382.3TIS', 'PRB100502.3TIS', 'PRB75383.2TIS', 'PRB100503.2TIS', 'POB343.2TIS', 'POB42.72.3TIS', 'POB48.62.3TIS', 'POB76.32.3TIS', 'POB101.63.2TIS', 'PRZ50251.3', 'PRZ50251.5', 'PRZ75381.3', 'CLZ751.3', 'CLZ751.7', 'CLZ1252.0', 'CLZ1002.0', 'RBB610SR24IF', 'RBB610SR24TIF', 'DBB1210SD40IF', 'DBB2510SD40IF', 'DBB1210SD40TIF', 'DBB2510SD40TIF', 'DBB1212SD40IF', 'COZ1.14', 'COZ1.34', 'COZ1.74');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'b7a0023a-d9c5-5407-a44c-d8423e539c3f', 'PTB3.848', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'a270695c-bbd7-5995-a567-226951a1fbf3', 'PTB3.848', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '8871d4d8-f5d2-557d-aaa2-bbf8e60a55a2', 'PTB5.848', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '15354e5d-d0f0-5844-9711-f5c4a8307d15', 'PTB5.848', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '942b18fc-a637-5b82-b7b4-db31593c6484', 'PTB5.8510', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '7f0dab61-cb3a-5a1f-8886-c33d84e749e2', 'PTB5.8510', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '0cd06648-cbba-5f40-8b76-17cb4292ec64', 'PTB5.8520', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'bf9a901f-b48d-5493-95f0-0da79fc829aa', 'PTB5.8520', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '254e4448-471a-5b81-80aa-6dc942f3fc6e', 'PTB11.8520', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '3d3b51a5-58e3-5c3c-98df-0216d6a520b4', 'PTB11.8520', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'e35e22bd-c4fe-5b43-8e43-39091ac1187a', 'PTH3.848I', 'FM-3', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-3');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '852292b3-0340-5f54-b3f0-05a19bf3f4de', 'PTH3.848I', 'FM-1', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-1');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '1e7f2b12-be85-5b90-94a8-fbc9cb5f6270', 'PTH5.848I', 'FM-3', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-3');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'b79dc0bd-ebad-5c97-8a6d-2ddda641c95a', 'PTH5.848I', 'FM-1', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-1');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '7ba475b6-d2ca-567d-ae95-5565381ffe49', 'UBB7556', 'FM-9', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-9');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '42b5f20b-adb3-5863-a6e9-cf969b4fb96d', 'UBB7556', 'FM-10', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-10');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '63b6123a-39c2-598f-8177-cbe673e9aaac', 'UBB1506.56', 'FM-9', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-9');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'b4b2ba5d-f0d8-5eba-a964-bc24247d2e9e', 'UBB1506.56', 'FM-10', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-10');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '2c61bdb9-66f9-5dcb-bee1-91c0e69da382', 'WFB2005.506', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '62f003c7-4292-57ce-b8cf-82124f9ec7bd', 'WFB2005.506', 'FM-6', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-6');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '0708f32f-29e0-54dc-9b6c-23e1dcda7a79', 'WFB250606', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'e2da8f10-0f73-5182-81ae-fcf045277e71', 'WFB250606', 'FM-6', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-6');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'df547afd-2afb-59e3-bd5f-d30e5665a5fa', 'WFB350706', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '6fafd38e-098b-5932-aada-ef9290cb9483', 'WFB350706', 'FM-6', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-6');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'cd86093d-ad45-51a7-ace8-344dbbf386be', 'WFB400806', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '8eabcdb6-4e04-5cf6-af91-676b2603d241', 'WFB400806', 'FM-6', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-6');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'c150c691-1fed-5993-a65e-2c43aeb179a9', 'HBB150706', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '5de2fb75-aca9-55c0-b86e-734e9fab4331', 'HBB150706', 'FM-6', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-6');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'f5337f9b-0a62-52db-ab93-4d4e8a2d0f72', 'HBB200806', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '67d6981c-8a20-597a-ad89-a7691ec97473', 'HBB200806', 'FM-6', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-6');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'b2868ea0-832b-5b82-b281-dc158d320dea', 'HBB250906', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '0eafb543-2416-544c-b495-eb71a0c77f6f', 'HBB250906', 'FM-6', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-6');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'ca6b8a64-0088-5082-a7a0-2390f4330095', 'CLB10050202.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'd7bbf732-2730-58f6-aa3c-8f97ef38ffc3', 'CLB10050202.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '7530da23-0d8e-5437-9edb-4f3076046dba', 'CLB15050202.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '0e8808c8-cbd2-5751-86e6-33fdbb6d4071', 'CLB15050202.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'f558c070-ab44-5843-8acf-0d1aa4b35fb2', 'CLB10050203.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '01240524-484f-551c-bb7b-e320b69c7a5e', 'CLB10050203.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'ea8dbdf4-4882-5c7f-a61d-612f3be24721', 'CLB15050203.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '84d9ddbd-5bf8-5e62-9654-c26f15d8166e', 'CLB15050203.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '5f759cd2-f0a4-54b8-928e-f3bfa57e9454', 'PSB382.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '9382da38-e8e5-5f7e-90ac-ef243e27c8ad', 'PSB382.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '24f95b63-aab9-5770-9051-c0642e2709cc', 'PSB502.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '501cb877-bff0-5e0f-b0b6-4399c85ed424', 'PSB502.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'dce5844f-2344-53cd-b58a-e41347b1f7a6', 'PSB752.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '123ec556-517c-5ac0-98a4-5c87979084bc', 'PSB752.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '0030be65-63c6-5257-8cd0-6a6ce6fafa2c', 'PSB383.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '298fd5f7-5b83-53fe-8ea5-3215bb577d34', 'PSB383.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '6e1e39e9-19db-5f6a-987d-9d51c1d555b1', 'PSB503.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '148da95e-2ff3-5762-b774-eff75205e97e', 'PSB503.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'ba251630-826a-5b38-9550-7fddd26be9b8', 'PSB753.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '429e6c47-82bc-59c2-830c-3d967f64cd99', 'PSB753.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '477393f1-5bad-5460-ae66-adb2bab5dbcd', 'PRB75382.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '0d2fbd1b-f878-5dea-837c-b2a047e26746', 'PRB75382.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '298518e4-ba32-5a18-8529-118318a4b6b0', 'PRB100502.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '3f9d639a-1b71-51fc-a327-c52d16bd90eb', 'PRB100502.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '48369d1f-6ca0-5cd6-a36e-16da02a6c398', 'PRB75383.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'f0f10701-6953-59f8-860a-77b973566976', 'PRB75383.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '7259ee67-7e68-5c78-ba1b-c71d18549a6d', 'PRB100503.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '23969219-01a7-55d5-afa8-501158f77657', 'PRB100503.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '6c232b05-14c7-5436-852d-5923e8bcb068', 'POB343.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '691c339a-6f93-50a6-bd80-8853ce803903', 'POB343.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '93a0f19b-9957-5768-a409-9110473cafa8', 'POB42.72.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'e6eae619-85e5-53ea-be26-08ee9c301c67', 'POB42.72.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '7a5002b9-8387-5b4f-81d5-8556f78b3d1e', 'POB48.62.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'b24fcd61-41eb-583d-9760-712a7d47b040', 'POB48.62.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '7fcedd46-302d-5fa5-9032-886c1d92eea0', 'POB76.32.3TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'e6deda39-615d-5522-8831-f8aa1208aabd', 'POB76.32.3TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'ff3c6571-6f06-5077-9d49-ef8d8feea47c', 'POB101.63.2TIS', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'eb7469e6-de9e-5383-bddf-4510b5785c4a', 'POB101.63.2TIS', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'a07ee910-204b-53d1-be1e-8c5b760cb14c', 'PRZ50251.3', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'bdb137a9-0143-5595-9618-562dbdbc7413', 'PRZ50251.3', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '2c971cf0-6a4a-5dea-8b0b-4b6467a7e278', 'PRZ50251.5', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '307bc521-9548-5efd-896c-ebed2a435ad8', 'PRZ50251.5', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '982e1f74-3815-55a8-935b-1576a414d5f9', 'PRZ75381.3', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '55a0bd3d-8e69-5ec2-b3f4-b23844fa8777', 'PRZ75381.3', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'c2318937-f335-5115-a19d-1ea58ff70d98', 'CLZ751.3', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'd5cfc94a-45f5-5445-8bd5-01803d494b56', 'CLZ751.3', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'e05e550e-e35a-579f-af87-4e427b1173bd', 'CLZ751.7', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '9ed6d8b1-5450-58c1-a394-f43a5679eb6b', 'CLZ751.7', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'aab9f3c9-7923-53d6-b2f9-fbf5d2d80de8', 'CLZ1252.0', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '8f5b42ce-a57d-5487-9aff-db5152184ecb', 'CLZ1252.0', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '3053bdca-c1e5-5df1-bb14-94afaa53103f', 'CLZ1002.0', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '6a05c96b-d246-51b4-b5dc-51ae5fce9152', 'CLZ1002.0', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '7e08aac3-e0af-5994-acc8-bd5294a9d30c', 'RBB610SR24IF', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'ddc85eb8-89cb-50f5-a47f-627d6351f776', 'RBB610SR24IF', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '5125b4ca-2a3b-59a7-a0f0-086ab4863c6d', 'RBB610SR24TIF', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '3d6ec7be-172a-5f62-99e3-df8d2dea2f71', 'RBB610SR24TIF', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '00bfec19-95e3-514d-afed-d441a9d50594', 'DBB1210SD40IF', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'f8219784-f2c0-582f-8b69-4df72aeca568', 'DBB1210SD40IF', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'a346a7db-a01e-5d4d-ad4a-669c0e015c50', 'DBB2510SD40IF', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '06c0c4f1-4e05-5692-941d-9757c14685e9', 'DBB2510SD40IF', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'b5bf627c-3942-503e-a715-1a6992b3ae4d', 'DBB1210SD40TIF', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '5779d541-96d0-5285-9072-106d7f2357b7', 'DBB1210SD40TIF', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '254663ea-79d1-505d-8f0b-2610fa9be92a', 'DBB2510SD40TIF', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '092c7588-cd74-5d65-befc-abc8f60f77e4', 'DBB2510SD40TIF', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'e19ac58a-4ef0-5a4e-82a9-e157d421773a', 'DBB1212SD40IF', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'ace5a1ec-7904-52e9-abc0-a3c4daa07bc2', 'DBB1212SD40IF', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '54c9a471-95b3-51b7-a206-a0c57cd12700', 'COZ1.14', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '866cc7c4-203e-5dd5-929e-34aca398de17', 'COZ1.14', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '5454f399-c504-5bff-949e-d59c53603d32', 'COZ1.34', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'b0402b54-8ee0-532c-bf72-87454a6ccf1a', 'COZ1.34', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT 'eff0a6bc-3666-5a38-84b6-10197cd91560', 'COZ1.74', 'FM-8', true, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-8');
-- INSERT INTO public.price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
--   SELECT '43931ddb-1150-52d3-a09e-666c53701cf1', 'COZ1.74', 'FM-7', false, now()
--   WHERE EXISTS (SELECT 1 FROM public.price_list_formulas f WHERE f.formula_code = 'FM-7');
-- COMMIT;
