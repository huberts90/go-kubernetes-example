package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func Load(logger *zap.Logger, filename string) {
	// Read Configuration
	viper.SetConfigFile(filename)
	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatal("Failed to read configuration file", zap.Error(err))
	}
}

func noConfig(key string) {
	panic(fmt.Sprintf("Configuration %s not found", key))
}

func GetString(key string) string {
	if viper.IsSet(key) {
		value := viper.GetString(key)
		if value != "" {
			return value
		}
		panic(fmt.Sprintf("Configuration %s not set", key))
	}
	noConfig(key)
	return ""
}

func GetDurationDefault(key string, d time.Duration) time.Duration {
	if viper.IsSet(key) {
		return viper.GetDuration(key)
	}
	return d
}
