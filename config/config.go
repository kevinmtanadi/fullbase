package config

// import (
// 	"encoding/json"
// 	"fmt"
// 	"funcbase/constants"
// 	"os"
// 	"reflect"
// 	"sync"
// 	"time"
// )

// type Config struct {
// 	AppName             string   `json:"app_name"`
// 	AppURL              string   `json:"app_url"`
// 	APIKey              string   `json:"api_key"`
// 	AllowedOrigins      []string `json:"allowed_origins"`
// 	AutomatedBackup     bool     `json:"automated_backup"`
// 	CronSchedule        string   `json:"cron_schedule"`
// 	DBMaxOpenConnection int      `json:"db_max_open_connection"`
// 	DBMaxIdleConnection int      `json:"db_max_idle_connection"`
// 	DBMaxLifetime       int      `json:"db_max_lifetime"`
// }

// var (
// 	instance *Config
// 	once     sync.Once
// )

// func GetInstance() *Config {
// 	once.Do(func() {
// 		instance = &Config{}
// 		instance.Load()
// 	})

// 	return instance
// }

// var configPath string = fmt.Sprintf("%s/%s", constants.DATA_PATH, constants.CONFIG_PATH)

// func (c *Config) Load() error {
// 	file, err := os.Open(configPath)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			config := Config{
// 				AppName: "Funcbase",
// 				AppURL:  "https://funcbase.com",
// 				APIKey:  "default-api-key",
// 				AllowedOrigins: []string{
// 					"http://localhost:8080",
// 					"http://localhost:3000",
// 				},
// 				AutomatedBackup:     false,
// 				CronSchedule:        "",
// 				DBMaxOpenConnection: 10,
// 				DBMaxIdleConnection: 5,
// 				DBMaxLifetime:       2,
// 			}
// 			config.Save()

// 			c = &config
// 		}

// 		return err
// 	}
// 	defer file.Close()

// 	decoder := json.NewDecoder(file)
// 	if err := decoder.Decode(&c); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (c *Config) Save() error {
// 	file, err := os.Create(configPath)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	encoder := json.NewEncoder(file)
// 	if err := encoder.Encode(c); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (c *Config) Get(key string) interface{} {
// 	val := reflect.ValueOf(c).Elem()
// 	typ := val.Type()

// 	for i := 0; i < val.NumField(); i++ {
// 		field := typ.Field(i)
// 		tag := field.Tag.Get("json")

// 		if tag == key {
// 			return val.Field(i).Interface()
// 		}
// 	}
// 	return nil
// }

// func (c *Config) Set(key string, value interface{}) error {
// 	val := reflect.ValueOf(c).Elem()
// 	typ := val.Type()

// 	for i := 0; i < val.NumField(); i++ {
// 		field := typ.Field(i)
// 		tag := field.Tag.Get("json")

// 		if tag == key {
// 			fieldValue := val.Field(i)
// 			if fieldValue.CanSet() {
// 				newValue := reflect.ValueOf(value)
// 				if newValue.Type().AssignableTo(fieldValue.Type()) {
// 					fieldValue.Set(newValue)
// 					return nil
// 				} else {
// 					return fmt.Errorf("cannot assign value of type %s to field of type %s", newValue.Type(), fieldValue.Type())
// 				}
// 			} else {
// 				return fmt.Errorf("cannot set value to field %s", field.Name)
// 			}
// 		}
// 	}
// 	return fmt.Errorf("no field found with json tag %s", key)
// }

// func (c *Config) WatchCronChanges(callback func()) {
// 	go func() {
// 		originalConfig := *c
// 		for {
// 			time.Sleep(1 * time.Minute) // Adjust the duration as needed
// 			c.Load()
// 			if originalConfig.CronSchedule != c.CronSchedule ||
// 				originalConfig.AutomatedBackup != c.AutomatedBackup {
// 				callback()
// 				originalConfig = *c
// 			}

// 		}
// 	}()
// }
