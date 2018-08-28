package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config defines a struct to match a configuration yaml file.
type Config struct {
	AlarmTime string `yaml:"AlarmTime"`
	I2CAddr   uint8  `yaml:"I2CAddr"`
	I2CBus    int    `yaml:"I2CBus"`
}

// NewConfig will create a new Config instance from the specified yaml file
func NewConfig(yamlFile string) (*Config, error) {
	config := Config{}
	source, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(source, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
