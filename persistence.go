package main

import (
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
)

func readState(path string) (*Config, error) {
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	err = yaml.Unmarshal(dat, config)
	return config, err
}

func saveState(path string, config *Config) error {
	events := *config.Thermostat.Events
	config.Thermostat.Events = nil

	dat, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	config.Thermostat.Events = &events

	return ioutil.WriteFile(path, dat, os.FileMode(int(0660)))
}
