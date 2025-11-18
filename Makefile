SHELL := /usr/bin/fish

.PHONY: test test-integration tidy seed-price-list-test

test:
	go test ./...

test-integration:
	go test -v -tags=integration ./internal/repositories/priceList

tidy:
	go mod tidy

.PHONY: seed-price-list seed-price-list-test

seed-price-list:
	@test -n "$(GROUP_ID)"; or begin; echo "GROUP_ID is required. Usage: make seed-price-list GROUP_ID=<uuid>"; exit 1; end
	go run ./internal/scripts/seed-price-list.go --group-id=$(GROUP_ID) \
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
