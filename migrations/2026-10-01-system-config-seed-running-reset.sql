INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('PPO', 'LOT', NULL, 'BigLot', '4', '0', '20251106-0002', 1, NULL, NULL);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('PO', 'NML', NULL, 'PurchaseNormal', '4', '0', '20251110-0001', 1, NULL, NULL);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('PO', 'PRE', NULL, 'PurchaseBigLot', '4', '0', '20251028-0001', 1, NULL, NULL);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('PO', 'TRD', NULL, 'PurchaseTrading', '4', '0', '20251107-0001', 1, NULL, NULL);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('PRICE', 'EXPIRY_PRICE_DAYS', NULL, NULL, NULL, NULL, '0', NULL, NULL, NULL);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('SO', 'TOLERANCE_SO', NULL, 'ToleranceSO', '0', '0', '3', NULL, NULL, NULL);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_SO', NULL, 'RunningSaleOrderCode', '', '', '', NULL, NULL, '{"year":"2026","month":"10","prefix":"SO","running_digit":4,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_DBS', NULL, 'RunningDeliveryBookingCode', '', '', '', NULL, NULL, '{"year":"2026","month":"10","prefix":"DBS","running_digit":4,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_AR', NULL, 'RunningTaxInvoiceCode', '', '', '', NULL, NULL, '{"year":"69","month":"09","prefix":"PO","running_digit":3,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_PO', NULL, 'RunningPurchasingOrderCode', '', '', '', NULL, NULL, '{"year":"2026","month":"09","prefix":"PO","running_digit":4,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_DN', NULL, 'RunningDebitNoteCode', '', '', '', NULL, NULL, '{"year":"69","month":"09","prefix":"PO","running_digit":3,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_CN', NULL, 'RunningCreditNoteCode', '', '', '', NULL, NULL, '{"year":"69","month":"09","prefix":"PO","running_digit":3,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('INVOICE', 'AP', NULL, 'AP', '0', '0', '5', NULL, NULL, NULL);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_PB', NULL, 'RunningPurchasingOrderBiglotCode', '', '', '', NULL, NULL, '{"year":"2026","month":"09","prefix":"PB","running_digit":4,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_AP', NULL, 'RunningGRForAccounting', NULL, NULL, NULL, NULL, NULL, '{"year":"26","month":"09","prefix":"PO","running_digit":4,"current_running":0}'::json);
INSERT INTO public.system_config
(topic_code, config_code, tenant_id, config_name, cond1, cond2, value, "sequence", remark, "json")
VALUES('RUNNING', 'RUNNING_QU', NULL, 'RunningQuotationCode', '', '', '', NULL, NULL, '{"year":"2026","month":"10","prefix":"QU","running_digit":4,"current_running":0}'::json);
