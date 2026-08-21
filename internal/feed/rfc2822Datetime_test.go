package feed

import (
	"testing"
	"time"
)

func TestRfc2822DatetimeFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			"numeric zone with weekday",
			"Tue, 2 Jan 2024 15:04:05 -0700",
			time.Date(2024, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600)),
		},
		{
			"named zone with weekday",
			"Tue, 2 Jan 2024 15:04:05 UTC",
			time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			"numeric zone without weekday",
			"2 Jan 2024 15:04:05 -0700",
			time.Date(2024, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600)),
		},
		{
			"named zone without weekday",
			"2 Jan 2024 15:04:05 UTC",
			time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			"surrounding whitespace is trimmed",
			"  Tue, 2 Jan 2024 15:04:05 UTC  ",
			time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Rfc2822Datetime
			if err := got.UnmarshalText([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v", tt.input, err)
			}
			if !got.Time.Equal(tt.want) {
				t.Errorf("UnmarshalText(%q) = %v, want %v", tt.input, got.Time, tt.want)
			}
		})
	}
}

func TestRfc2822DatetimeInvalid(t *testing.T) {
	tests := []string{
		"",
		"not a date",
		"2024-01-02T15:04:05Z",
		"Tue, 2 Jan 2024",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			var got Rfc2822Datetime
			if err := got.UnmarshalText([]byte(input)); err == nil {
				t.Errorf("UnmarshalText(%q) expected error, got nil", input)
			}
		})
	}
}
