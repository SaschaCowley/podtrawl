package app

import (
	"errors"

	"github.com/spf13/pflag"
)

type flags struct {
	verbosity  int
	json       bool
	configPath *string
	dryRun     bool
}

func parseArgs(args []string) (*flags, error) {
	f := flags{}
	fs := pflag.NewFlagSet("podtrawl", pflag.ContinueOnError)
	// Configuration
	configPath := fs.StringP("config", "c", "", "path to configuration file")
	// Behaviour
	fs.BoolVarP(&f.dryRun, "dry-run", "n", false, "report what would be downloaded, without downloading it")
	// Logging
	verbose := fs.CountP("verbose", "v", "produce more output")
	quiet := fs.CountP("quiet", "q", "produce less output")
	fs.BoolVar(&f.json, "json", false, "output logs in JSON format")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.Changed("quiet") && fs.Changed("verbose") {
		return nil, errors.New("the quiet and verbose flags are mutually exclusive")
	} else if *verbose > 0 {
		f.verbosity = -*verbose
	} else if *quiet > 0 {
		f.verbosity = *quiet
	}
	if fs.Changed("config") {
		if *configPath == "" {
			return nil, errors.New("the config flag needs a path")
		}
		f.configPath = configPath
	}
	return &f, nil
}
