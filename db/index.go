package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const migrationsDirectoryName = "migrations"

type AddTableParams struct {
	Name string `json:"name"`
}

type DeleteTableParams struct {
	Name string `json:"name"`
}

type AddColumnParams struct {
	Table        string `json:"table"`
	Column       string `json:"column"`
	Type         string `json:"type"`
	IsNullable   bool   `json:"isNullable"`
	DefaultValue string `json:"defaultValue"`
}

type DeleteColumnParams struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

type RenameColumnParams struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

type Action struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Migration struct {
	SchemaVersion string   `json:"schemaVersion"`
	Id            string   `json:"id"`
	Description   string   `json:"description"`
	Actions       []Action `json:"actions"`
}

type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	IsNullable   bool   `json:"isNullable"`
	DefaultValue string `json:"defaultValue"`
}

type ColumnName string

type Table struct {
	Name        string   `json:"name"`
	Columns     []Column `json:"columns"`
	PrimaryKeys []ColumnName
}

type Snapshot struct {
	Tables []Table `json:"tables"`
}

func GetMigrationsDirectoryPath() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	directory := filepath.Join(pwd, migrationsDirectoryName)
	return directory, nil
}

func AddMigration(description string) error {

	id := time.Now().UTC().Format("20060102150405")
	fileName := id + ".json"
	migration := Migration{
		SchemaVersion: "1",
		Id:            id,
		Description:   description,
		Actions:       []Action{},
	}

	migrationsDir, err := GetMigrationsDirectoryPath()
	if err != nil {
		return err
	}

	//TODO: add checking usage of instance name
	if _, err := os.Stat(migrationsDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		err = os.Mkdir(migrationsDir, 0777)
		if err != nil {
			return err
		}
	}

	packedMigration, err := json.MarshalIndent(migration, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(migrationsDir, fileName), packedMigration, 0777)
}

func getMigrationPath(id string) (string, error) {

	migrationsDirectory, err := GetMigrationsDirectoryPath()
	if err != nil {
		return "", err
	}

	migrationPath := filepath.Join(migrationsDirectory, id+".json")
	return migrationPath, nil
}

func GetText(id string) (string, error) {

	migrationPath, err := getMigrationPath(id)
	if err != nil {
		return "", nil
	}

	migration, err := ioutil.ReadFile(migrationPath)
	return string(migration), nil
}

func Get(id string) (*Migration, error) {
	rawMigration, err := GetText(id)
	if err != nil {
		return nil, err
	}

	var migration Migration
	err = json.Unmarshal(([]byte)(rawMigration), &migration)

	if err != nil {
		return nil, fmt.Errorf("can't parse migration: %v/n", err)
	}

	return &migration, nil
}

func GetList() (*[]Migration, error) {

	migrationsDirectoryPath, err := GetMigrationsDirectoryPath()
	if err != nil {
		return nil, err
	}

	configsPathPattern := filepath.Join(migrationsDirectoryPath, "*.json")
	files, err := filepath.Glob(configsPathPattern)
	sort.Strings(files)

	if err != nil {
		return nil, err
	}

	result := []Migration{}

	for _, migrationPath := range files {
		_, fileName := filepath.Split(migrationPath)
		migrationId := strings.TrimSuffix(fileName, ".json")

		migration, err := Get(migrationId)
		if err != nil {
			return nil, fmt.Errorf("can't read migration %v/n", err)
		}

		result = append(result, *migration)
	}

	return &result, err
}

func addActionToMigrationFile(method string, params interface{}) (string, error) {

	migrations, err := GetList()
	if err != nil {
		return "", fmt.Errorf("can't get migration %v/n", err)
	}

	migrationsSize := len(*migrations)
	if migrationsSize == 0 {
		return "", fmt.Errorf("migration doesn't exist, please add migration/n")
	}

	packedParams, _ := json.MarshalIndent(params, "", "  ")

	lastMigration := (*migrations)[migrationsSize-1]
	lastMigration.Actions = append(lastMigration.Actions, Action{
		Method: method,
		Params: (json.RawMessage)(packedParams),
	})

	packedMigration, _ := json.MarshalIndent(lastMigration, "", "  ")
	migrationPath, _ := getMigrationPath(lastMigration.Id)
	err = ioutil.WriteFile(migrationPath, packedMigration, 0777)
	if err != nil {
		return "", fmt.Errorf("can't write migration/n")
	}

	return lastMigration.Id, nil
}

func AddTable(tableName string) (string, error) {
	if strings.TrimSpace(tableName) == "" {
		return "", fmt.Errorf("table name is required /n")
	}

	params := AddTableParams{
		Name: tableName,
	}

	return addActionToMigrationFile("addTable", params)
}

func DeleteTable(tableName string) (string, error) {
	if strings.TrimSpace(tableName) == "" {
		return "", fmt.Errorf("table name is required /n")
	}

	params := DeleteTableParams{
		Name: tableName,
	}

	return addActionToMigrationFile("deleteTable", params)
}

func AddColumn(tableName string, columnName string, columnType string, isNullable bool, defaultValue string) (string, error) {
	if strings.TrimSpace(tableName) == "" {
		return "", fmt.Errorf("table name is required /n")
	}

	if strings.TrimSpace(columnName) == "" {
		return "", fmt.Errorf("column name is required /n")
	}

	if strings.TrimSpace(columnType) == "" {
		return "", fmt.Errorf("column type is required /n")
	}

	params := AddColumnParams{
		Table:        tableName,
		Column:       columnName,
		IsNullable:   isNullable,
		Type:         columnType,
		DefaultValue: defaultValue,
	}

	return addActionToMigrationFile("addColumn", params)
}

func DeleteColumn(tableName string, columnName string) (string, error) {
	if strings.TrimSpace(tableName) == "" {
		return "", fmt.Errorf("table name is required /n")
	}

	if strings.TrimSpace(columnName) == "" {
		return "", fmt.Errorf("column name is required /n")
	}

	params := DeleteColumnParams{
		Table:  tableName,
		Column: columnName,
	}

	return addActionToMigrationFile("deleteColumn", params)
}

func Sync() error {
	migrations, err := GetList()
	if err != nil {
		return fmt.Errorf("can't read migrations: %v/n", err)
	}

	dbConnectionString := fmt.Sprintf("user=%v password=%v dbname=%v host=%v port=%v sslmode=disable",
		"admin",
		"123456",
		"timeio",
		"localhost",
		5432)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return fmt.Errorf("can't connect to db: %v", err)
	}
	defer func() { db.Close() }()

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("can't connect to db: %v", err)
	}

	log.Println("Connected to db")
	transaction, err := db.Begin()
	if err != nil {
		transaction.Rollback()
		return fmt.Errorf("can't start transaction: %v", err)
	}

	err = addMigrationsTableIfNotExist(transaction)
	if err != nil {
		transaction.Rollback()
		return fmt.Errorf("can't add migration table: %v", err)
	}

	currentMigrationId, err := getCurrentSyncedMigrationId(transaction)
	if err != nil {
		transaction.Rollback()
		return fmt.Errorf("can't read current migration state: %v", err)
	}

	isCurrentMigrationPassed := currentMigrationId == ""

	for _, migration := range *migrations {

		if migration.Id == currentMigrationId {
			isCurrentMigrationPassed = true
			continue
		}

		if !isCurrentMigrationPassed {
			continue
		}

		err = applyMigrationActions(transaction, migration)
		if err != nil {
			transaction.Rollback()
			return fmt.Errorf("can't apply migration %v: %v/n", migration.Id, err)
		}

		addMigrationToMigrationsTable(transaction, migration)
		if err != nil {
			transaction.Rollback()
			return fmt.Errorf("can't add migration to migrations table %v: %v/n", migration.Id, err)
		}
	}

	return transaction.Commit()
}

func getCurrentSyncedMigrationId(transaction *sql.Tx) (string, error) {
	row := transaction.QueryRow("SELECT id FROM _migrations  ORDER BY id DESC  LIMIT 1")

	var migrationId string
	err := row.Scan(&migrationId)
	if err == sql.ErrNoRows {
		return "", nil
	}

	return migrationId, err
}

func applyMigrationActions(transaction *sql.Tx, migration Migration) error {
	for _, action := range migration.Actions {
		var err error

		method, params, err := decodeAction(action.Method, action.Params)
		if err != nil {
			return fmt.Errorf("can't decode action %v/n", err)
		}

		switch method {
		case "addTable":
			err = applyAddTable(transaction, params.(AddTableParams))
			break
		case "deleteTable":
			err = applyDeleteTable(transaction, params.(DeleteTableParams))
			break
		case "addColumn":
			err = applyAddColumn(transaction, params.(AddColumnParams))
			break
		case "deleteColumn":
			err = applyDeleteColumn(transaction, params.(DeleteColumnParams))
			break
		}

		if err != nil {
			return fmt.Errorf("can't apply action %v: %v/n", params, err)
		}
	}

	return nil
}

func decodeAction(method string, params json.RawMessage) (string, interface{}, error) {

	var err error
	switch method {
	case "addTable":
		var addTableParams AddTableParams
		err = json.Unmarshal(params, &addTableParams)
		if err != nil {
			return "", nil, err
		}

		return method, addTableParams, nil

	case "deleteTable":
		var deleteTableParams DeleteTableParams
		err = json.Unmarshal(params, &deleteTableParams)
		if err != nil {
			return "", nil, err
		}

		return method, deleteTableParams, nil

	case "addColumn":
		var addColumnParams AddColumnParams
		err = json.Unmarshal(params, &addColumnParams)
		if err != nil {
			return "", nil, err
		}

		return method, addColumnParams, nil

	case "deleteColumn":
		var deleteColumnParams DeleteColumnParams
		err = json.Unmarshal(params, &deleteColumnParams)
		if err != nil {
			return "", nil, err
		}

		return method, deleteColumnParams, nil
	}

	return "", nil, nil
}

func addMigrationsTableIfNotExist(transaction *sql.Tx) error {
	_, err := transaction.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
        	id varchar(255) NOT NULL,
        	data text NOT NULL,
        	PRIMARY KEY (id)
    )`)

	return err
}

func addMigrationToMigrationsTable(transaction *sql.Tx, migration Migration) error {
	packedMigration, _ := json.Marshal(migration)
	_, err := transaction.Exec("INSERT INTO _migrations (id, data) VALUES ($1, $2)", migration.Id, packedMigration)
	return err
}

func applyAddTable(transaction *sql.Tx, params AddTableParams) error {
	query := fmt.Sprintf("CREATE TABLE \"%v\" ();", params.Name)
	_, err := transaction.Exec(query)
	if err != nil {
		return fmt.Errorf("can't create table %v: %v/n", params.Name, err)
	}

	return nil
}

func applyDeleteTable(transaction *sql.Tx, params DeleteTableParams) error {
	query := fmt.Sprintf("DROP TABLE \"%v\"", params.Name)
	_, err := transaction.Exec(query)
	if err != nil {
		return fmt.Errorf("can't delete table %v: %v/n", params.Name, err)
	}

	return nil
}

func applyAddColumn(transaction *sql.Tx, params AddColumnParams) error {
	columnType := params.Type
	notNullParam := ""
	if !params.IsNullable {
		notNullParam = "NOT NULL"
	}

	defaultValueParam := ""
	if params.DefaultValue != "" {
		defaultValueParam = fmt.Sprintf("DEFAULT '%v';", params.DefaultValue)
	}

	query := fmt.Sprintf(`
		ALTER TABLE "%v"
			ADD COLUMN "%v" %v %v %v
	`, params.Table, params.Column, columnType, notNullParam, defaultValueParam)

	fmt.Println(query)
	_, err := transaction.Exec(query)
	if err != nil {
		return fmt.Errorf("can't add column '%v' to table '%v': %v/n", params.Column, params.Table, err)
	}

	return nil
}

func applyDeleteColumn(transaction *sql.Tx, params DeleteColumnParams) error {
	query := fmt.Sprintf(`
		ALTER TABLE "%v"
			DROP COLUMN "%v"
	`, params.Table, params.Column)

	_, err := transaction.Exec(query)
	if err != nil {
		return fmt.Errorf("can't delete column '%v' at table '%v': %v/n", params.Column, params.Table, err)
	}

	return nil
}

func GetSnapshot() (*Snapshot, error) {
	migrations, err := GetList()
	if err != nil {
		return nil, fmt.Errorf("can't read migrations: %v", err)
	}

	snapshot := Snapshot{
		Tables: []Table{},
	}
	for _, migration := range *migrations {

		err = applyMigrationToSnapshot(&snapshot, migration)
		if err != nil {
			return nil, fmt.Errorf("can't apply migration %v to snapshot: %v/n", migration.Id, err)
		}
	}

	return &snapshot, nil
}

func applyMigrationToSnapshot(snapshot *Snapshot, migration Migration) error {
	for _, action := range migration.Actions {
		method, params, err := decodeAction(action.Method, action.Params)
		if err != nil {
			return fmt.Errorf("can't decode action %v/n", err)
		}

		switch method {
		case "addTable":
			err = applyAddTableToSnapshot(snapshot, params.(AddTableParams))
			break
		case "deleteTable":
			err = applyDeleteTableFromSnapshot(snapshot, params.(DeleteTableParams))
			break
		case "addColumn":
			err = applyAddColumnToSnapshot(snapshot, params.(AddColumnParams))
			break
		case "deleteColumn":
			err = applyDeleteColumnFromSnapshot(snapshot, params.(DeleteColumnParams))
			break
		}

		if err != nil {
			return fmt.Errorf("can't apply action %v: %v/n", params, err)
		}
	}

	return nil
}

func getTableFromSnapshot(snapshot *Snapshot, tableName string) *Table {

	tables := snapshot.Tables

	for index := 0; index < len(tables); index++ {
		table := &(tables[index])
		if table.Name == tableName {
			return table
		}
	}

	return nil
}

func applyAddTableToSnapshot(snapshot *Snapshot, params AddTableParams) error {
	existingTable := getTableFromSnapshot(snapshot, params.Name)
	if existingTable != nil {
		return fmt.Errorf("table already exist")
	}

	snapshot.Tables = append(snapshot.Tables, Table{
		Name:        params.Name,
		Columns:     []Column{},
		PrimaryKeys: []ColumnName{},
	})

	return nil
}

func applyDeleteTableFromSnapshot(snapshot *Snapshot, params DeleteTableParams) error {
	tableName := params.Name
	existingTable := getTableFromSnapshot(snapshot, tableName)
	if existingTable == nil {
		return fmt.Errorf("table doesn't exist")
	}

	for index, table := range snapshot.Tables {
		if table.Name != tableName {
			continue
		}

		snapshot.Tables = append(snapshot.Tables[:index], snapshot.Tables[index+1:]...)
	}
	return nil
}

func getColumnFromTable(table *Table, columnName string) *Column {

	columns := table.Columns

	for index := 0; index < len(columns); index++ {
		column := columns[index]

		if column.Name == columnName {
			return &column
		}
	}

	return nil
}

func applyAddColumnToSnapshot(snapshot *Snapshot, params AddColumnParams) error {
	table := getTableFromSnapshot(snapshot, params.Table)
	if table == nil {
		return fmt.Errorf("table doesn't exist")
	}

	column := getColumnFromTable(table, params.Column)
	if column != nil {
		return fmt.Errorf("column already exist")
	}

	table.Columns = append(table.Columns, Column{
		Name:         params.Column,
		Type:         params.Type,
		IsNullable:   params.IsNullable,
		DefaultValue: params.DefaultValue,
	})

	return nil
}

func applyDeleteColumnFromSnapshot(snapshot *Snapshot, params DeleteColumnParams) error {
	table := getTableFromSnapshot(snapshot, params.Table)
	if table == nil {
		return fmt.Errorf("table doesn't exist")
	}

	columnName := params.Column
	column := getColumnFromTable(table, columnName)
	if column == nil {
		return fmt.Errorf("column doesn't exist")
	}

	for index, column := range table.Columns {
		if column.Name != columnName {
			continue
		}

		table.Columns = append(table.Columns[:index], table.Columns[index+1:]...)
	}
	return nil
}
