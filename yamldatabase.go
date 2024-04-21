package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type YamlDatabase struct {
	Initialized  bool
	Data         map[string]map[string]string
	SystemNS     string `mapstructure:"systemnamespace"`
	DatabaseName string
}

func (DB *YamlDatabase) GetSystemNS() string {
	return DB.SystemNS
}

func (DB *YamlDatabase) Init() {
	if DB.DatabaseName == "" {
		DB.DatabaseName = "db.yaml"
	}
	if DB.SystemNS == "" {
		DB.SystemNS = "kvdb"
	}
	defer DB.PrivateInitialize()

	logger.Debug("Initializing Yaml Database", "function", "Init", "struct", "YamlDatabase")
	yamlFile, err := os.ReadFile(DB.DatabaseName)
	if err != nil {
		// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
		DB.Data = map[string]map[string]string{}
	} else {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic: %+v\n", r)
			}
		}()
		err = yaml.Unmarshal(yamlFile, &DB.Data)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

	}
	logger.Debug("Initialization complete", "function", "Init", "struct", "YamlDatabase")
	DB.Initialized = true
}

func (DB *YamlDatabase) PrivateInitialize() {
	if DB.Data == nil {
		DB.Data = map[string]map[string]string{}
		DB.Initialized = true

		logger.Debug("recovered", "function", "PrivateInitialize", "struct", "YamlDatabase")
	}
}

func (DB *YamlDatabase) Set(namespace string, key string, value interface{}) error {
	if !DB.Initialized {
		panic("Unable to set. db not initialized()")
	}
	// https://aguidehub.com/blog/2022-08-28-golang-convert-interface-to-string/?expand_article=1
	if _, ok := DB.Data[namespace]; !ok {
		DB.Data[namespace] = map[string]string{}
	}
	DB.Data[namespace][key] = fmt.Sprint(value)
	return DB.Write()
}

func (DB *YamlDatabase) CreateNamespace(namespace string) error {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	if _, ok := DB.Data[namespace]; !ok {
		DB.Data[namespace] = map[string]string{}
	}
	return nil
}

func (DB *YamlDatabase) DeleteNamespace(namespace string) error {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	if namespace == DB.GetSystemNS() {
		return &ErrNotAllowed{Value: fmt.Sprintf("delete System NS %v", namespace)}
	}
	delete(DB.Data, namespace)
	return nil
}

func (DB *YamlDatabase) Get(namespace string, key string) (string, error) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	// https://stackoverflow.com/questions/27545270/how-to-get-a-value-from-map
	if _, ok := DB.Data[namespace]; !ok {
		return "", fmt.Errorf("namespace not found %v", namespace)
	}
	value, ok := DB.Data[namespace][key]
	if ok {
		return value, nil
	} else {
		return "", &ErrNotFound{Value: key}
	}
}

func (DB *YamlDatabase) Write() error {
	//https://gobyexample.com/writing-files
	// https://stackoverflow.com/questions/65207143/writing-the-contents-of-a-struct-to-yml-file
	logger.Debug(fmt.Sprintf("Writing: %+v\n", DB.Data), "function", "Write", "struct", "YamlDatabase")

	file, err := os.OpenFile(DB.DatabaseName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		logger.Error("error opening/creating file", "function", "Write", "struct", "YamlDatabase", "error", err)
		return err
	}
	defer file.Close()
	enc := yaml.NewEncoder(file)
	err = enc.Encode(DB.Data)
	if err != nil {
		logger.Error("error encoding", "function", "Write", "struct", "YamlDatabase", "error", err)
		return err
	}
	return nil
}

func (DB *YamlDatabase) Keys(namespace string) ([]string, error) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	var length int
	if namespace == "" {
		length = len(DB.Data)
	} else {
		length = len(DB.Data[namespace])
	}
	keys := make([]string, length)

	i := 0
	if namespace == "" {
		for k := range DB.Data {
			keys[i] = k
			i++
		}
	} else {
		for k := range DB.Data[namespace] {
			keys[i] = k
			i++
		}
	}
	return keys, nil
}

func (DB *YamlDatabase) DeleteKey(namespace string, key string) error {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	delete(DB.Data[namespace], key)
	return nil
}

func (DB *YamlDatabase) IsInitialized() bool {
	return DB.Initialized
}

func (DB *YamlDatabase) Close() {
	if !DB.Initialized {
		panic("Unable to close. db not initialized()")
	}
	DB.Write()
	logger.Debug("Closed database connection", "function", "Close", "struct", "YamlDatabase")
}
