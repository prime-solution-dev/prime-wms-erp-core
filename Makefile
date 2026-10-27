SHELL := /usr/bin/fish

.PHONY: test test-integration test-integration-group test-integration-prepurchase test-integration-purchase test-integration-unit test-integration-pricelist tidy seed-price-list-test seed-price-list-formulas

test:
	go test ./...

test-integration:
	go test -v -tags=integration ./internal/repositories/priceList

test-integration-group:
	go test -v -tags=integration ./internal/services/group-service

test-integration-prepurchase:
	go test -v -tags=integration ./internal/repositories/prePurchase

test-integration-purchase:
	go test -v -tags=integration ./internal/repositories/purchase

test-integration-unit:
	go test -v -tags=integration ./internal/repositories/unit

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
		

MIGRATE = go run ./cmd/migrate

# DB_URL บน command line มีลำดับเหนือ .env; ส่งมาแต่ว่าง = error ทันที ไม่ fallback ไป .env
ifeq ($(origin DB_URL),command line)
ifeq ($(strip $(value DB_URL)),)
$(error DB_URL is empty)
endif
# ใช้ $(value) เพื่อรับ DB_URL ตามตัวอักษร ไม่ให้ make ขยาย $ — รหัสผ่านที่มี $ ใช้ได้เลย แค่ครอบด้วย single quote ใน shell
export database_sqlx_url_prime_erp := $(value DB_URL)
endif

.PHONY: migrate-up migrate-down migrate-status migrate-create migrate-reset migrate-fix \
	seed-master-up seed-master-down seed-master-status seed-master-create \
	seed-dev-up seed-dev-down seed-dev-status seed-dev-create \
	seed-up seed-down seed-status seed-create \
	db-setup

# ---- schema (migrations/, ตาราง goose_db_version) ----
migrate-up:
	$(MIGRATE) -command up

migrate-down:
	$(MIGRATE) -command down

migrate-status:
	$(MIGRATE) -command status

# make migrate-create NAME=add_x_column
migrate-create:
	$(MIGRATE) -command create -name "$(NAME)"

# make migrate-reset CONFIRM=yes  (ห้ามใช้บน prod)
migrate-reset:
	$(MIGRATE) -command reset -confirm "$(CONFIRM)"

migrate-fix:
	$(MIGRATE) -command fix

# ---- seed master (seeds/master/, ตาราง goose_seed_master_version) — ทุก env รวม prod ----
seed-master-up:
	$(MIGRATE) -set master -command up

seed-master-down:
	$(MIGRATE) -set master -command down

seed-master-status:
	$(MIGRATE) -set master -command status

seed-master-create:
	$(MIGRATE) -set master -command create -name "$(NAME)"

# ---- seed dev (seeds/dev/, ตาราง goose_seed_dev_version) — ห้ามใช้บน prod, up/down ต้อง CONFIRM=yes ----
seed-dev-up:
	$(MIGRATE) -set dev -command up -confirm "$(CONFIRM)"

seed-dev-down:
	$(MIGRATE) -set dev -command down -confirm "$(CONFIRM)"

seed-dev-status:
	$(MIGRATE) -set dev -command status

seed-dev-create:
	$(MIGRATE) -set dev -command create -name "$(NAME)"

# ---- seed folder ใดก็ได้ใต้ seeds/ เช่น make seed-up SET=Pacifica-uat (ตาราง goose_seed_<slug>_version) ----
seed-up:
	$(MIGRATE) -set "$(SET)" -command up -confirm "$(CONFIRM)"

seed-down:
	$(MIGRATE) -set "$(SET)" -command down -confirm "$(CONFIRM)"

seed-status:
	$(MIGRATE) -set "$(SET)" -command status

# make seed-create SET=Pacifica-uat NAME=add_x — สร้าง folder ให้ถ้ายังไม่มี
seed-create:
	$(MIGRATE) -set "$(SET)" -command create -name "$(NAME)"

# ---- สำหรับ CI/deploy: schema แล้วตามด้วย seed master ----
db-setup:
	$(MIGRATE) -command up
	$(MIGRATE) -set master -command up
