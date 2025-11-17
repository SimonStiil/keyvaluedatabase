package main

import (
	"fmt"
	"os"
	"strings"

	"database/sql"

	"github.com/SimonStiil/keyvaluedatabase/rest"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

type PostgresDatabase struct {
	Initialized  bool
	Connection   *sql.DB
	Config       *ConfigPostgres
	Password     string
	DatabaseName string
}

type ConfigPostgres struct {
	Address         string `mapstructure:"address"`
	Username        string `mapstructure:"username"`
	DatabaseName    string `mapstructure:"databaseName"`
	SystemTableName string `mapstructure:"systemTableName"`
	EnvVariableName string `mapstructure:"envVariableName"`
	KeyName         string `mapstructure:"keyName"`
	ValueName       string `mapstructure:"valueName"`
	SSLMode         string `mapstructure:"sslMode"`
}

func PostgresGetDefaults(configReader *viper.Viper) {
	configReader.SetDefault("postgres.address", "localhost:5432")
	configReader.SetDefault("postgres.username", "kvdb")
	configReader.SetDefault("postgres.databaseName", "")
	configReader.SetDefault("postgres.systemTableName", "kvdb")
	configReader.SetDefault("postgres.envVariableName", BaseENVname+"_POSTGRES_PASSWORD")
	configReader.SetDefault("postgres.keyName", "key")
	configReader.SetDefault("postgres.valueName", "value")
	configReader.SetDefault("postgres.sslMode", "disable")
}

func (PDB *PostgresDatabase) Init() {
	logger.Debug("Initializing PostgreSQL", "function", "Init", "struct", "PostgresDatabase")
	if PDB.Config.DatabaseName == "" {
		PDB.DatabaseName = PDB.Config.Username
	} else {
		PDB.DatabaseName = PDB.Config.DatabaseName
	}
	PDB.Password = os.Getenv(PDB.Config.EnvVariableName)
	var err error
	connectionString := fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=%v",
		strings.Split(PDB.Config.Address, ":")[0],
		strings.Split(PDB.Config.Address, ":")[1],
		PDB.Config.Username, PDB.Password, PDB.DatabaseName, PDB.Config.SSLMode)

	logger.Debug("Connection string (password hidden)", "function", "Init", "struct", "PostgresDatabase")
	PDB.Connection, err = sql.Open("postgres", connectionString)
	if err != nil {
		panic(err.Error())
	}

	// Test connection
	err = PDB.Connection.Ping()
	if err != nil {
		panic(err.Error())
	}

	// Create system table
	_, err = PDB.Connection.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%v" ( 
		"%v" CHAR(%v) PRIMARY KEY, 
		"%v" VARCHAR(%v) NOT NULL)`,
		PDB.Config.SystemTableName, PDB.Config.KeyName, rest.KeyMaxLength,
		PDB.Config.ValueName, rest.ValueMaxLength))
	if err != nil {
		panic(err.Error())
	}
	PDB.Initialized = true
	err = PDB.CreateNamespace(PDB.GetSystemNS())
	if err != nil {
		panic(err.Error())
	}
	logger.Debug("Initialization complete", "function", "Init", "struct", "PostgresDatabase")
}

func (PDB *PostgresDatabase) GetSystemNS() string {
	return PDB.Config.SystemTableName
}

func (PDB *PostgresDatabase) Set(namespace string, key string, value interface{}) error {
	if !PDB.Initialized {
		panic("F Unable to set. db not initialized()")
	}
	PDB.CreateNamespace(namespace)
	statement, err := PDB.Connection.Prepare(fmt.Sprintf(`INSERT INTO "%v" ("%v", "%v") VALUES ($1, $2) 
		ON CONFLICT ("%v") DO UPDATE SET "%v"=$2`,
		namespace, PDB.Config.KeyName, PDB.Config.ValueName,
		PDB.Config.KeyName, PDB.Config.ValueName))
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(key, value)
	if err != nil {
		return err
	}
	return nil
}

func (PDB *PostgresDatabase) Get(namespace string, key string) (string, error) {
	if !PDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	rows, err := PDB.Connection.Query(fmt.Sprintf(`SELECT * FROM "%v" WHERE "%v" = $1`,
		namespace, PDB.Config.KeyName), key)

	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return "", &ErrNotFound{Value: namespace}
		}
		logger.Error("Query failed with error", "function", "Get", "struct", "PostgresDatabase", "namespace", namespace, "error", err)
		return "", err
	}
	defer rows.Close()
	kvpair := rest.KVPairV2{}
	found := false
	for rows.Next() {
		err = rows.Scan(&kvpair.Key, &kvpair.Value)
		if err != nil {
			logger.Error("Scan row failed with error", "function", "Get", "struct", "PostgresDatabase", "namespace", namespace, "error", err)
			return "", err
		}
		found = true
	}
	if found {
		return kvpair.Value, err
	} else {
		return "", &ErrNotFound{Value: key}
	}
}

func (PDB *PostgresDatabase) Keys(namespace string) ([]string, error) {
	if !PDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	var rows *sql.Rows
	var err error
	if namespace == "" {
		rows, err = PDB.Connection.Query(`SELECT tablename FROM pg_catalog.pg_tables 
			WHERE schemaname = 'public'`)
	} else {
		rows, err = PDB.Connection.Query(fmt.Sprintf(`SELECT "%v" FROM "%v"`,
			PDB.Config.KeyName, namespace))
	}
	if err != nil {
		logger.Error("Query failed with error", "function", "Keys", "struct", "PostgresDatabase", "namespace", namespace, "error", err)
		return nil, err
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			logger.Error("Scan row failed with error", "function", "Keys", "struct", "PostgresDatabase", "namespace", namespace, "error", err)
			return keys, err
		}
		keys = append(keys, strings.TrimSpace(key))
	}
	return keys, nil
}

func (PDB *PostgresDatabase) DeleteKey(namespace string, key string) error {
	if !PDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	stmt, err := PDB.Connection.Prepare(fmt.Sprintf(`DELETE FROM "%v" WHERE "%v" = $1`,
		namespace, PDB.Config.KeyName))
	if err != nil {
		logger.Error("Prepare failed with error", "function", "DeleteKey", "struct", "PostgresDatabase", "namespace", namespace, "key", key, "error", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(key)
	if err != nil {
		logger.Error("Exec failed with error", "function", "DeleteKey", "struct", "PostgresDatabase", "namespace", namespace, "key", key, "error", err)
		return err
	}
	return nil
}

func (PDB *PostgresDatabase) CreateNamespace(namespace string) error {
	if !PDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	result, err := PDB.Connection.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%v" ( 
		"%v" CHAR(%v) PRIMARY KEY, 
		"%v" VARCHAR(%v) NOT NULL)`,
		namespace, PDB.Config.KeyName, rest.KeyMaxLength,
		PDB.Config.ValueName, rest.ValueMaxLength))
	logger.Debug("Create table if not exists", "function", "createTable", "struct", "PostgresDatabase", "namespace", namespace, "result", result)
	if err != nil {
		logger.Error("Error creating table", "function", "createTable", "struct", "PostgresDatabase", "namespace", namespace, "error", err)
		return err
	}
	return nil
}

func (PDB *PostgresDatabase) DeleteNamespace(namespace string) error {
	if !PDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	if namespace == PDB.GetSystemNS() {
		return &ErrNotAllowed{Value: fmt.Sprintf("delete System NS %v", namespace)}
	}
	_, err := PDB.Connection.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%v"`, namespace))
	if err != nil {
		logger.Error("Exec failed with error", "function", "Delete", "struct", "PostgresDatabase", "namespace", namespace, "error", err)
		return err
	}
	return nil
}

func (PDB *PostgresDatabase) IsInitialized() bool {
	return PDB.Initialized
}

func (PDB *PostgresDatabase) Close() {
	if !PDB.Initialized {
		panic("F Unable to close. db not initialized()")
	}
	err := PDB.Connection.Close()
	if err != nil {
		panic(err)
	}
	logger.Debug("Closed database connection", "function", "Close", "struct", "PostgresDatabase")
}
