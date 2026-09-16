package ledger

import (
	"encoding/json/v2"
	"maps"
	"slices"
	"testing"
)

func TestSetMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		set  strSet
		want string
	}{
		{"nil", nil, `[]`},
		{"empty", strSet{}, `[]`},
		{"one member", strSet{"guid1": {}}, `["guid1"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.set)
			if err != nil {
				t.Fatalf("json.Marshal(%#v) = %v", tt.set, err)
			}
			if string(got) != tt.want {
				t.Errorf("json.Marshal(%#v) = %s, want %s", tt.set, got, tt.want)
			}
		})
	}
}

func TestSetUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    strSet
		wantErr bool
	}{
		{"empty array", `[]`, strSet{}, false},
		{"null", `null`, strSet{}, false},
		{"members", `["guid1","guid2"]`, strSet{"guid1": {}, "guid2": {}}, false},
		// A guid listed twice says nothing the first mention didn't.
		{"duplicates collapse", `["guid1","guid1"]`, strSet{"guid1": {}}, false},
		{"not an array", `{"guid1":{}}`, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got strSet
			err := json.Unmarshal([]byte(tt.json), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("json.Unmarshal(%s) = nil, want an error", tt.json)
				}
				return
			}
			if err != nil {
				t.Fatalf("json.Unmarshal(%s) = %v", tt.json, err)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("json.Unmarshal(%s) = %#v, want %#v", tt.json, got, tt.want)
			}
		})
	}
}

// TestSetOmitEmpty checks that a feed with nothing downloaded writes no
// downloaded member at all, rather than an empty array.
func TestSetOmitEmpty(t *testing.T) {
	for _, tt := range []struct {
		name string
		feed feed
	}{
		{"nil set", feed{}},
		{"empty set", feed{Downloaded: strSet{}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.feed)
			if err != nil {
				t.Fatalf("json.Marshal(%#v) = %v", tt.feed, err)
			}
			if string(got) != `{}` {
				t.Errorf("json.Marshal(%#v) = %s, want {}", tt.feed, got)
			}
		})
	}
}

func TestSetRoundTrip(t *testing.T) {
	want := strSet{"guid1": {}, "guid2": {}, "guid3": {}, "guid4": {}}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal(%#v) = %v", want, err)
	}
	// Check that marshaling didn't emit anything unexpected
	var members []string
	if err := json.Unmarshal(data, &members); err != nil {
		t.Fatalf("json.Unmarshal(%s) as an array = %v", data, err)
	}
	if len(members) != len(want) {
		t.Errorf("json.Marshal(%#v) = %s, want %d members", want, data, len(want))
	}
	slices.Sort(members)
	if !slices.Equal(members, slices.Sorted(maps.Keys(want))) {
		t.Errorf("json.Marshal(%#v) = %s, want the same members", want, data)
	}
	var got strSet
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v", data, err)
	}
	if !maps.Equal(got, want) {
		t.Errorf("round trip of %#v through %s = %#v", want, data, got)
	}
}
