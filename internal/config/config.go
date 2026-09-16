package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const appDirName string = "podtrawl"
const configFileName string = "config.toml"

type Common struct {
	Dirname  *string `toml:"dirname"`
	Filename *string `toml:"filename"`
	Sanitize *bool   `toml:"sanitize"`
	Existing *string `toml:"existing"`
}

type Feed struct {
	Common
	Url string `toml:"url"`
}

type Config struct {
	Root     *string `toml:"root"`
	Defaults Common  `toml:"defaults"`
	Feeds    []Feed  `toml:"feed"`
}

func load(path string) (*Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	// Each feed may only be declared once.
	// The ledger is keyed by feed URL,
	// so allowing duplicate entries would cause collisions.
	seen := make(map[string]bool, len(c.Feeds))
	for _, f := range c.Feeds {
		if seen[f.Url] {
			return nil, fmt.Errorf("feed url %q is configured more than once", f.Url)
		}
		seen[f.Url] = true
	}
	return &c, nil
}

var ErrLoad = errors.New("unable to load config")

// Get finds and loads the config file.
func Get(path *string, log *slog.Logger) (*Config, error) {
	errs := []error{ErrLoad}
	// try loads one candidate, recording it if it doesn't work out.
	try := func(configPath string) *Config {
		config, err := load(configPath)
		if err != nil {
			log.Debug("config not loaded", "path", configPath, "err", err)
			errs = append(errs, err)
			return nil
		}
		log.Debug("config loaded", "path", configPath, "feeds", len(config.Feeds))
		return config
	}
	if path != nil {
		// Explicit path; only ever use it
		if config := try(*path); config != nil {
			return config, nil
		}
	} else {
		// Try loading the config from the user's config directory
		if ucd, err := os.UserConfigDir(); err != nil {
			log.Debug("no user config directory", "err", err)
			errs = append(errs, err)
		} else if config := try(filepath.Join(ucd, appDirName, configFileName)); config != nil {
			return config, nil
		}
		// Try loading the config file from next to the executable
		if executable, err := os.Executable(); err != nil {
			log.Debug("no executable path", "err", err)
			errs = append(errs, err)
		} else if config := try(filepath.Join(filepath.Dir(executable), configFileName)); config != nil {
			return config, nil
		}
	}
	// Joined rather than summarised, so that ErrLoad and every cause stay inspectable.
	return nil, errors.Join(errs...)
}
