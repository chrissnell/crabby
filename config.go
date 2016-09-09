package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config is the base of our configuration data structure
type Config struct {
	Jobs     []Job          `yaml:"jobs"`
	Selenium SeleniumConfig `yaml:"selenium"`
}

// SeleniumConfig holds the configuration for our Selenium service
type SeleniumConfig struct {
	URL string `yaml:"url"`
}

// NewConfig creates an new config object from the given filename.
func NewConfig(filename string) (Config, error) {
	cfgFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	c := Config{}
	err = yaml.Unmarshal(cfgFile, &c)
	if err != nil {
		return Config{}, err
	}

	return c, nil
}
