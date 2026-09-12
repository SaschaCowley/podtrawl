package app

import (
	"errors"

	"github.com/spf13/pflag"
	"ssch.cc/podtrawl/internal/naming"
)

type flags struct {
	verbosity  int
	json       bool
	configPath *string
	dryRun     bool
	noCache    bool
	redownload bool
	existing   *naming.Existing
}

func parseArgs(args []string) (*flags, error) {
	f := flags{}
	fs := pflag.NewFlagSet("podtrawl", pflag.ContinueOnError)
	// Configuration
	configPath := fs.StringP("config", "c", "", "path to configuration file")
	var existing naming.Existing
	fs.TextVar(&existing, "existing", naming.Increment, "policy for file conflicts")
	// Behaviour
	fs.BoolVarP(&f.dryRun, "dry-run", "n", false, "report what would be downloaded, without downloading it")
	fs.BoolVar(&f.noCache, "no-cache", false, "do not read or write the cache")
	fs.BoolVar(&f.redownload, "redownload", false, "redownload episodes that have previously been downloaded")
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
	if fs.Changed("no-cache") && fs.Changed("redownload") {
		return nil, errors.New("the no-cache and redownload flags are mutually exclusive")
	}
	if fs.Changed("existing") {
		f.existing = &existing
	}
	return &f, nil
}
