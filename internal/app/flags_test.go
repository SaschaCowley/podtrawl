package app

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

// defaultFlags is what parseArgs returns for an empty argument list.
// Provided in case defaults change from their zero values.
func defaultFlags() flags {
	return flags{}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		change  func(*flags) // nil when the arguments change nothing
		wantErr bool
	}{
		{name: "no args", args: []string{}},
		{name: "verbose is subtractive", args: []string{"-v", "--verbose"}, change: func(f *flags) { f.verbosity = -2 }},
		{name: "quiet is additive", args: []string{"--quiet", "-q"}, change: func(f *flags) { f.verbosity = 2 }},
		{name: "verbose and quiet are mutually exclusive", args: []string{"-v", "-q"}, wantErr: true},
		{name: "config path is passed along", args: []string{"-c", "path/to/config.toml"}, change: func(f *flags) { f.configPath = new("path/to/config.toml") }},
		{name: "config path may not be empty", args: []string{"--config", ""}, wantErr: true},
		{name: "json is passed along", args: []string{"--json"}, change: func(f *flags) { f.json = true }},
		{name: "dry run is passed along", args: []string{"-n"}, change: func(f *flags) { f.dryRun = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseArgs(%q) = %s, nil, want an error", tt.args, fmtFlags(got))
				}
				if got != nil {
					t.Errorf("parseArgs(%q) = %s, _, alongside an error, want nil", tt.args, fmtFlags(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%q) = _, %v, want nil", tt.args, err)
			}
			if got == nil {
				t.Fatalf("parseArgs(%q) = nil, nil, want flags", tt.args)
			}
			want := defaultFlags()
			if tt.change != nil {
				tt.change(&want)
			}
			if !reflect.DeepEqual(*got, want) {
				t.Errorf("parseArgs(%q) = %s, want %s", tt.args, fmtFlags(got), fmtFlags(&want))
			}
		})
	}
}

// fmtFlags renders flags for a failure message.
func fmtFlags(f *flags) string {
	if f == nil {
		return "nil"
	}
	configPath := "nil"
	if f.configPath != nil {
		configPath = strconv.Quote(*f.configPath)
	}
	return fmt.Sprintf("{configPath:%s, dryRun:%t, json:%t, verbosity:%d}",
		configPath, f.dryRun, f.json, f.verbosity)
}
