# scripts sql
```
-- Auto-generated SQL script #202511051543
INSERT INTO public.price_list_sub_group (price_list_group_id,subgroup_key,is_trading,price_unit,extra_price_unit,total_net_price_unit,price_weight)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'GROUP_1_ITEM_1|GROUP_2_ITEM_2|GROUP_5_ITEM_2|GROUP_6_ITEM_2',false,800,500,900,45);

-- Auto-generated SQL script #202511051507
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP1','GROUP_1_ITEM_1',1);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP2','GROUP_2_ITEM_2',2);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP5','GROUP_5_ITEM_1',3);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP6','GROUP_6_ITEM_1',4);

-- Auto-generated SQL script #202511051507
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP1','GROUP_1_ITEM_1',1);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP2','GROUP_2_ITEM_1',2);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP5','GROUP_5_ITEM_1',3);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP6','GROUP_6_ITEM_4',4);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('83827ba7-dcb3-4e41-a812-0e4d46b95cd6'::uuid,'PRODUCT_GROUP3','GROUP_3_ITEM_1',5);



-- Auto-generated SQL script #202511051647
INSERT INTO public.group_item (group_id,item_code,item_name)
	VALUES ('f1f5c0ff-a926-4ef9-85fc-669aa53b502a'::uuid,'GROUP_8_ITEM_1','โก 7 = YK');
INSERT INTO public.group_item (group_id,item_code,item_name)
	VALUES ('f1f5c0ff-a926-4ef9-85fc-669aa53b502a'::uuid,'GROUP_8_ITEM_2','โก 7 = LPN');


-- Auto-generated SQL script #202511051557
-- Group2Item3
INSERT INTO public.price_list_sub_group (price_list_group_id,subgroup_key)
	VALUES ('9837db53-1a3c-49a9-a197-1fdfa38984bd'::uuid,'GROUP_1_ITEM_1|GROUP_2_ITEM_3|GROUP_3_ITEM_7|GROUP_5_ITEM_8|GROUP_6_ITEM_14|GROUP_8_ITEM_2');
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('571e9676-9085-426e-9291-582d3aeb3343'::uuid,'PRODUCT_GROUP1','GROUP_1_ITEM_1',1);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('571e9676-9085-426e-9291-582d3aeb3343'::uuid,'PRODUCT_GROUP2','GROUP_2_ITEM_3',2);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('571e9676-9085-426e-9291-582d3aeb3343'::uuid,'PRODUCT_GROUP3','GROUP_3_ITEM_7',3);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('571e9676-9085-426e-9291-582d3aeb3343'::uuid,'PRODUCT_GROUP5','GROUP_5_ITEM_8',4);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('571e9676-9085-426e-9291-582d3aeb3343'::uuid,'PRODUCT_GROUP6','GROUP_6_ITEM_14',4);
INSERT INTO public.price_list_sub_group_key (sub_group_id,code,value,seq)
	VALUES ('571e9676-9085-426e-9291-582d3aeb3343'::uuid,'PRODUCT_GROUP8','GROUP_8_ITEM_2',4);

```