package utils

import (
	"fmt"
	"gopkg.in/ini.v1"
	"strconv"
)

type Config struct {
	Language string
	AutoLock bool
}

func ReadIniFile() Config {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Print("Không thể mở file INI: %v", err)
		return Config{}
	}

	var appCfg Config

	section := cfg.Section("Startup")
	appCfg.Language = section.Key("Language").String()
	appCfg.AutoLock, _ = strconv.ParseBool(section.Key("AutoLock").String())

	return appCfg
}

func WriteIniFile(appConfig Config) {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		return
	}

	section := cfg.Section("Startup")
	// Ghi giá trị mới vào file INI
	section.Key("Language").SetValue(appConfig.Language)
	section.Key("AutoLock").SetValue(strconv.FormatBool(appConfig.AutoLock))

	err = cfg.SaveTo("config.ini")
	if err != nil {
		fmt.Print("Không thể ghi vào file INI: %v", err)
	}
}
