package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type CommonConfig struct {
	UserGrpc         string `mapstructure:"USER_GRPC"`
	ImageGrpc        string `mapstructure:"IMAGE_GRPC"`
	StoreGrpc        string `mapstructure:"STORE_GRPC"`
	NotificationGrpc string `mapstructure:"NOTIFICATION_GRPC"`
	ProductGrpc      string `mapstructure:"PRODUCT_GRPC"`
}
type Config struct {
	MODE     string `mapstructure:"MODE"`
	DB_URL   string `mapstructure:"DB_URL"`
	HTTPPort string `mapstructure:"HTTP_PORT"`
	GRPCPort string `mapstructure:"GRPC_PORT"`

	CommonConfig `mapstructure:",squash"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile("./config/config.yaml")
	// viper.SetConfigFile("/go/src/app/config/config.yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var Config Config
	if err := viper.Unmarshal(&Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	viper.SetConfigFile("./config/common-config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := viper.Unmarshal(&Config.CommonConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	var commonConfig CommonConfig
	if err := viper.Unmarshal(&commonConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	Config.CommonConfig = commonConfig

	log.Printf("Config: %+v", Config)

	return &Config, nil
}
