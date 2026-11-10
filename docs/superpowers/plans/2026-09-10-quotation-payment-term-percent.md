# ค่าธรรมเนียมตาม payment term ในหน้า quotation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** เลือก payment method / term ในหน้า quotation แล้ว price list ต้องขยับตามเปอร์เซ็นต์ของ term นั้นจริง ๆ (ตอนนี้ไม่ขยับเลยสักกรณี)

**Architecture:** แก้ 2 จุดใน `src/utils/helper/quotationCalculation.ts` — ตารางแปลงรหัส term ที่ใช้คีย์ผิด และสูตรที่คิดเปอร์เซ็นต์จากฐานผิดหน่วย

**Repo:** `C:\work-prime\wms-web` เท่านั้น (erp-core ไม่เกี่ยว)

**Branch:** ทำบน branch ใหม่ `fix/quotation-payment-term-percent` แตกจาก `origin/Develop`

## อาการและสาเหตุ (ยืนยันแล้วจากโค้ด + ข้อมูลจริงบน TMI UAT)

เลือก paymentMethod เป็น DUE แล้วเลือก term ราคาบนตารางไม่เปลี่ยนเลย ทั้งที่ `GetPriceListGroup` คืน terms มาครบ

**สาเหตุที่ 1 — ตารางแปลงรหัสใช้คีย์ที่ไม่มีอยู่จริง** (`quotationCalculation.ts:24-33`)

```ts
const priceTermCodeByDropdownTerm: Record<string, string | null> = {
  DD1: null, DD2: null, DD3: null,
  DD4: 'T1', DD5: 'T1', DD6: 'T2', DD7: 'T3', DD8: 'T3',
};
```

dropdown ส่งค่าจริงเป็น `T1`–`T8` ไม่ใช่ `DD1`–`DD8` — `quotationForm/index.vue:783-789` ยิง
`getPaymentTerm({term_type:['DROPDOWN']})` แล้ว map `value: term.termCode` ตรงจาก DB ซึ่งเก็บ
`T1..T8` (term_name = 3, 7, 15, 30, 45, 60, 90, 120 วัน)

ผลคือ `priceTermCodeByDropdownTerm['T4']` เป็น `undefined` แล้ว `resolvePaymentTermSurcharge:263-266`
คืน `CASH_SURCHARGE` (= 0) ทุกครั้ง → ราคาไม่ขยับ

**สาเหตุที่ 2 — คิดเปอร์เซ็นต์จากฐานผิดหน่วย** (`quotationCalculation.ts:217-235`)

```ts
const surcharge = resolvePaymentTermSurcharge(basePriceWeight, ...);  // คิดจากราคา/กิโล เสมอ
return {
  priceListUnit:   basePriceUnit   + surcharge,   // เอา surcharge ของกิโลมาบวกใส่ราคา/ชิ้น
  priceListWeight: basePriceWeight + surcharge,
};
```

ตัวอย่างจริง (`SG1111`, `total_net_price_unit = 37666`, `total_net_price_weight = 20.9`, DUE 2%):
ควรได้ `37,666 + (37,666 × 2%) = 38,419.32` แต่โค้ดให้ `37,666 + (20.9 × 2%) = 37,666.42`

## กติกาที่ SA เคาะ (2026-09-10)

**การจับคู่ term** — dropdown เก็บจำนวนวัน, price list มี T1/T2/T3 = 30/60, 60/90, 90/120

| วัน (term_code ใน dropdown) | ใช้ค่าธรรมเนียมของ |
|---|---|
| 3 (`T1`), 7 (`T2`), 15 (`T3`) | ไม่บวกอะไร (เหมือน Cash) |
| 30 (`T4`), 45 (`T5`) | price list term `T1` |
| 60 (`T6`) | price list term `T2` |
| 90 (`T7`), 120 (`T8`) | price list term `T3` |

**สูตร** — `[ราคา last updated ตามหน่วย Price UOM] + ([ราคาเดียวกันนั้น] × %)`

```
UOM = kg   : 20.9   + (20.9   × 2%) = 21.32
UOM = pcs  : 37,666 + (37,666 × 2%) = 38,419.32
```

- `paymentMethod = CASH` → ไม่บวกอะไร (เจ้าของยืนยัน 2026-09-10)
- ใช้ `pdcPercent` / `duePercent` เท่านั้น **เลิก fallback ไปใช้ค่าบาท** (`pdc` / `due`) ตามที่เจ้าของสั่ง
- ข้อมูลบางแถวใน `price_list_group_term` เก็บ percent เป็น `0.015` แทน `1.5` (46 จาก 63 แถว) — เจ้าของสั่งว่า
  เป็นข้อมูลเก่า/ผิด **ไม่ต้องสนใจ** ทำตามสูตรตรง ๆ (`percent = 2` แปลว่า 2%)

## Global Constraints

- commit message เป็นภาษาอังกฤษเสมอ คอมเมนต์ในโค้ดเขียนไทยได้
- ห้าม `git add -A` — repo นี้มี `shi-7-step-list.txt` (untracked) และ `.env.development` (แก้ค้าง) ห้ามหลุดเข้า commit
- ห้ามยิง API หรือต่อ DB ของ UAT
- `./node_modules/.bin/vitest run src/utils/helper` ต้องเขียว (`npx vitest` ใช้ไม่ได้ในเครื่องนี้)
- `npx vue-tsc --noEmit` ไม่เกิน baseline **2978** errors
- ห้ามแก้ signature ของ `adjustPriceByPaymentTerm` — มีผู้เรียก 2 จุด (`quotationCreate/index.vue:1131`,
  `quotationDetail/index.vue:887`) ที่ต้องไม่ต้องแก้ตาม

---

## Task 1: แก้ตารางแปลงรหัสและสูตรคิดเปอร์เซ็นต์

**Files:**
- Modify: `src/utils/helper/quotationCalculation.ts` (บรรทัด 22-33 และ 217-286)
- Test: `src/utils/helper/quotationCalculation.spec.ts` (describe block `adjustPriceByPaymentTerm` ~บรรทัด 253-360)

**Interfaces:**
- คง signature เดิม: `adjustPriceByPaymentTerm(basePriceUnit, basePriceWeight, terms, paymentMethod, termCode) => { priceListUnit, priceListWeight }`

- [ ] **Step 1: เขียนเทสใหม่ให้ตรงกติกา SA**

เทสเดิมในไฟล์ pin พฤติกรรมเก่าไว้ (ใช้คีย์ `DD*` และคาดหวังการบวกค่าบาท เช่นเคส
"keeps a zero unit base while adjusting the weight base") — **ต้องเขียน describe block นั้นใหม่ทั้งบล็อก**
ไม่ใช่เพิ่มเคสต่อท้าย เพราะของเดิมขัดกับกติกาใหม่โดยตรง

เคสที่ต้องมี (ใช้ terms ที่มี percent จริง เช่น `T1` pdc 1 / due 2, `T2` pdc 3 / due 4, `T3` pdc 5 / due 6):
- `CASH` ไม่ว่า term ไหน → ราคาไม่เปลี่ยนทั้งสองช่อง
- `T1`, `T2`, `T3` (3/7/15 วัน) กับ DUE และ PDC → ไม่เปลี่ยนทั้งสองช่อง
- `T4`, `T5` → ใช้เปอร์เซ็นต์ของ price list term `T1`
- `T6` → ของ `T2`
- `T7`, `T8` → ของ `T3`
- เคสตัวเลขจริงจาก SA: `adjustPriceByPaymentTerm(37666, 20.9, terms, 'DUE', 'T6')` เมื่อ `T2.duePercent = 2`
  ต้องได้ `priceListUnit = 38419.32` และ `priceListWeight = 21.32` — เคสนี้คือหัวใจ เพราะมันจับทั้งสองบั๊กพร้อมกัน
- ฐานเป็น 0 → ยังเป็น 0 (ไม่ใช่ 0 + ค่าของอีกหน่วย)
- term ที่ price list ไม่ได้ส่งมา (เช่นเลือก `T7` แต่ terms มีแค่ `T1`) → ไม่บวกอะไร
- percent เป็น 0 หรือไม่มีค่า → ไม่บวกอะไร (ห้ามตกไปใช้ `pdc`/`due` ที่เป็นค่าบาท)

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/wms-web && ./node_modules/.bin/vitest run src/utils/helper/quotationCalculation.spec.ts
```

Expected: FAIL หลายเคส (คีย์ `T*` ยังไม่รู้จัก และสูตรยังคิดจากฐานผิด)

- [ ] **Step 3: แก้ตารางแปลงรหัส**

```ts
/**
 * dropdown ของ payment term เก็บ "จำนวนวัน" เป็น term_code T1..T8 (3, 7, 15, 30, 45, 60, 90, 120)
 * ส่วน price list ส่ง term มาแค่ T1/T2/T3 ซึ่งหมายถึงช่วงเครดิต 30/60, 60/90, 90/120
 * รหัสซ้ำกันแต่คนละความหมาย ตารางนี้จึงเป็นตัวแปลง (SA เคาะ 2026-09-10)
 *
 * เดิมคีย์เป็น DD1..DD8 ซึ่งไม่เคยมีในระบบ ทำให้ lookup ไม่เจอแล้วคืนค่าธรรมเนียม 0 ทุกครั้ง
 * ผลคือเลือก term ไหนราคาก็ไม่ขยับเลย
 */
const priceTermCodeByDropdownTerm: Record<string, string | null> = {
  T1: null, // 3 วัน
  T2: null, // 7 วัน
  T3: null, // 15 วัน
  T4: 'T1', // 30 วัน  -> 30/60
  T5: 'T1', // 45 วัน  -> 30/60
  T6: 'T2', // 60 วัน  -> 60/90
  T7: 'T3', // 90 วัน  -> 90/120
  T8: 'T3', // 120 วัน -> 90/120
};
```

- [ ] **Step 4: แก้สูตรให้แต่ละหน่วยคิดจากฐานของตัวเอง**

แทนที่ `calculateTermSurcharge` + `resolvePaymentTermSurcharge` + ตัว `adjustPriceByPaymentTerm` ด้วยรูปนี้
(ชื่อฟังก์ชันภายในเปลี่ยนได้ ขอให้ signature ที่ export ยังเดิม):

```ts
/** ค่าธรรมเนียมของ term ที่เลือก คิดเป็นเปอร์เซ็นต์ ไม่ใช้ช่องค่าบาท (SA เคาะ 2026-09-10) */
const resolvePaymentTermPercent = (
  terms: GetPriceListGroupTermResponse[],
  paymentMethod: string,
  termCode: string,
): number => {
  if (paymentMethod !== 'PDC' && paymentMethod !== 'DUE') {
    return 0;
  }

  const priceTermCode = priceTermCodeByDropdownTerm[termCode];
  if (!priceTermCode) {
    return 0;
  }

  const priceTerm = (terms || []).find(
    (term) => term.termCode === priceTermCode,
  );
  if (!priceTerm) {
    return 0;
  }

  const percent =
    paymentMethod === 'PDC' ? priceTerm.pdcPercent : priceTerm.duePercent;

  return Number.isFinite(percent) && percent > 0 ? percent : 0;
};

/** ราคาบวกค่าธรรมเนียม โดยคิดเปอร์เซ็นต์จากราคาของหน่วยนั้นเอง ไม่ข้ามหน่วยกัน */
const withTermSurcharge = (basePrice: number, percent: number): number => {
  if (!basePrice || !percent) {
    return basePrice;
  }

  return Number((basePrice + basePrice * (percent / 100)).toFixed(2));
};
```

แล้ว `adjustPriceByPaymentTerm` กลายเป็น

```ts
export const adjustPriceByPaymentTerm = (
  basePriceUnit: number,
  basePriceWeight: number,
  terms: GetPriceListGroupTermResponse[],
  paymentMethod: string,
  termCode: string,
): AdjustedPriceList => {
  const percent = resolvePaymentTermPercent(terms, paymentMethod, termCode);

  return {
    priceListUnit: withTermSurcharge(basePriceUnit, percent),
    priceListWeight: withTermSurcharge(basePriceWeight, percent),
  };
};
```

ลบ `CASH_SURCHARGE` กับ `calculateTermSurcharge` ถ้าไม่มีใครใช้แล้ว (grep ก่อนลบ)

- [ ] **Step 5: รันเทสและ type check**

```bash
cd /c/work-prime/wms-web && ./node_modules/.bin/vitest run src/utils/helper && npx vue-tsc --noEmit 2>&1 | tail -3
```

Expected: เทสผ่านทั้งหมด, vue-tsc ไม่เกิน 2978

- [ ] **Step 6: Commit**

```bash
cd /c/work-prime/wms-web
git add src/utils/helper/quotationCalculation.ts src/utils/helper/quotationCalculation.spec.ts
git commit -m "fix(quotation): apply the payment term surcharge to the price of its own unit"
```

---

## Task 2: ตรวจว่าผู้เรียกทั้งสองจุดได้ผลตามกติกา

**Files:** ไม่มีการแก้โค้ด (ถ้าเจอว่าต้องแก้ ให้รายงานกลับ อย่าแก้เอง)

- [ ] **Step 1: ไล่ผู้เรียกทั้งสองจุด**

1. `quotationDetail/index.vue:876-899` `applyPaymentTermAdjustment` — ยืนยันว่า `record.basePriceListUnit` /
   `basePriceListWeight` ถูกเซ็ตจาก `priceListItem.totalNetPriceUnit` / `totalNetPriceWeight` (ราคาดิบจาก API)
   ไม่ใช่ราคาที่บวกค่าธรรมเนียมไปแล้ว — ถ้าเป็นอย่างหลัง การเปลี่ยน term ซ้ำ ๆ จะบวกทบกันไปเรื่อย ๆ
2. `quotationCreate/index.vue:1125-1140` — ตัว `priceMap` ที่ใช้ตอน convert เป็น SO ใช้ฟังก์ชันเดียวกัน
   ยืนยันว่าส่ง `termCode` ตัวเดียวกับที่หน้าจอใช้ (`formData.paymentMethod === 'PDC' ? pdcTerm : dueTerm`)

- [ ] **Step 2: ตรวจว่า watcher ครบ**

`quotationDetail/index.vue:637-664` ต้องมี watcher ของ `paymentMethod`, `pdcTerm`, `dueTerm` ครบ 3 ตัว
และทุกตัวเรียก `recalculateAllPriceList()`

- [ ] **Step 3: รายงาน**

เขียนสิ่งที่ตรวจแล้วพร้อมไฟล์+บรรทัด และเคสที่เจ้าของต้องกดทดสอบบนจอจริง:
สินค้า `CLZ1001.2` (unit Pcs, qty 12, sale method Pcs) เลือก DUE + term 60 วัน แล้วดูว่าคอลัมน์ Price list
ขยับจาก `37,666` เป็น `38,419.32` (ถ้า Price UOM เป็น pcs) หรือจาก `20.9` เป็น `21.32` (ถ้าเป็น kg)
