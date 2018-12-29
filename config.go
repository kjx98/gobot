package gobot

//"io/ioutil"
//yaml "gopkg.in/yaml.v2"

const (
	ConfigFile = "config.yaml"
)

type Config struct {
	Tuling Tuling `yaml:"tuling"`
	Uin    string
}

type Tuling struct {
	URL       string `yaml:"url"`
	GroupOnly bool   `yaml:"groupOnly,omitempty"`
	KeyAPI    string `yaml:"APIkey"`
}

/*
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

*/

func NewConfig(key string) Config {
	url := tulingURL

	if key == "" {
		key = "808811ad0fd34abaa6fe800b44a9556a"
	}
	var cfg = Config{Tuling{URL: url,
		GroupOnly: true,
		KeyAPI:    key},
		"",
	}
	return cfg
}
