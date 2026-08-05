package db

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DatabaseManage struct {
	dbName       string
	ginContext   *gin.Context
	sqlxInstance *sqlx.DB
	tx           *sqlx.Tx
}

func ConnectSqlx(databaseName string) (*sqlx.DB, error) {
	dabaseUrl := os.Getenv(fmt.Sprintf("database_sqlx_url_%s", databaseName))
	if dabaseUrl == `` {
		return nil, fmt.Errorf("not found database_sqlx_url")
	}

	sqlxInstance, err := sqlx.Connect("postgres", dabaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	return sqlxInstance, nil
}

func NewDatabaseManage(dbName string, ginCtx *gin.Context) (*DatabaseManage, error) {
	sqlxInstance, err := ConnectSqlx(dbName)
	if err != nil {
		return nil, err
	}

	return &DatabaseManage{
		dbName:       dbName,
		ginContext:   ginCtx,
		sqlxInstance: sqlxInstance,
	}, nil
}

func (dbManage *DatabaseManage) Execute(query string) error {
	_, err := dbManage.sqlxInstance.QueryxContext(context.Background(), query)

	return err
}

func ExecuteQuery(db *sqlx.DB, query string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	rows, err := db.QueryxContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %v", err)
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]

			switch val := val.(type) {
			case []byte:
				strVal := string(val)
				if f, err := strconv.ParseFloat(strVal, 64); err == nil {
					v = f
				} else {
					if parsedUUID, err := uuid.FromBytes(val); err == nil {
						v = parsedUUID.String()
					} else {
						v = strVal
					}
				}
			case string:
				if parsedUUID, err := uuid.Parse(val); err == nil {
					v = parsedUUID.String()
				} else {
					v = val
				}
			case float64:
				v = float64(val)
			case int32:
				v = int(val)
			case int64:
				v = int64(val)
			default:
				v = val
			}

			row[col] = v
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %v", err)
	}

	return results, nil
}

func (dbManage *DatabaseManage) ExecuteQueryWithPanding(query string) ([]map[string]interface{}, error) {
	queryParams := dbManage.ginContext.Request.URL.Query()

	pageNum := 0
	pageSizeNum := 0

	if page, pageExists := queryParams["page"]; pageExists {
		var err error
		pageNum, err = strconv.Atoi(page[0])
		if err != nil {
			return nil, fmt.Errorf("invalid value for 'page'")
		}
	}

	if pageSize, pageSizeExists := queryParams["pageSize"]; pageSizeExists {
		var err error
		pageSizeNum, err = strconv.Atoi(pageSize[0])
		if err != nil {
			return nil, fmt.Errorf("invalid value for 'pageSize'")
		}
	}

	offset := (pageNum - 1) * pageSizeNum

	var queryString string

	if pageNum > 0 && pageSizeNum > 0 {
		queryString = fmt.Sprintf("%s limit %d offset %d", query, pageSizeNum, offset)
	} else {
		queryString = query
	}

	return dbManage.ExecuteQuery(queryString)
}

func (dbManage *DatabaseManage) BeginTransaction() error {
	var err error
	if dbManage.tx == nil {
		dbManage.tx, err = dbManage.sqlxInstance.Beginx()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}
	} else {
		return fmt.Errorf("transaction already in progress")
	}
	return nil
}

func (dbManage *DatabaseManage) CommitTransaction() error {
	if dbManage.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}

	err := dbManage.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	dbManage.tx = nil
	return nil
}

func (dbManage *DatabaseManage) RollbackTransaction() error {
	if dbManage.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}

	err := dbManage.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %v", err)
	}
	dbManage.tx = nil
	return nil
}

func (dbManage *DatabaseManage) ExecuteInTransaction(query string) error {
	if dbManage.tx == nil {
		return fmt.Errorf("no transaction started")
	}

	_, err := dbManage.tx.ExecContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("transactional query execution failed: %v", err)
	}
	return nil
}

func (dbManage *DatabaseManage) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	rows, err := dbManage.sqlxInstance.QueryxContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	fmt.Println(rows)
	defer rows.Close()

	columns, err := rows.Columns()
	fmt.Println(columns)
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %v", err)
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]

			switch val := val.(type) {
			case []byte:
				if parsedUUID, err := uuid.FromBytes(val); err == nil {
					v = parsedUUID.String()
				} else {
					strVal := string(val)
					if f, err := strconv.ParseFloat(strVal, 64); err == nil {
						v = f
					} else {
						v = strVal
					}
				}
			case string:
				if parsedUUID, err := uuid.Parse(val); err == nil {
					v = parsedUUID.String()
				} else {
					v = val
				}
			case float64:
				v = float64(val)
			case int32:
				v = int(val)
			case int64:
				v = int64(val)
			default:
				v = val
			}

			row[col] = v
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %v", err)
	}

	return results, nil
}
