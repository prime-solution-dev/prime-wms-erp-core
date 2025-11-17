package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type productGroupConfig struct {
	Code  string
	Items []string
}

// SeedConfig describes the inputs for generating seed statements.
type SeedConfig struct {
	Count                int
	GroupID              uuid.UUID
	PriceMin             float64
	PriceMax             float64
	ProductGroups        []productGroupConfig
	ExplicitSubgroupKeys []string
	RandomSeed           int64
}

// GeneratedSubGroup holds data for inspection in tests.
type GeneratedSubGroup struct {
	ID                        uuid.UUID
	PriceListGroupID          uuid.UUID
	SubGroupKey               string
	IsTrading                 bool
	PriceUnit                 float64
	ExtraPriceUnit            float64
	TermPriceUnit             float64
	TotalNetPriceUnit         float64
	PriceWeight               float64
	ExtraPriceWeight          float64
	TermPriceWeight           float64
	TotalNetPriceWeight       float64
	BeforePriceUnit           float64
	BeforeExtraPriceUnit      float64
	BeforeTermPriceUnit       float64
	BeforeTotalNetPriceUnit   float64
	BeforePriceWeight         float64
	BeforeExtraPriceWeight    float64
	BeforeTermPriceWeight     float64
	BeforeTotalNetPriceWeight float64
}

// SeedResult contains the generated SQL statements and raw records for tests.
type SeedResult struct {
	SubGroupStatements    []string
	SubGroupKeyStatements []string
	Records               []GeneratedSubGroup
}

var groupSegmentRegexp = regexp.MustCompile(`GROUP_(\d+)_`)

func main() {
	var (
		count            = flag.Int("count", 10, "Number of price_list_sub_group records to generate")
		groupIDRaw       = flag.String("group-id", "", "Existing price_list_group_id (UUID) [required]")
		priceMin         = flag.Float64("price-min", 0, "Minimum price/value to generate")
		priceMax         = flag.Float64("price-max", 1000, "Maximum price/value to generate")
		productGroupsRaw = flag.String("product-groups", "", "Comma-separated PRODUCT_GROUP definitions (PRODUCT_GROUP1 or PRODUCT_GROUP1:GROUP_1_ITEM_1|GROUP_1_ITEM_2)")
		groupItemsRaw    = flag.String("group-items", "", "JSON map or @path to JSON describing items per product group (e.g. {\"PRODUCT_GROUP1\":[\"GROUP_1_ITEM_1\"]})")
		subgroupKeysCSV  = flag.String("subgroup-keys", "", "Comma-separated list of explicit subgroup_key values")
		randomSeed       = flag.Int64("seed", time.Now().UnixNano(), "Seed for random generator")
		outputPath       = flag.String("output", "", "Optional output file path (defaults to stdout)")
		executeSeed      = flag.Bool("execute", false, "Execute generated statements against the configured database")
		databaseName     = flag.String("database", "prime_erp", "Database name suffix (reads database_gorm_url_<name> from env)")
	)

	var repeatedKeys multiStringFlag
	flag.Var(&repeatedKeys, "subgroup-key", "Explicit subgroup_key value (repeatable)")

	flag.Parse()

	if *groupIDRaw == "" {
		exitWithError(errors.New("flag --group-id is required"))
	}
	groupID, err := uuid.Parse(*groupIDRaw)
	if err != nil {
		exitWithError(fmt.Errorf("invalid --group-id: %w", err))
	}

	if *count <= 0 {
		exitWithError(errors.New("flag --count must be greater than zero"))
	}

	if *priceMin < 0 || *priceMax < 0 {
		exitWithError(errors.New("--price-min and --price-max must be non-negative"))
	}

	if *priceMin > *priceMax {
		exitWithError(fmt.Errorf("--price-min cannot be greater than --price-max (%.2f > %.2f)", *priceMin, *priceMax))
	}

	groupItems, err := parseGroupItems(*groupItemsRaw)
	if err != nil {
		exitWithError(err)
	}

	productGroups, err := parseProductGroups(*productGroupsRaw, groupItems)
	if err != nil {
		exitWithError(err)
	}

	explicitKeys := mergeSubgroupKeys(*subgroupKeysCSV, repeatedKeys)

	if len(productGroups) == 0 && len(explicitKeys) == 0 {
		exitWithError(errors.New("either --product-groups or --subgroup-key/--subgroup-keys must be provided"))
	}

	cfg := SeedConfig{
		Count:                *count,
		GroupID:              groupID,
		PriceMin:             *priceMin,
		PriceMax:             *priceMax,
		ProductGroups:        productGroups,
		ExplicitSubgroupKeys: explicitKeys,
		RandomSeed:           *randomSeed,
	}

	result, err := GenerateSeedStatements(cfg)
	if err != nil {
		exitWithError(err)
	}

	writer := os.Stdout
	if *outputPath != "" {
		file, err := os.Create(*outputPath)
		if err != nil {
			exitWithError(fmt.Errorf("failed to create output file: %w", err))
		}
		defer file.Close()
		writer = file
	}

	fmt.Fprintf(writer, "-- Auto-generated seed (%s)\n", time.Now().UTC().Format(time.RFC3339))
	for _, stmt := range result.SubGroupStatements {
		fmt.Fprintln(writer, stmt)
	}
	for _, stmt := range result.SubGroupKeyStatements {
		fmt.Fprintln(writer, stmt)
	}

	if *executeSeed {
		if err := executeSeedStatements(*databaseName, result); err != nil {
			exitWithError(err)
		}
		fmt.Fprintf(os.Stderr, "Executed seed statements against database %s\n", *databaseName)
	}
}

// GenerateSeedStatements builds INSERT statements according to the configuration.
func GenerateSeedStatements(cfg SeedConfig) (SeedResult, error) {
	randSrc := rand.New(rand.NewSource(cfg.RandomSeed))
	result := SeedResult{}
	groupKeyPool := cfg.ExplicitSubgroupKeys

	for i := 0; i < cfg.Count; i++ {
		subGroupID := uuid.New()
		subGroupKey, keyEntries, err := resolveSubGroupKey(i, groupKeyPool, cfg.ProductGroups, randSrc)
		if err != nil {
			return SeedResult{}, err
		}

		record := GeneratedSubGroup{
			ID:                        subGroupID,
			PriceListGroupID:          cfg.GroupID,
			SubGroupKey:               subGroupKey,
			IsTrading:                 randSrc.Intn(2) == 0,
			PriceUnit:                 randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			ExtraPriceUnit:            randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			TermPriceUnit:             randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			TotalNetPriceUnit:         randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			PriceWeight:               randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			ExtraPriceWeight:          randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			TermPriceWeight:           randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			TotalNetPriceWeight:       randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforePriceUnit:           randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforeExtraPriceUnit:      randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforeTermPriceUnit:       randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforeTotalNetPriceUnit:   randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforePriceWeight:         randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforeExtraPriceWeight:    randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforeTermPriceWeight:     randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
			BeforeTotalNetPriceWeight: randomPrice(randSrc, cfg.PriceMin, cfg.PriceMax),
		}

		result.Records = append(result.Records, record)

		subGroupStmt := fmt.Sprintf(
			"INSERT INTO public.price_list_sub_group (id, price_list_group_id, subgroup_key, is_trading, price_unit, extra_price_unit, term_price_unit, total_net_price_unit, price_weight, extra_price_weight, term_price_weight, total_net_price_weight, before_price_unit, before_extra_price_unit, before_term_price_unit, before_total_net_price_unit, before_price_weight, before_extra_price_weight, before_term_price_weight, before_total_net_price_weight) VALUES (%s, %s, %s, %t, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s);",
			formatUUID(record.ID),
			formatUUID(record.PriceListGroupID),
			formatString(record.SubGroupKey),
			record.IsTrading,
			formatFloat(record.PriceUnit),
			formatFloat(record.ExtraPriceUnit),
			formatFloat(record.TermPriceUnit),
			formatFloat(record.TotalNetPriceUnit),
			formatFloat(record.PriceWeight),
			formatFloat(record.ExtraPriceWeight),
			formatFloat(record.TermPriceWeight),
			formatFloat(record.TotalNetPriceWeight),
			formatFloat(record.BeforePriceUnit),
			formatFloat(record.BeforeExtraPriceUnit),
			formatFloat(record.BeforeTermPriceUnit),
			formatFloat(record.BeforeTotalNetPriceUnit),
			formatFloat(record.BeforePriceWeight),
			formatFloat(record.BeforeExtraPriceWeight),
			formatFloat(record.BeforeTermPriceWeight),
			formatFloat(record.BeforeTotalNetPriceWeight),
		)
		result.SubGroupStatements = append(result.SubGroupStatements, subGroupStmt)

		for _, entry := range keyEntries {
			keyStmt := fmt.Sprintf(
				"INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq) VALUES (%s, %s, %s, %s, %d);",
				formatUUID(uuid.New()),
				formatUUID(record.ID),
				formatString(entry.Code),
				formatString(entry.Value),
				entry.Seq,
			)
			result.SubGroupKeyStatements = append(result.SubGroupKeyStatements, keyStmt)
		}
	}

	return result, nil
}

type subGroupKeyEntry struct {
	Code  string
	Value string
	Seq   int
}

func resolveSubGroupKey(idx int, explicitKeys []string, productGroups []productGroupConfig, randSrc *rand.Rand) (string, []subGroupKeyEntry, error) {
	if len(explicitKeys) > 0 {
		key := strings.TrimSpace(explicitKeys[idx%len(explicitKeys)])
		if key == "" {
			return "", nil, errors.New("explicit subgroup_key cannot be empty")
		}
		entries, err := buildEntriesFromExplicitKey(key)
		if err != nil {
			return "", nil, err
		}
		return key, entries, nil
	}

	if len(productGroups) == 0 {
		return "", nil, errors.New("no product groups available for generating subgroup_key")
	}

	parts := make([]string, 0, len(productGroups))
	entries := make([]subGroupKeyEntry, 0, len(productGroups))

	for i, pg := range productGroups {
		val, err := resolveGroupItem(pg, randSrc)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, val)
		entries = append(entries, subGroupKeyEntry{
			Code:  pg.Code,
			Value: val,
			Seq:   i + 1,
		})
	}

	return strings.Join(parts, "|"), entries, nil
}

func resolveGroupItem(pg productGroupConfig, randSrc *rand.Rand) (string, error) {
	if len(pg.Items) > 0 {
		return pg.Items[randSrc.Intn(len(pg.Items))], nil
	}

	groupNumber, err := extractGroupNumber(pg.Code)
	if err != nil {
		return "", fmt.Errorf("cannot derive group number for %s: %w", pg.Code, err)
	}

	itemIdx := randSrc.Intn(99) + 1
	return fmt.Sprintf("GROUP_%d_ITEM_%d", groupNumber, itemIdx), nil
}

func buildEntriesFromExplicitKey(key string) ([]subGroupKeyEntry, error) {
	segments := strings.Split(key, "|")
	if len(segments) == 0 {
		return nil, fmt.Errorf("invalid subgroup_key: %s", key)
	}

	entries := make([]subGroupKeyEntry, 0, len(segments))
	for idx, raw := range segments {
		segment := strings.TrimSpace(raw)
		if segment == "" {
			return nil, fmt.Errorf("invalid subgroup_key: empty segment in %s", key)
		}

		groupNumber, err := extractGroupNumber(segment)
		if err != nil {
			return nil, fmt.Errorf("unable to determine product group for %s: %w", segment, err)
		}

		entries = append(entries, subGroupKeyEntry{
			Code:  fmt.Sprintf("PRODUCT_GROUP%d", groupNumber),
			Value: segment,
			Seq:   idx + 1,
		})
	}

	return entries, nil
}

func extractGroupNumber(value string) (int, error) {
	matches := groupSegmentRegexp.FindStringSubmatch(value)
	if len(matches) < 2 {
		return 0, fmt.Errorf("value %s does not contain GROUP_X pattern", value)
	}
	var num int
	_, err := fmt.Sscanf(matches[1], "%d", &num)
	if err != nil {
		return 0, fmt.Errorf("invalid group number in %s: %w", value, err)
	}
	return num, nil
}

func randomPrice(r *rand.Rand, min, max float64) float64 {
	if max-min < 0.0001 {
		return min
	}
	return min + r.Float64()*(max-min)
}

func formatUUID(id uuid.UUID) string {
	return fmt.Sprintf("'%s'::uuid", id.String())
}

func formatFloat(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

func formatString(val string) string {
	escaped := strings.ReplaceAll(val, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func mergeSubgroupKeys(csv string, repeated multiStringFlag) []string {
	result := make([]string, 0, len(repeated)+1)

	if strings.TrimSpace(csv) != "" {
		for _, part := range strings.Split(csv, ",") {
			value := strings.TrimSpace(part)
			if value != "" {
				result = append(result, value)
			}
		}
	}

	result = append(result, repeated...)

	return result
}

func parseProductGroups(raw string, extras map[string][]string) ([]productGroupConfig, error) {
	if strings.TrimSpace(raw) == "" && len(extras) == 0 {
		return nil, nil
	}

	configs := map[string]*productGroupConfig{}

	if strings.TrimSpace(raw) != "" {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			entry := strings.TrimSpace(part)
			if entry == "" {
				continue
			}
			config, err := parseProductGroupEntry(entry)
			if err != nil {
				return nil, err
			}
			configs[config.Code] = &config
		}
	}

	for code, items := range extras {
		if cfg, ok := configs[code]; ok {
			if len(cfg.Items) == 0 {
				cfg.Items = append([]string{}, items...)
			}
		} else {
			configs[code] = &productGroupConfig{
				Code:  code,
				Items: append([]string{}, items...),
			}
		}
	}

	result := make([]productGroupConfig, 0, len(configs))
	for _, cfg := range configs {
		result = append(result, *cfg)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Code < result[j].Code
	})

	return result, nil
}

func parseProductGroupEntry(entry string) (productGroupConfig, error) {
	segments := strings.SplitN(entry, ":", 2)
	code := strings.TrimSpace(segments[0])
	if code == "" {
		return productGroupConfig{}, fmt.Errorf("invalid product group entry: %s", entry)
	}

	cfg := productGroupConfig{Code: code}
	if len(segments) == 2 {
		itemsRaw := strings.TrimSpace(segments[1])
		if itemsRaw != "" {
			cfg.Items = splitAndTrim(itemsRaw, "|")
		}
	}
	return cfg, nil
}

func parseGroupItems(raw string) (map[string][]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string][]string{}, nil
	}

	if strings.HasPrefix(raw, "@") {
		path := strings.TrimPrefix(raw, "@")
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read group-items file: %w", err)
		}
		raw = string(content)
	}

	parsedMap := map[string][]string{}
	if err := json.Unmarshal([]byte(raw), &parsedMap); err == nil {
		return normalizeGroupItems(parsedMap), nil
	}

	arrayParsed, err := parseGroupItemsArray(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group-items JSON: %w", err)
	}
	return normalizeGroupItems(arrayParsed), nil
}

func executeSeedStatements(databaseName string, result SeedResult) error {
	gormDB, err := db.ConnectGORM(databaseName)
	if err != nil {
		return fmt.Errorf("failed to connect database_gorm_url_%s: %w", databaseName, err)
	}
	defer db.CloseGORM(gormDB)

	return ApplySeedStatements(gormDB, result)
}

// ApplySeedStatements persists generated statements inside a transaction.
func ApplySeedStatements(gormDB *gorm.DB, result SeedResult) error {
	if gormDB == nil {
		return errors.New("gorm DB instance is nil")
	}

	tx := gormDB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	if err := execStatements(tx, result.SubGroupStatements, "price_list_sub_group"); err != nil {
		tx.Rollback()
		return err
	}

	if err := execStatements(tx, result.SubGroupKeyStatements, "price_list_sub_group_key"); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func execStatements(tx *gorm.DB, statements []string, label string) error {
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute %s statement: %w", label, err)
		}
	}
	return nil
}

func parseGroupItemsArray(raw string) (map[string][]string, error) {
	var entries []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, err
	}

	result := map[string][]string{}
	for _, entry := range entries {
		if len(entry) == 0 {
			continue
		}

		if codeVal, ok := entry["code"]; ok {
			code, ok := codeVal.(string)
			if !ok || strings.TrimSpace(code) == "" {
				return nil, fmt.Errorf("array entry missing valid code field")
			}
			itemsVal, ok := entry["items"]
			if !ok {
				return nil, fmt.Errorf("array entry for %s missing items field", code)
			}
			items, err := interfaceToStringSlice(itemsVal)
			if err != nil {
				return nil, fmt.Errorf("invalid items for %s: %w", code, err)
			}
			result[code] = append(result[code], items...)
			continue
		}

		for code, values := range entry {
			if strings.TrimSpace(code) == "" {
				continue
			}
			items, err := interfaceToStringSlice(values)
			if err != nil {
				return nil, fmt.Errorf("invalid items for %s: %w", code, err)
			}
			result[code] = append(result[code], items...)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no group items found in array")
	}

	return result, nil
}

func interfaceToStringSlice(value interface{}) ([]string, error) {
	list, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected array value")
	}

	result := make([]string, 0, len(list))
	for _, item := range list {
		str, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("expected string value")
		}
		if trimmed := strings.TrimSpace(str); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}

func normalizeGroupItems(items map[string][]string) map[string][]string {
	normalized := make(map[string][]string, len(items))
	for key, values := range items {
		if strings.TrimSpace(key) == "" {
			continue
		}
		normalized[key] = deduplicate(values)
	}
	return normalized
}

func splitAndTrim(raw string, sep string) []string {
	items := strings.Split(raw, sep)
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func deduplicate(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, v := range values {
		value := strings.TrimSpace(v)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
