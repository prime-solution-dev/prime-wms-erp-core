-- +goose Up
-- baseline: pg_dump --schema-only ของ prime_erp (server ต้นแบบ, 2026-10-26) แปลงเป็น idempotent
-- รันบน DB ที่มี schema อยู่แล้วจะไม่เปลี่ยนอะไร — ห้ามแก้ไฟล์นี้ ให้สร้าง migration ใหม่แทน
-- +goose StatementBegin
-- รอ lock ไม่เกิน 10 วินาที ถ้าตารางถูกใช้งานอยู่ให้ fail แทนการค้างรอจน traffic อื่นต่อคิว
SET LOCAL lock_timeout = '10s';

CREATE TABLE IF NOT EXISTS public.approval (
    id uuid NOT NULL,
    approve_code character varying,
    approve_topic character varying,
    document_type character varying,
    document_code character varying,
    document_data json,
    action_date timestamp with time zone,
    status character varying,
    remark character varying,
    curent_step_seq integer,
    create_date timestamp with time zone,
    create_by character varying,
    update_date timestamp with time zone,
    update_by character varying,
    md_item_code character varying
);

CREATE TABLE IF NOT EXISTS public.approval_item (
    id uuid NOT NULL,
    approval_id uuid,
    step_seq integer,
    is_condition boolean,
    condition json,
    status character varying,
    action_by character varying,
    action_date timestamp with time zone,
    create_date timestamp with time zone,
    create_by character varying,
    update_date timestamp with time zone,
    update_by character varying
);

CREATE TABLE IF NOT EXISTS public.approval_item_permission (
    id uuid NOT NULL,
    approval_item_id uuid,
    user_code character varying
);

CREATE TABLE IF NOT EXISTS public.config_next_flow (
    module character varying,
    type character varying,
    is_next_flow boolean,
    topic character varying(50)
);

CREATE TABLE IF NOT EXISTS public.credit (
    id uuid NOT NULL,
    customer_code character varying,
    amount numeric(30,6),
    effective_dtm timestamp with time zone,
    is_active boolean,
    doc_ref character varying,
    approve_date timestamp with time zone,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone,
    alert_balance_credit boolean
);

CREATE TABLE IF NOT EXISTS public.credit_extra (
    id uuid NOT NULL,
    credit_id uuid,
    extra_type character varying,
    amount numeric(30,6),
    effective_dtm timestamp with time zone,
    expire_dtm timestamp with time zone,
    doc_ref character varying,
    approve_date timestamp with time zone,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.credit_request (
    id uuid NOT NULL,
    request_code character varying,
    customer_code character varying,
    amount numeric(30,6),
    request_type character varying,
    status character varying,
    is_approve boolean,
    reason character varying,
    effective_dtm timestamp with time zone,
    expire_dtm timestamp with time zone,
    request_date timestamp with time zone,
    action_date timestamp with time zone,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone,
    is_action boolean,
    approve_date timestamp with time zone,
    cancel_reason character varying
);

CREATE TABLE IF NOT EXISTS public.credit_transaction (
    id uuid NOT NULL,
    transaction_code character varying,
    transaction_type character varying,
    amount numeric(30,6),
    adjust_amount numeric(30,6),
    effective_dtm timestamp with time zone,
    expire_dtm timestamp with time zone,
    force_expire_dtm timestamp with time zone,
    is_approve boolean,
    status character varying,
    reason character varying,
    approve_date timestamp with time zone,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.delivery_booking (
    id uuid NOT NULL,
    delivery_code character varying NOT NULL,
    company_code character varying,
    site_code character varying,
    delivery_method character varying,
    document_ref character varying,
    customer_code character varying,
    ship_to_address text,
    delivery_date date,
    delivery_time_code character varying,
    license_plate character varying,
    contact_name character varying,
    tel character varying,
    total_weight numeric(20,8),
    status character varying,
    remark character varying,
    create_date timestamp without time zone,
    create_by character varying,
    update_date timestamp without time zone,
    update_by character varying,
    booking_slot_type character varying,
    status_approve_gi character varying,
    external_id character varying,
    external_document_number character varying
);

CREATE TABLE IF NOT EXISTS public.delivery_booking_item (
    id uuid NOT NULL,
    delivery_item character varying,
    delivery_id uuid NOT NULL,
    product_code character varying,
    qty numeric(20,8),
    unit_code character varying,
    price_list_unit numeric(20,8),
    sale_qty numeric(20,8),
    sale_unit_code character varying,
    total_weight numeric(20,8),
    create_date timestamp without time zone,
    create_by character varying,
    update_date timestamp without time zone,
    update_by character varying,
    document_ref_item character varying,
    status character varying,
    weight numeric(20,8),
    weight_unit numeric(20,8),
    remark character varying,
    product_desc character varying(255)
);

CREATE TABLE IF NOT EXISTS public.deposit (
    id uuid NOT NULL,
    deposit_code character varying,
    doc_ref_type character varying,
    doc_ref character varying,
    customer_code character varying,
    deposit_date timestamp with time zone,
    amount_total numeric(30,6),
    amount_used numeric(30,6),
    status character varying,
    remark character varying,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone,
    amount_remain numeric(30,6),
    company_format character varying
);

CREATE TABLE IF NOT EXISTS public.extra_pattern (
    pattern_code character varying NOT NULL,
    pattern_name character varying
);

CREATE TABLE IF NOT EXISTS public."group" (
    id uuid NOT NULL,
    group_code character varying(225) NOT NULL,
    group_name character varying(255),
    value character varying(255),
    value_int numeric(20,4),
    seq integer,
    create_dtm timestamp with time zone,
    create_by character varying(225),
    update_dtm timestamp with time zone,
    update_by character varying(225)
);

CREATE TABLE IF NOT EXISTS public.group_item (
    id uuid NOT NULL,
    group_id uuid,
    item_code character varying(225) NOT NULL,
    item_name character varying(255),
    value character varying(255),
    value_int numeric(20,4),
    create_dtm timestamp with time zone,
    create_by character varying(225),
    update_dtm timestamp with time zone,
    update_by character varying(225),
    parent_group_code character varying(255),
    parent_group_item_code character varying(255)
);

CREATE TABLE IF NOT EXISTS public.invoice (
    id uuid NOT NULL,
    invoice_code character varying,
    due_date timestamp with time zone,
    status character varying,
    remark character varying,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_dtm timestamp with time zone,
    total_amount numeric(20,4),
    invoice_ref character varying,
    invoice_type character varying,
    document_ref_type character varying,
    document_ref character varying,
    credit_term_day integer,
    payment_date timestamp with time zone,
    document_date timestamp with time zone,
    tax_date timestamp with time zone,
    tax_invoice character varying,
    party_type character varying,
    party_code character varying,
    total_vat numeric(20,4),
    party_name character varying,
    party_branch character varying,
    party_address character varying,
    party_email character varying,
    party_tel character varying,
    party_tax_id character varying,
    external_id character varying,
    payment_method character varying,
    submit_date timestamp with time zone,
    owner_name character varying,
    party_document_ref character varying,
    subtotal_excl_vat_deposit numeric(20,2),
    subtotal_excl_vat numeric(20,2),
    company_code character varying,
    site_code character varying,
    is_interface boolean,
    invoice_date timestamp with time zone,
    total_discount numeric(20,2),
    note character varying
);

CREATE TABLE IF NOT EXISTS public.invoice_deposit (
    id uuid NOT NULL,
    invoice_id uuid,
    deposit_code character varying,
    apply_date timestamp with time zone,
    amount numeric(30,6),
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.invoice_item (
    id uuid NOT NULL,
    invoice_item character varying,
    invoice_id uuid,
    doc_ref_item character varying,
    product_code character varying,
    qty numeric(30,6),
    unit_code character varying,
    price_unit numeric(30,6),
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_dtm timestamp with time zone,
    invoice_qty numeric(30,6),
    invoice_unit character varying,
    invoice_unit_type character varying,
    unit_uom character varying,
    weight_unit numeric(30,6),
    avg_weight_unit numeric(30,6),
    invoice_weight numeric(30,6),
    total_discount numeric(30,6),
    total_discount_percent numeric(30,6),
    document_ref_type character varying,
    document_ref character varying,
    document_ref_item character varying,
    source_type character varying,
    source_code character varying,
    source_item character varying,
    total_amount numeric(30,6),
    total_vat numeric(30,6),
    status character varying,
    remark character varying,
    weight numeric(30,6),
    product_description character varying,
    price_list_unit numeric(30,6),
    subtotal_excl_vat_deposit numeric(30,6),
    subtotal_excl_vat numeric(30,6),
    document_date timestamp with time zone,
    article_code character varying,
    product_desc character varying,
    article_type character varying,
    total_weight numeric(30,6)
);

CREATE TABLE IF NOT EXISTS public.payment (
    id uuid NOT NULL,
    payment_code character varying,
    customer_code character varying,
    payment_date timestamp with time zone,
    amount numeric(30,6),
    method character varying,
    status character varying,
    remark character varying,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone,
    external_id character varying
);

CREATE TABLE IF NOT EXISTS public.payment_invoice (
    id uuid NOT NULL,
    payment_id uuid,
    invoice_code character varying,
    amount numeric(30,6),
    apply_date timestamp with time zone,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_date timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.payment_term (
    id uuid NOT NULL,
    term_code character varying,
    term_name character varying,
    term_type character varying,
    create_by character varying,
    create_dtm timestamp with time zone,
    update_by character varying,
    update_dtm timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.pre_purchase (
    id uuid NOT NULL,
    pre_purchase_code character varying(255),
    purchase_type character varying(255),
    company_code character varying(255),
    site_code character varying(255),
    doc_ref_type character varying(255),
    doc_ref character varying(255),
    supplier_code character varying(255),
    delivery_address character varying(255),
    status character varying(255),
    total_amount numeric(20,4),
    total_weight numeric(20,4),
    total_discount numeric(20,4),
    total_vat numeric(20,4),
    subtotal_excl_vat numeric(20,4),
    is_approved boolean,
    status_approve character varying(255),
    remark character varying(255),
    create_by character varying(255),
    create_dtm timestamp without time zone,
    update_by character varying(255),
    update_dtm timestamp without time zone,
    subtotal_excl_discount_excl_vat numeric(20,4),
    supplier_name character varying(255),
    supplier_address character varying(255),
    supplier_phone character varying(255),
    supplier_email character varying(255),
    credit_term integer,
    note character varying(255)
);

CREATE TABLE IF NOT EXISTS public.pre_purchase_item (
    id uuid NOT NULL,
    pre_purchase_id uuid,
    pre_item character varying,
    hierarchy_type character varying(255),
    hierarchy_code character varying(255),
    doc_ref_item character varying(255),
    qty numeric(20,4),
    unit character varying(255),
    purchase_qty numeric(20,4),
    purchase_unit character varying(255),
    purchase_unit_type character varying(255),
    price_unit numeric(20,4),
    total_discount numeric(20,4),
    total_amount numeric(20,4),
    unit_uom character varying(255),
    total_cost numeric(20,4),
    total_discount_percent numeric(20,4),
    total_vat numeric(20,4),
    subtotal_excl_vat numeric(20,4),
    weight_unit numeric(20,4),
    total_weight numeric(20,4),
    status character varying(255),
    remark character varying(255),
    create_dtm timestamp without time zone,
    create_by character varying(255),
    update_dtm timestamp without time zone,
    update_by character varying(255),
    discount_type character varying(20)
);

CREATE TABLE IF NOT EXISTS public.price_list_extra_config (
    id uuid NOT NULL,
    group_code character varying(255),
    is_active boolean,
    config_json json,
    create_by character varying(255),
    create_dtm timestamp with time zone DEFAULT now(),
    update_by character varying(255),
    update_dtm timestamp with time zone DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.price_list_extra_config_backup_20261005 (
    id uuid,
    group_code character varying(255),
    config_json json,
    update_by character varying(255),
    update_dtm timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.price_list_formulas (
    id uuid NOT NULL,
    formula_code character varying NOT NULL,
    name text NOT NULL,
    uom text NOT NULL,
    formula_type text NOT NULL,
    expression text,
    params jsonb,
    rounding integer DEFAULT 2,
    create_dtm timestamp with time zone DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.price_list_group (
    id uuid NOT NULL,
    company_code character varying,
    site_code character varying,
    group_code character varying NOT NULL,
    price_unit numeric(30,6),
    price_weight numeric(30,6),
    before_price_unit numeric(30,6),
    before_price_weight numeric(30,6),
    currency character varying DEFAULT 'THB'::character varying,
    effective_date timestamp with time zone,
    extra_pattern character varying,
    remark text,
    create_by character varying,
    create_dtm timestamp with time zone DEFAULT now(),
    update_by character varying,
    update_dtm timestamp with time zone,
    group_key character varying,
    group_name character varying(255),
    seq integer
);

CREATE TABLE IF NOT EXISTS public.price_list_group_extra (
    id uuid NOT NULL,
    price_list_group_id uuid,
    extra_key character varying,
    length_extra_key integer,
    operator character varying,
    cond_range_min numeric(30,6),
    cond_range_max numeric(30,6),
    create_by character varying,
    create_dtm timestamp with time zone DEFAULT now(),
    update_by character varying,
    update_dtm timestamp with time zone,
    condition_code character varying,
    value_int numeric(30,6)
);

CREATE TABLE IF NOT EXISTS public.price_list_group_extra_key (
    id uuid NOT NULL,
    group_extra_id uuid,
    code character varying,
    value character varying,
    seq integer
);

CREATE TABLE IF NOT EXISTS public.price_list_group_history (
    id uuid NOT NULL,
    company_code character varying,
    site_code character varying,
    group_code character varying,
    price_unit numeric(30,6),
    price_weight numeric(30,6),
    before_price_unit numeric(30,6),
    before_price_weight numeric(30,6),
    currency character varying DEFAULT 'THB'::character varying,
    effective_date timestamp with time zone,
    expiry_date timestamp with time zone,
    extra_pattern character varying,
    remark character varying,
    create_by character varying,
    create_dtm timestamp with time zone DEFAULT now(),
    update_by character varying,
    update_dtm timestamp with time zone,
    price_list_group_id uuid
);

CREATE TABLE IF NOT EXISTS public.price_list_group_key (
    id uuid NOT NULL,
    price_list_group_id uuid,
    code character varying(255),
    value character varying(255),
    seq bigint
);

CREATE TABLE IF NOT EXISTS public.price_list_group_term (
    id uuid NOT NULL,
    price_list_group_id uuid,
    term_code character varying,
    pdc numeric(30,6),
    pdc_percent numeric(30,6),
    due numeric(30,6),
    due_percent numeric(30,6),
    create_by character varying,
    create_dtm timestamp with time zone DEFAULT now(),
    update_by character varying,
    update_dtm timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.price_list_sub_group (
    id uuid NOT NULL,
    price_list_group_id uuid,
    subgroup_key character varying,
    is_trading boolean,
    price_unit numeric(30,6),
    extra_price_unit numeric(30,6),
    total_net_price_unit numeric(30,6),
    price_weight numeric(30,6),
    extra_price_weight numeric(30,6),
    term_price_weight numeric(30,6),
    total_net_price_weight numeric(30,6),
    before_price_unit numeric(30,6),
    before_extra_price_unit numeric(30,6),
    before_term_price_unit numeric(30,6),
    before_total_net_price_unit numeric(30,6),
    before_price_weight numeric(30,6),
    before_extra_price_weight numeric(30,6),
    before_term_price_weight numeric(30,6),
    before_total_net_price_weight numeric(30,6),
    effective_date timestamp with time zone,
    remark text,
    create_by character varying,
    create_dtm timestamp with time zone DEFAULT now(),
    update_by character varying,
    update_dtm timestamp with time zone,
    udf_json json,
    subgroup_code character varying,
    need_recalculate boolean DEFAULT false NOT NULL
);

CREATE TABLE IF NOT EXISTS public.price_list_sub_group_history (
    id uuid NOT NULL,
    price_list_group_id uuid,
    subgroup_key character varying,
    is_trading boolean,
    price_unit numeric(30,6),
    extra_price_unit numeric(30,6),
    total_net_price_unit numeric(30,6),
    price_weight numeric(30,6),
    extra_price_weight numeric(30,6),
    term_price_weight numeric(30,6),
    total_net_price_weight numeric(30,6),
    before_price_unit numeric(30,6),
    before_extra_price_unit numeric(30,6),
    before_term_price_unit numeric(30,6),
    before_total_net_price_unit numeric(30,6),
    before_price_weight numeric(30,6),
    before_extra_price_weight numeric(30,6),
    before_term_price_weight numeric(30,6),
    before_total_net_price_weight numeric(30,6),
    effective_date timestamp with time zone,
    expiry_date timestamp with time zone,
    remark text,
    create_by character varying,
    create_dtm timestamp with time zone DEFAULT now(),
    update_by character varying,
    update_dtm timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.price_list_sub_group_key (
    id uuid NOT NULL,
    sub_group_id uuid,
    code character varying,
    value character varying,
    seq integer
);

CREATE TABLE IF NOT EXISTS public.price_list_sub_group_key_history (
    id uuid NOT NULL,
    sub_group_history_id uuid,
    code character varying,
    value character varying,
    seq integer
);

CREATE TABLE IF NOT EXISTS public.price_list_subgroup_formulas_map (
    id uuid NOT NULL,
    price_list_subgroup_code character varying NOT NULL,
    price_list_formulas_code character varying NOT NULL,
    is_default boolean DEFAULT false,
    create_dtm timestamp with time zone DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.purchase (
    id uuid NOT NULL,
    purchase_code character varying(255),
    purchase_type character varying(255),
    company_code character varying(255),
    site_code character varying(255),
    doc_ref_type character varying(255),
    doc_ref character varying(225),
    supplier_code character varying(255),
    delivery_date timestamp with time zone,
    delivery_address character varying(255),
    status character varying(255),
    total_amount numeric(20,4),
    total_weight numeric(20,4),
    total_discount numeric(20,4),
    total_vat numeric(30,6),
    subtotal_excl_vat numeric(20,4),
    is_approved boolean,
    status_approve character varying(255),
    remark character varying(255),
    create_by character varying(255),
    create_dtm timestamp without time zone,
    update_by character varying(255),
    update_dtm timestamp without time zone,
    status_payment character varying(255),
    subtotal_excl_discount_excl_vat numeric(20,4),
    trading_ref character varying(255),
    supplier_name character varying(255),
    supplier_address character varying(255),
    supplier_phone character varying(255),
    supplier_email character varying(255),
    credit_term integer,
    used_type character varying,
    used_status character varying,
    note character varying(255)
);

CREATE TABLE IF NOT EXISTS public.purchase_item (
    id uuid NOT NULL,
    purchase_id uuid,
    purchase_item character varying,
    product_code character varying(255),
    doc_ref_item character varying(255),
    qty numeric(20,4),
    unit character varying(255),
    purchase_qty numeric(20,4),
    purchase_unit character varying(255),
    purchase_unit_type character varying(255),
    price_unit numeric(20,4),
    total_discount numeric(20,4),
    total_amount numeric(20,4),
    unit_uom character varying(255),
    total_cost numeric(20,4),
    total_discount_percent numeric(20,4),
    total_vat numeric(20,4),
    subtotal_excl_vat numeric(20,4),
    weight_unit numeric(20,4),
    total_weight numeric(20,4),
    status character varying(255),
    remark character varying(255),
    create_dtm timestamp without time zone,
    create_by character varying(255),
    update_dtm timestamp without time zone,
    update_by character varying(255),
    discount_type character varying(20),
    status_payment character varying(255),
    product_desc character varying(255),
    product_group_code character varying(255),
    product_group_name character varying(255)
);

CREATE TABLE IF NOT EXISTS public.quotation (
    id uuid NOT NULL,
    quotation_code character varying,
    customer_code character varying,
    customer_name character varying,
    delivery_date timestamp without time zone,
    sold_to_code character varying,
    sold_to_address character varying,
    bill_to_code character varying,
    bill_to_address character varying,
    ship_to_code character varying,
    ship_to_type character varying,
    ship_to_address character varying,
    delivery_method character varying,
    transport_cost_type character varying,
    pass_price boolean,
    total_transport_cost numeric(20,8),
    total_amount numeric(20,8),
    total_weight numeric(20,8),
    subtotal_excl_transport numeric(20,8),
    subtotal_weight_excl_transport numeric(20,8),
    payment_method character varying,
    peyment_term_code character varying,
    sale_person_code character varying,
    effective_date_price timestamp without time zone,
    expire_price_day integer,
    expire_price_date timestamp without time zone,
    pass_price_list character varying,
    pass_atp_check character varying,
    pass_credit_limit character varying,
    pass_price_expire character varying,
    status character varying,
    remark character varying,
    is_approved boolean,
    status_approve character varying,
    create_date timestamp without time zone,
    create_by character varying,
    update_date timestamp without time zone,
    update_by character varying,
    company_code character varying,
    site_code character varying,
    total_vat numeric(20,8),
    total_discount numeric(20,8),
    subtotal_excl_vat numeric(20,8),
    total_transport_cost_vat numeric(20,8),
    remark_approval character varying,
    revision numeric(30,6),
    quotation_code_ref character varying,
    credit_term character varying,
    payer_term character varying,
    ref_po_doc character varying
);

CREATE TABLE IF NOT EXISTS public.quotation_item (
    id uuid NOT NULL,
    quotation_id uuid,
    quotation_item character varying,
    product_code character varying,
    product_desc character varying,
    qty numeric(20,8),
    unit character varying,
    price_list_unit numeric(20,8),
    sale_qty numeric(20,8),
    sale_unit character varying,
    sale_unit_type character varying,
    pass_price character varying,
    pass_weight character varying,
    price_unit numeric(20,8),
    total_amount numeric(20,8),
    transport_cost_unit numeric(20,8),
    subtotal_excl_transport numeric(20,8),
    net_price_unit_excl_transport numeric(20,8),
    weight_unit numeric(20,8),
    avg_weight_unit numeric(20,8),
    total_weight numeric(20,8),
    transport_cost_weight_unit numeric(20,8),
    subtotal_weight_excl_transport numeric(20,8),
    net_price_per_weight_excl_transport numeric(20,8),
    status character varying,
    remark character varying,
    create_date timestamp without time zone,
    create_by character varying,
    update_date timestamp without time zone,
    update_by character varying,
    subtotal_excl_vat numeric(20,8),
    total_vat numeric(20,8),
    unit_uom character varying,
    total_discount numeric(20,8),
    total_discount_percent numeric(20,8)
);

CREATE TABLE IF NOT EXISTS public.sale (
    id uuid NOT NULL,
    sale_code character varying,
    customer_code character varying,
    customer_name character varying,
    delivery_date timestamp without time zone,
    sold_to_code character varying,
    sold_to_address character varying,
    bill_to_code character varying,
    bill_to_address character varying,
    ship_to_code character varying,
    ship_to_type character varying,
    ship_to_address character varying,
    delivery_method character varying,
    transport_cost_type character varying,
    total_transport_cost numeric(20,8),
    total_amount numeric(20,8),
    total_weight numeric(20,8),
    subtotal_excl_transport numeric(20,8),
    subtotal_weight_excl_transport numeric(20,8),
    payment_method character varying,
    peyment_term_code character varying,
    sale_person_code character varying,
    effective_date_price timestamp without time zone,
    expire_price_day integer,
    expire_price_date timestamp without time zone,
    pass_price_list character varying,
    pass_atp_check character varying,
    pass_credit_limit character varying,
    pass_price_expire character varying,
    status character varying,
    status_payment character varying,
    remark character varying,
    is_approved boolean,
    status_approve character varying,
    create_date timestamp without time zone,
    create_by character varying,
    update_date timestamp without time zone,
    update_by character varying,
    company_code character varying,
    site_code character varying,
    total_vat numeric(20,8),
    total_discount numeric(20,8),
    subtotal_excl_vat numeric(20,8),
    total_transport_cost_vat numeric(20,8),
    remark_approval character varying,
    status_payment_action character varying,
    ref_po_doc character varying,
    credit_term character varying,
    payer_term character varying
);

CREATE TABLE IF NOT EXISTS public.sale_deposit (
    id uuid NOT NULL,
    sale_id uuid,
    deposit_code character varying,
    deposit_date timestamp without time zone,
    amount_total numeric(20,8),
    amount_used numeric(20,8),
    amount_remain numeric(20,8)
);

CREATE TABLE IF NOT EXISTS public.sale_item (
    id uuid NOT NULL,
    sale_id uuid,
    sale_item character varying,
    product_code character varying,
    product_desc character varying,
    qty numeric(20,8),
    origin_qty numeric(20,8),
    unit character varying,
    price_list_unit numeric(20,8),
    sale_qty numeric(20,8),
    sale_unit character varying,
    sale_unit_type character varying,
    pass_price_unit character varying,
    pass_price_weight character varying,
    price_unit numeric(20,8),
    total_amount numeric(20,8),
    transport_cost_unit numeric(20,8),
    subtotal_excl_transport numeric(20,8),
    net_price_unit_excl_transport numeric(20,8),
    weight_unit numeric(20,8),
    avg_weight_unit numeric(20,8),
    total_weight numeric(20,8),
    transport_cost_weight_unit numeric(20,8),
    subtotal_weight_excl_transport numeric(20,8),
    net_price_per_weight_excl_transport numeric(20,8),
    status character varying,
    remark character varying,
    create_date timestamp without time zone,
    create_by character varying,
    update_date timestamp without time zone,
    update_by character varying,
    subtotal_excl_vat numeric(20,8),
    total_vat numeric(20,8),
    unit_uom character varying,
    total_discount numeric(20,8),
    total_discount_percent numeric(20,8),
    document_ref character varying,
    document_ref_item character varying,
    old_price_list_unit numeric(20,8),
    convert_price_list_unit numeric(20,8) DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.system_config (
    topic_code character varying(50),
    config_code character varying(50),
    tenant_id uuid,
    config_name character varying(100),
    cond1 character varying(50),
    cond2 character varying(50),
    value character varying(100),
    sequence integer,
    remark character varying,
    "json" json
);

CREATE TABLE IF NOT EXISTS public."time" (
    id uuid NOT NULL,
    topic character varying,
    code character varying,
    name character varying,
    start_time time without time zone,
    end_time time without time zone
);

CREATE TABLE IF NOT EXISTS public.unit (
    id uuid NOT NULL,
    topic character varying(255),
    unit_code character varying(255),
    unit_name character varying(255)
);

CREATE TABLE IF NOT EXISTS public.unit_method (
    id uuid NOT NULL,
    method_code character varying(255),
    method_name character varying(255),
    unit_id uuid
);

CREATE TABLE IF NOT EXISTS public.unit_uom (
    id uuid NOT NULL,
    uom_code character varying(255),
    uom_name character varying(255),
    method_id uuid
);

DO $baseline$ BEGIN
ALTER TABLE ONLY public.credit_extra
    ADD CONSTRAINT credit_extra_unique UNIQUE (credit_id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.delivery_booking_item
    ADD CONSTRAINT delivery_booking_item_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.delivery_booking
    ADD CONSTRAINT delivery_booking_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public."group"
    ADD CONSTRAINT group_group_code_key UNIQUE (group_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.group_item
    ADD CONSTRAINT group_item_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public."group"
    ADD CONSTRAINT group_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.payment_term
    ADD CONSTRAINT payment_term_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.pre_purchase_item
    ADD CONSTRAINT pre_purchase_item_pk PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.pre_purchase
    ADD CONSTRAINT pre_purchase_pk PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.pre_purchase
    ADD CONSTRAINT pre_purchase_pre_purchase_code_key UNIQUE (pre_purchase_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_extra_config
    ADD CONSTRAINT price_list_extra_config_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_formulas
    ADD CONSTRAINT price_list_formulas_code_unique UNIQUE (formula_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_formulas
    ADD CONSTRAINT price_list_formulas_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_extra_key
    ADD CONSTRAINT price_list_group_extra_key_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_extra
    ADD CONSTRAINT price_list_group_extra_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_history
    ADD CONSTRAINT price_list_group_history_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_key
    ADD CONSTRAINT price_list_group_key_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group
    ADD CONSTRAINT price_list_group_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_term
    ADD CONSTRAINT price_list_group_term_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group
    ADD CONSTRAINT price_list_group_unique UNIQUE (group_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group
    ADD CONSTRAINT price_list_sub_group__code_unique UNIQUE (subgroup_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_subgroup_formulas_map
    ADD CONSTRAINT price_list_sub_group_formulas_map_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group_history
    ADD CONSTRAINT price_list_sub_group_history_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group_key_history
    ADD CONSTRAINT price_list_sub_group_key_history_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group_key
    ADD CONSTRAINT price_list_sub_group_key_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group
    ADD CONSTRAINT price_list_sub_group_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.purchase_item
    ADD CONSTRAINT purchase_item_pk PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.purchase
    ADD CONSTRAINT purchase_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.purchase
    ADD CONSTRAINT purchase_purchase_code_key UNIQUE (purchase_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.quotation_item
    ADD CONSTRAINT quotation_item_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.quotation
    ADD CONSTRAINT quotation_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.sale_deposit
    ADD CONSTRAINT sale_deposit_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.sale_item
    ADD CONSTRAINT sale_item_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.sale
    ADD CONSTRAINT sale_pkey PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

CREATE INDEX IF NOT EXISTS delivery_booking_company_site_idx ON public.delivery_booking USING btree (company_code, site_code);

CREATE INDEX IF NOT EXISTS delivery_booking_document_ref_idx ON public.delivery_booking USING btree (document_ref);

CREATE INDEX IF NOT EXISTS delivery_booking_item_delivery_id_idx ON public.delivery_booking_item USING btree (delivery_id);

CREATE INDEX IF NOT EXISTS delivery_booking_item_document_ref_item_idx ON public.delivery_booking_item USING btree (document_ref_item);

CREATE INDEX IF NOT EXISTS idx_price_list_group_company_site ON public.price_list_group USING btree (company_code, site_code);

CREATE INDEX IF NOT EXISTS idx_price_list_group_extra_key_group_extra_id ON public.price_list_group_extra_key USING btree (group_extra_id);

CREATE INDEX IF NOT EXISTS idx_price_list_group_extra_price_list_group_id ON public.price_list_group_extra USING btree (price_list_group_id);

CREATE INDEX IF NOT EXISTS idx_price_list_group_term_price_list_group_id ON public.price_list_group_term USING btree (price_list_group_id);

CREATE INDEX IF NOT EXISTS idx_price_list_subgroup_formulas_map_subgroup_code ON public.price_list_subgroup_formulas_map USING btree (price_list_subgroup_code);

CREATE INDEX IF NOT EXISTS quotation_company_site_idx ON public.quotation USING btree (company_code, site_code);

CREATE INDEX IF NOT EXISTS quotation_item_quotation_id_idx ON public.quotation_item USING btree (quotation_id);

CREATE INDEX IF NOT EXISTS sale_company_site_idx ON public.sale USING btree (company_code, site_code);

CREATE INDEX IF NOT EXISTS sale_deposit_sale_id_idx ON public.sale_deposit USING btree (sale_id);

CREATE INDEX IF NOT EXISTS sale_item_sale_id_idx ON public.sale_item USING btree (sale_id);

CREATE INDEX IF NOT EXISTS sale_item_sale_item_idx ON public.sale_item USING btree (sale_item);

DO $baseline$ BEGIN
ALTER TABLE ONLY public.group_item
    ADD CONSTRAINT group_item_group_id_fkey FOREIGN KEY (group_id) REFERENCES public."group"(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.pre_purchase_item
    ADD CONSTRAINT pre_purchase_item_pre_purchase_fk FOREIGN KEY (pre_purchase_id) REFERENCES public.pre_purchase(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_extra_key
    ADD CONSTRAINT price_list_group_extra_key_group_extra_id_fkey FOREIGN KEY (group_extra_id) REFERENCES public.price_list_group_extra(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_extra
    ADD CONSTRAINT price_list_group_extra_price_list_group_id_fkey FOREIGN KEY (price_list_group_id) REFERENCES public.price_list_group(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_history
    ADD CONSTRAINT price_list_group_history_price_list_group_fk FOREIGN KEY (price_list_group_id) REFERENCES public.price_list_group(id);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_key
    ADD CONSTRAINT price_list_group_key_price_list_group_id_fkey FOREIGN KEY (price_list_group_id) REFERENCES public.price_list_group(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_group_term
    ADD CONSTRAINT price_list_group_term_price_list_group_id_fkey FOREIGN KEY (price_list_group_id) REFERENCES public.price_list_group(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_subgroup_formulas_map
    ADD CONSTRAINT price_list_sub_group_formulas_map_price_list_formulas_code_fkey FOREIGN KEY (price_list_formulas_code) REFERENCES public.price_list_formulas(formula_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_subgroup_formulas_map
    ADD CONSTRAINT price_list_sub_group_formulas_map_price_list_subgroup_code_fkey FOREIGN KEY (price_list_subgroup_code) REFERENCES public.price_list_sub_group(subgroup_code);
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group_history
    ADD CONSTRAINT price_list_sub_group_history_price_list_group_id_fkey FOREIGN KEY (price_list_group_id) REFERENCES public.price_list_group(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group_key_history
    ADD CONSTRAINT price_list_sub_group_key_history_sub_group_history_id_fkey FOREIGN KEY (sub_group_history_id) REFERENCES public.price_list_sub_group_history(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group_key
    ADD CONSTRAINT price_list_sub_group_key_sub_group_id_fkey FOREIGN KEY (sub_group_id) REFERENCES public.price_list_sub_group(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.price_list_sub_group
    ADD CONSTRAINT price_list_sub_group_price_list_group_id_fkey FOREIGN KEY (price_list_group_id) REFERENCES public.price_list_group(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;

DO $baseline$ BEGIN
ALTER TABLE ONLY public.purchase_item
    ADD CONSTRAINT purchase_item_purchase_fk FOREIGN KEY (purchase_id) REFERENCES public.purchase(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table OR invalid_table_definition OR object_not_in_prerequisite_state OR duplicate_function THEN NULL;
END $baseline$;
-- +goose StatementEnd

-- +goose Down
-- baseline ไม่มี down
