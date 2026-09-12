package cache

import (
	"encoding/json/v2"
	"maps"
	"slices"
)

// strSet is a Set of strings that reads and writes as a JSON array.
type strSet map[string]struct{}

// MarshalJSON writes the set as an array.
func (s strSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(slices.Collect(maps.Keys(s)))
}

// UnmarshalJSON reads an array into the set, discarding any duplicates.
func (s *strSet) UnmarshalJSON(data []byte) error {
	var members []string
	if err := json.Unmarshal(data, &members); err != nil {
		return err
	}
	*s = make(strSet, len(members))
	for _, m := range members {
		(*s)[m] = struct{}{}
	}
	return nil
}
