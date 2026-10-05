# GetPurchaseItemRemain

Optional fields for selecting PO quantities while creating or editing IB/Invoice documents:

```json
{
  "company_code": "COMPANY",
  "site_code": "SITE",
  "usage_type": "IV",
  "exclude_document_code": "AP202609-0001",
  "selected_purchase_items": [
    {"purchase_code": "PO202609-0035", "purchase_item": "1", "qty": 4}
  ]
}
```

- `usage_type`: `IV` (Invoice AP) or `IB` (Inbound); trimmed and case-insensitive. Omit for legacy behavior. It selects which document type to exclude; the existing PO/IB/GR/AP accounting formula is retained.
- `exclude_document_code`: the saved document being edited. Requires `usage_type`. Omit when creating a document. All its rows are excluded before aggregation. IV exclusion occurs before GR/AP matching; IB exclusion occurs before collecting related GRs. IB exclusion matches the code within the requested company and site only; other inbound records retain legacy behavior. Invoices already use company/site scope. Sending usage_type alone does not add any document filters.
- `selected_purchase_items`: current quantities from **other rows in the unsaved form**, excluding the row whose picker is open. Send each quantity in the PO `qty`/`unit` basis, not an unconverted purchase unit. Duplicate PO/item pairs are summed; different PO codes remain independent. Quantities must be finite and non-negative.
- `not_purchase_items`: retains its existing behavior of hiding an entire PO/item pair. Do not put a pair here when it should remain selectable based on quantity.

The new parameters are separate from `not_purchase_items`; no extra enable flag is required. When all three new parameters are omitted or empty, the original filtering, quantities, pagination, and response fields are retained. The two new response fields below are emitted only when `usage_type`, `exclude_document_code`, or a non-empty `selected_purchase_items` is supplied.

Returned quantity fields:

| Field | Meaning |
| --- | --- |
| `qty` | Original PO item quantity (existing field) |
| `available_qty_before_selection` | Available quantity after saved-document accounting and self-exclusion, before form selections |
| `selected_qty` | Quantity used by other form rows supplied in this request |
| `remain_qty` | Available quantity after subtracting `selected_qty` |

Rows with no remaining quantity are omitted before pagination. Existing PO status, supplier, product, and exclusion filters still apply; self-exclusion does not restore a PO filtered out by those conditions. The default PO status remains `PENDING`.

Example without IB/GR: PO quantity 10, other invoices use 2, current invoice previously used 6. Excluding the current invoice gives 8. Other form rows select 4, so this row can select up to 4.

This endpoint is a selection preview. These changes do not add save-time validation or concurrency control. The save endpoint must independently validate the full new document against current saved usage, excluding only the actual document being updated.
