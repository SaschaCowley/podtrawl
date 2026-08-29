package app

import (
	"fmt"

	"github.com/spf13/pflag"
)

type flags struct {
	verbosity int
	json      bool
}

func (f *flags) parse(args []string) error {
	fs := pflag.NewFlagSet("podtrawl", pflag.ContinueOnError)
	verbose := fs.CountP("verbose", "v", "Produce more output")
	quiet := fs.CountP("quiet", "q", "produce less output")
	fs.BoolVar(&f.json, "json", false, "output logs in JSON format")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.Changed("quiet") && fs.Changed("verbose") {
		return fmt.Errorf("The quiet and verbose flags are mutually exclusive")
	} else if *verbose > 0 {
		f.verbosity = -*verbose
	} else if *quiet > 0 {
		f.verbosity = *quiet
	}
	return nil
}
