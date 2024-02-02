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
	Data         map[string]string
	DatabaseName string
	Config       *ConfigType
}

func (DB *YamlDatabase) Init() {
	if DB.DatabaseName == "" {
		DB.DatabaseName = "db.yaml"
	}
	defer DB.PrivateInitialize()
	if DB.Config.Debug {
		log.Println("db.init (yaml)")
	}
	yamlFile, err := os.ReadFile(DB.DatabaseName)
	if err != nil {
		// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
		DB.Data = map[string]string{}
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
	if DB.Config.Debug {
		log.Println("db.init - complete")
	}
	DB.Initialized = true
}

func (DB *YamlDatabase) PrivateInitialize() {
	if DB.Data == nil {
		DB.Data = map[string]string{}
		DB.Initialized = true
		if DB.Config.Debug {
			log.Println("db.init - recovered")
		}

	}
}

func (DB *YamlDatabase) Set(key string, value interface{}) {
	if !DB.Initialized {
		panic("Unable to set. db not initialized()")
	}
	// https://aguidehub.com/blog/2022-08-28-golang-convert-interface-to-string/?expand_article=1
	DB.Data[key] = fmt.Sprint(value)
	DB.Write()
}

func (DB *YamlDatabase) Get(key string) (string, bool) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	// https://stackoverflow.com/questions/27545270/how-to-get-a-value-from-map
	value, ok := DB.Data[key]
	return value, ok
}

func (DB *YamlDatabase) Write() {
	//https://gobyexample.com/writing-files
	// https://stackoverflow.com/questions/65207143/writing-the-contents-of-a-struct-to-yml-file
	if DB.Config.Debug {
		log.Printf("Writing: %+v\n", DB.Data)
	}

	file, err := os.OpenFile(DB.DatabaseName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("error opening/creating file: %v", err)
	}
	defer file.Close()
	enc := yaml.NewEncoder(file)
	err = enc.Encode(DB.Data)
	if err != nil {
		log.Fatalf("error encoding: %v", err)
	}

}

func (DB *YamlDatabase) Keys() []string {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	keys := make([]string, len(DB.Data))

	i := 0
	for k := range DB.Data {
		keys[i] = k
		i++
	}
	return keys
}

func (DB *YamlDatabase) Delete(key string) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	delete(DB.Data, key)
}

func (DB *YamlDatabase) IsInitialized() bool {
	return DB.Initialized
}

func (DB *YamlDatabase) Close() {
	if !DB.Initialized {
		panic("Unable to close. db not initialized()")
	}
	DB.Write()
	if DB.Config.Debug {
		log.Println("db.Closed")
	}
}
