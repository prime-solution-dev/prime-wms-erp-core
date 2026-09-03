SHELL := /usr/bin/fish

.PHONY: test test-integration test-integration-group test-integration-prepurchase test-integration-pricelist tidy seed-price-list-test seed-price-list-formulas

test:
	go test ./...

test-integration:
	go test -v -tags=integration ./internal/repositories/priceList

test-integration-group:
	go test -v -tags=integration ./internal/services/group-service

test-integration-prepurchase:
	go test -v -tags=integration ./internal/repositories/prePurchase

test-integration-pricelist:
	go test -v -tags=integration ./internal/services/price-service

tidy:
	go mod tidy

.PHONY: seed-price-list seed-price-list-test seed-price-list-formulas

seed-price-list:
	@test -n "$(GROUP_ID)"; or begin; echo "GROUP_ID is required. Usage: make seed-price-list GROUP_ID=<uuid>"; exit 1; end
	go run ./internal/scripts/price_list_sub_group/seed-price-list.go --group-id=$(GROUP_ID) \
		$(if $(COUNT),--count=$(COUNT),) \
		$(if $(PRICE_MIN),--price-min=$(PRICE_MIN),) \
		$(if $(PRICE_MAX),--price-max=$(PRICE_MAX),) \
		$(if $(PRODUCT_GROUPS),"--product-groups=$(PRODUCT_GROUPS)",) \
		$(if $(GROUP_ITEMS),"--group-items=$(GROUP_ITEMS)",) \
		$(if $(SUBGROUP_KEYS),"--subgroup-keys=$(SUBGROUP_KEYS)",) \
		$(if $(SUBGROUP_KEY),"--subgroup-key=$(SUBGROUP_KEY)",) \
		$(if $(OUTPUT),"--output=$(OUTPUT)",) \
		$(if $(SEED),"--seed=$(SEED)",) \
		$(if $(EXECUTE),"--execute=$(EXECUTE)",) \
		$(if $(DATABASE),"--connection-string=$(DATABASE)",)

seed-price-list-formulas:
	go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go \
		$(if $(INPUT),--input=$(INPUT),) \
		$(if $(OUTPUT),--output=$(OUTPUT),) \
		$(if $(EXECUTE),--execute=$(EXECUTE),) \
		$(if $(database),--connection-string=$(CONNECTION_STRING),)
		
