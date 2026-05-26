package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func (e *Eval) builtinPdoOpen(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pdo/open requires 2 arguments (driver dsn)")
	}
	driver, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("pdo/open: driver must be a string")
	}
	dsn, ok := args[1].(String)
	if !ok {
		return nil, fmt.Errorf("pdo/open: dsn must be a string")
	}

	driverName := string(driver)
	switch driverName {
	case "sqlite":
		driverName = "sqlite"
	case "mysql":
		driverName = "mysql"
	case "postgres", "postgresql":
		driverName = "pgx"
	default:
		return nil, fmt.Errorf("pdo/open: unsupported driver '%s' (use: sqlite, mysql, postgres)", string(driver))
	}

	db, err := sql.Open(driverName, string(dsn))
	if err != nil {
		return nil, fmt.Errorf("pdo/open: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pdo/open: %v", err)
	}
	return &PdoConnection{DB: db}, nil
}

func (e *Eval) builtinPdoExec(args []Value) (Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("pdo/exec requires at least 2 arguments (conn sql [params...])")
	}
	conn, ok := args[0].(*PdoConnection)
	if !ok {
		return nil, fmt.Errorf("pdo/exec: first argument must be a pdo-connection")
	}
	sqlStr, ok := args[1].(String)
	if !ok {
		return nil, fmt.Errorf("pdo/exec: sql must be a string")
	}

	params := make([]interface{}, 0, len(args)-2)
	for _, a := range args[2:] {
		params = append(params, valueToInterface(a))
	}

	result, err := conn.DB.Exec(string(sqlStr), params...)
	if err != nil {
		return nil, fmt.Errorf("pdo/exec: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	return Integer(rowsAffected), nil
}

func (e *Eval) builtinPdoQuery(args []Value) (Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("pdo/query requires at least 2 arguments (conn sql [params...])")
	}
	conn, ok := args[0].(*PdoConnection)
	if !ok {
		return nil, fmt.Errorf("pdo/query: first argument must be a pdo-connection")
	}
	sqlStr, ok := args[1].(String)
	if !ok {
		return nil, fmt.Errorf("pdo/query: sql must be a string")
	}

	params := make([]interface{}, 0, len(args)-2)
	for _, a := range args[2:] {
		params = append(params, valueToInterface(a))
	}

	rows, err := conn.DB.Query(string(sqlStr), params...)
	if err != nil {
		return nil, fmt.Errorf("pdo/query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("pdo/query: %v", err)
	}

	var result []Value
	for rows.Next() {
		row := make([]interface{}, len(columns))
		rowPtrs := make([]interface{}, len(columns))
		for i := range row {
			rowPtrs[i] = &row[i]
		}
		if err := rows.Scan(rowPtrs...); err != nil {
			return nil, fmt.Errorf("pdo/query: %v", err)
		}
		// Build alist for this row
		var rowList Value = Nil
		for i := len(columns) - 1; i >= 0; i-- {
			colName := columns[i]
			colVal := interfaceToValue(row[i])
			rowList = &Cons{
				Car: &Cons{Car: String(colName), Cdr: colVal},
				Cdr: rowList,
			}
		}
		result = append(result, rowList)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pdo/query: %v", err)
	}
	return SliceToList(result), nil
}

func (e *Eval) builtinPdoClose(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("pdo/close requires 1 argument")
	}
	conn, ok := args[0].(*PdoConnection)
	if !ok {
		return nil, fmt.Errorf("pdo/close: argument must be a pdo-connection")
	}
	if err := conn.DB.Close(); err != nil {
		return nil, fmt.Errorf("pdo/close: %v", err)
	}
	return Nil, nil
}

func valueToInterface(v Value) interface{} {
	switch val := v.(type) {
	case Integer:
		return int64(val)
	case Float:
		return float64(val)
	case String:
		return string(val)
	case Boolean:
		return bool(val)
	case *NilType:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}

func interfaceToValue(v interface{}) Value {
	if v == nil {
		return Nil
	}
	switch val := v.(type) {
	case int64:
		return Integer(val)
	case float64:
		if val == float64(int64(val)) && !strings.Contains(fmt.Sprintf("%g", val), ".") {
			return Integer(int64(val))
		}
		return Float(val)
	case string:
		return String(val)
	case bool:
		return Boolean(val)
	case []byte:
		return String(string(val))
	default:
		return String(fmt.Sprintf("%v", val))
	}
}
