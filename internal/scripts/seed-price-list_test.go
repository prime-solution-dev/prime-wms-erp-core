package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var postgresContainer tc.Container

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "postgres:16",
		Env:          map[string]string{"POSTGRES_PASSWORD": "test", "POSTGRES_USER": "test", "POSTGRES_DB": "testdb"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		fmt.Printf("failed to start postgres container: %v\n", err)
		os.Exit(1)
	}
	postgresContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Printf("failed to get host: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("failed to get mapped port: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mappedPort.Port())
	os.Setenv("database_gorm_url_prime_erp", dsn)

	if err := createSeedTestSchema(); err != nil {
		fmt.Printf("failed to create schema: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func createSeedTestSchema() error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	statements := []string{
		`CREATE TABLE IF NOT EXISTS price_list_group (
			id uuid PRIMARY KEY,
			company_code text,
			site_code text,
			group_code text,
			price_unit double precision,
			price_weight double precision,
			before_price_unit double precision,
			before_price_weight double precision,
			currency text,
			effective_date timestamp NULL,
			remark text,
			group_key text,
			create_by text,
			create_dtm timestamp,
			update_by text,
			update_dtm timestamp
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group (
			id uuid PRIMARY KEY,
			price_list_group_id uuid REFERENCES price_list_group(id),
			subgroup_key text,
			is_trading boolean,
			price_unit double precision,
			extra_price_unit double precision,
			total_net_price_unit double precision,
			price_weight double precision,
			extra_price_weight double precision,
			term_price_weight double precision,
			total_net_price_weight double precision,
			before_price_unit double precision,
			before_extra_price_unit double precision,
			before_total_net_price_unit double precision,
			before_price_weight double precision,
			before_extra_price_weight double precision,
			before_term_price_weight double precision,
			before_total_net_price_weight double precision,
			effective_date timestamp NULL,
			remark text,
			create_by text,
			create_dtm timestamp,
			update_by text,
			update_dtm timestamp,
			udf_json json NULL
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group_key (
			id uuid PRIMARY KEY,
			sub_group_id uuid REFERENCES price_list_sub_group(id),
			code text,
			value text,
			seq integer
		);`,
	}

	for _, stmt := range statements {
		if err := gormx.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func TestGenerateSeedStatementsWithExplicitKey(t *testing.T) {
	cfg := SeedConfig{
		Count:                 1,
		GroupID:               uuid.New(),
		PriceMin:              0,
		PriceMax:              100,
		ExplicitSubgroupKeys:  []string{"GROUP_1_ITEM_3|GROUP_2_ITEM_1|GROUP_4_ITEM_2|GROUP_6_ITEM_2"},
		GroupItemCombinations: nil,
		RandomSeed:            123,
	}

	result, err := GenerateSeedStatements(cfg)
	require.NoError(t, err)
	require.Len(t, result.SubGroupStatements, 1)
	require.Len(t, result.SubGroupKeyStatements, 4)
	require.Contains(t, result.SubGroupStatements[0], "GROUP_1_ITEM_3|GROUP_2_ITEM_1|GROUP_4_ITEM_2|GROUP_6_ITEM_2")

	for _, stmt := range result.SubGroupKeyStatements {
		require.NotEmpty(t, stmt)
		require.Contains(t, stmt, "INSERT INTO public.price_list_sub_group_key")
	}
}

func TestGeneratedSQLExecutesAgainstPostgres(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	groupID := uuid.New()
	err = gormx.Exec("INSERT INTO price_list_group (id, company_code, site_code, group_code, price_unit, price_weight, before_price_unit, before_price_weight, currency) VALUES (?, 'TEST', 'TEST', 'GROUP', 0, 0, 0, 0, 'THB')", groupID).Error
	require.NoError(t, err)

	cfg := SeedConfig{
		Count:    2,
		GroupID:  groupID,
		PriceMin: 0,
		PriceMax: 1000,
		ProductGroups: []productGroupConfig{
			{Code: "PRODUCT_GROUP1", Items: []string{"GROUP_1_ITEM_1"}},
			{Code: "PRODUCT_GROUP2", Items: []string{"GROUP_2_ITEM_1"}},
			{Code: "PRODUCT_GROUP4", Items: []string{"GROUP_4_ITEM_2"}},
		},
		GroupItemCombinations: nil,
		RandomSeed:            42,
	}

	result, err := GenerateSeedStatements(cfg)
	require.NoError(t, err)
	require.Len(t, result.SubGroupStatements, 2)
	require.Len(t, result.SubGroupKeyStatements, 6)

	for _, stmt := range result.SubGroupStatements {
		require.NoError(t, gormx.Exec(stmt).Error)
	}
	for _, stmt := range result.SubGroupKeyStatements {
		require.NoError(t, gormx.Exec(stmt).Error)
	}

	var subGroups []models.PriceListSubGroup
	require.NoError(t, gormx.Find(&subGroups).Error)
	require.Len(t, subGroups, 2)

	for _, sg := range subGroups {
		require.GreaterOrEqual(t, sg.PriceUnit, cfg.PriceMin)
		require.LessOrEqual(t, sg.PriceUnit, cfg.PriceMax)
		require.NotEmpty(t, sg.SubgroupKey)
	}

	var keys []models.PriceListSubGroupKey
	require.NoError(t, gormx.Find(&keys).Error)
	require.Len(t, keys, 6)

	for _, key := range keys {
		require.Contains(t, []string{"PRODUCT_GROUP1", "PRODUCT_GROUP2", "PRODUCT_GROUP4"}, key.Code)
		require.NotEmpty(t, key.Value)
	}
}

func TestParseGroupItemsSupportsMapAndArray(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string][]string
	}{
		{
			name:  "map format with deduplication",
			input: `{"PRODUCT_GROUP1":["GROUP_1_ITEM_1","GROUP_1_ITEM_1","GROUP_1_ITEM_2"]}`,
			expected: map[string][]string{
				"PRODUCT_GROUP1": []string{"GROUP_1_ITEM_1", "GROUP_1_ITEM_2"},
			},
		},
		{
			name: "array format with code/items",
			input: `[
				{"code":"PRODUCT_GROUP1","items":["GROUP_1_ITEM_1","GROUP_1_ITEM_2"]},
				{"code":"PRODUCT_GROUP2","items":["GROUP_2_ITEM_3"]}
			]`,
			expected: map[string][]string{
				"PRODUCT_GROUP1": []string{"GROUP_1_ITEM_1", "GROUP_1_ITEM_2"},
				"PRODUCT_GROUP2": []string{"GROUP_2_ITEM_3"},
			},
		},
		{
			name: "array format with implicit codes",
			input: `[
				{"PRODUCT_GROUP1":["GROUP_1_ITEM_1"]},
				{"PRODUCT_GROUP3":["GROUP_3_ITEM_9","GROUP_3_ITEM_9"]}
			]`,
			expected: map[string][]string{
				"PRODUCT_GROUP1": []string{"GROUP_1_ITEM_1"},
				"PRODUCT_GROUP3": []string{"GROUP_3_ITEM_9"},
			},
		},
		{
			name: "array format mixed entries",
			input: `[
				{"code":"PRODUCT_GROUP5","items":["GROUP_5_ITEM_10"]},
				{"PRODUCT_GROUP6":["GROUP_6_ITEM_2","GROUP_6_ITEM_3"]}
			]`,
			expected: map[string][]string{
				"PRODUCT_GROUP5": []string{"GROUP_5_ITEM_10"},
				"PRODUCT_GROUP6": []string{"GROUP_6_ITEM_2", "GROUP_6_ITEM_3"},
			},
		},
		{
			name: "array entries containing multiple product groups",
			input: `[
				{
					"PRODUCT_GROUP1":["GROUP_1_ITEM_4"],
					"PRODUCT_GROUP2":["GROUP_2_ITEM_6"]
				},
				{
					"PRODUCT_GROUP1":["GROUP_1_ITEM_5"],
					"PRODUCT_GROUP2":["GROUP_2_ITEM_7"]
				}
			]`,
			expected: map[string][]string{
				"PRODUCT_GROUP1": []string{"GROUP_1_ITEM_4", "GROUP_1_ITEM_5"},
				"PRODUCT_GROUP2": []string{"GROUP_2_ITEM_6", "GROUP_2_ITEM_7"},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := parseGroupItems(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestApplySeedStatementsRequiresDB(t *testing.T) {
	err := ApplySeedStatements(nil, SeedResult{})
	require.Error(t, err)
}

func TestGenerateSeedStatementsWithGroupItemCombinations(t *testing.T) {
	cfg := SeedConfig{
		Count:    2,
		GroupID:  uuid.New(),
		PriceMin: 0,
		PriceMax: 100,
		ProductGroups: []productGroupConfig{
			{Code: "PRODUCT_GROUP1"},
			{Code: "PRODUCT_GROUP2"},
			{Code: "PRODUCT_GROUP4"},
		},
		GroupItemCombinations: []map[string][]string{
			{
				"PRODUCT_GROUP1": {"GROUP_1_ITEM_4"},
				"PRODUCT_GROUP2": {"GROUP_2_ITEM_6"},
				"PRODUCT_GROUP4": {"GROUP_4_ITEM_6"},
			},
			{
				"PRODUCT_GROUP1": {"GROUP_1_ITEM_4"},
				"PRODUCT_GROUP2": {"GROUP_2_ITEM_6"},
				"PRODUCT_GROUP4": {"GROUP_4_ITEM_7"},
			},
		},
		RandomSeed: 123,
	}

	result, err := GenerateSeedStatements(cfg)
	require.NoError(t, err)
	require.Len(t, result.SubGroupStatements, 2)
	require.Len(t, result.SubGroupKeyStatements, 6) // 2 records * 3 product groups

	// First record should have the first combination
	require.Contains(t, result.SubGroupStatements[0], "GROUP_1_ITEM_4|GROUP_2_ITEM_6|GROUP_4_ITEM_6")

	// Second record should have the second combination
	require.Contains(t, result.SubGroupStatements[1], "GROUP_1_ITEM_4|GROUP_2_ITEM_6|GROUP_4_ITEM_7")

	// Verify key entries
	require.Len(t, result.Records, 2)
	require.Equal(t, "GROUP_1_ITEM_4|GROUP_2_ITEM_6|GROUP_4_ITEM_6", result.Records[0].SubGroupKey)
	require.Equal(t, "GROUP_1_ITEM_4|GROUP_2_ITEM_6|GROUP_4_ITEM_7", result.Records[1].SubGroupKey)
}

func TestParseGroupItemsReturnsCombinations(t *testing.T) {
	input := `[
		{
			"PRODUCT_GROUP1": ["GROUP_1_ITEM_4"],
			"PRODUCT_GROUP2": ["GROUP_2_ITEM_6"]
		},
		{
			"PRODUCT_GROUP1": ["GROUP_1_ITEM_5"],
			"PRODUCT_GROUP2": ["GROUP_2_ITEM_7"]
		}
	]`

	merged, combinations, err := parseGroupItems(input)
	require.NoError(t, err)

	// Check merged map (order may vary, so check individual items)
	require.Len(t, merged, 2)
	require.Contains(t, merged, "PRODUCT_GROUP1")
	require.Contains(t, merged, "PRODUCT_GROUP2")
	require.ElementsMatch(t, []string{"GROUP_1_ITEM_4", "GROUP_1_ITEM_5"}, merged["PRODUCT_GROUP1"])
	require.ElementsMatch(t, []string{"GROUP_2_ITEM_6", "GROUP_2_ITEM_7"}, merged["PRODUCT_GROUP2"])

	// Check combinations array
	require.Len(t, combinations, 2)
	require.Equal(t, map[string][]string{
		"PRODUCT_GROUP1": {"GROUP_1_ITEM_4"},
		"PRODUCT_GROUP2": {"GROUP_2_ITEM_6"},
	}, combinations[0])
	require.Equal(t, map[string][]string{
		"PRODUCT_GROUP1": {"GROUP_1_ITEM_5"},
		"PRODUCT_GROUP2": {"GROUP_2_ITEM_7"},
	}, combinations[1])
}
