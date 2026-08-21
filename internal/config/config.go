package config

import (
	"errors"
	"fmt"

	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const appDirName string = "podtrawl"
const configFileName string = "config.toml"

type Feed struct {
	Url string `toml:"url"`
}

type Config struct {
	Feeds []Feed `toml:"feed"`
}

func load(path string) (*Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

var ErrLoad = errors.New("unable to load config")

func Get(path *string) (*Config, error) {
	if path != nil {
		// Explicit path; only ever use it
		if config, err := load(*path); err != nil {
			fmt.Println(err)
		} else {
			return config, nil
		}
	} else {
		// Try loading the config from the user's config directory
		if ucd, err := os.UserConfigDir(); err != nil {
			fmt.Println(err)
		} else {
			configPath := filepath.Join(ucd, appDirName, configFileName)
			if config, err := load(configPath); err != nil {
				fmt.Println(err)
			} else {
				return config, nil
			}
		}
		// Try loading the config file from next to the executable
		if executable, err := os.Executable(); err != nil {
			fmt.Println(err)
		} else {
			configPath := filepath.Join(filepath.Dir(executable), configFileName)
			if config, err := load(configPath); err != nil {
				fmt.Println(err)
			} else {
				return config, nil
			}
		}
	}
	return nil, ErrLoad
}
