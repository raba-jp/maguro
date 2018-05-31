package config

import (
	"io/ioutil"
	"log"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Channels     []string     `yaml:"channels"`
	Repositories []Repository `yaml:"repositories"`
	Schedules    []Schedule   `yaml:"schedules"`
}

type Repository struct {
	Name string   `yaml:"name"`
	Env  []string `yaml:"env"`
}

type Schedule struct {
	Name string `yaml:"name"`
	Cron string `yaml:"cron"`
}

func LoadConfig() (*Config, error) {
	buf, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Printf("failed read config file: %s", err)
		return nil, err
	}
	var config Config
	if err = yaml.Unmarshal(buf, &config); err != nil {
		log.Printf("failed read config file: %s", err)
		return nil, err
	}
	return &config, nil
}
