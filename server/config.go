package server

import (
	"fmt"

	"github.com/caarlos0/env/v10"
)

// Config represents Main service configuration
type Config struct {
	Port int `env:"RP_SERVER_PORT" envDefault:"8080"`
}

// LoadConfig loads configuration from provided file and serializes it into RpConfig struct
func LoadConfig(cfg interface{}) error {
	err := env.Parse(cfg)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return err
	}

	return nil
}

// EmptyConfig creates empty config
func EmptyConfig() *Config {
	return &Config{}
}
