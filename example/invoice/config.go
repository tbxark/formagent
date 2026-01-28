package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

func loadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Config
	err = json.Unmarshal(file, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}
