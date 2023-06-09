package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen string `yaml:"listen"`
	OIDC   []OIDC `yaml:"oidc"`
}

type OIDC struct {
	Provider     string `yaml:"provider"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
}

var config *Config

func loadConfig(configPath string) error {
	config = &Config{}
	configF, err := os.Open(configPath)
	if err != nil {
		return err
	}
	return yaml.NewDecoder(configF).Decode(config)
}
