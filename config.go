package gobot

import (
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	ConfigFile = "config.yaml"
)

type Config struct {
	Tuling Tuling `yaml:"tuling"`
}

type Rebot struct {
	Name string `yaml: "Name"`
	Key  string `yaml:"key"`
}
type Tuling struct {
	URL  string           `yaml:"url"`
	Keys map[string]Rebot `yaml:"keys"`
}

func Load() Config {
	var cfg Config
	if bb, err := ioutil.ReadFile(ConfigFile); err != nil {
		panic(err)
	} else {
		if err := yaml.Unmarshal(bb, &cfg); err != nil {
			panic(err)
		}
	}

	return cfg
}
