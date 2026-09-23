package feed_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"ssch.cc/podtrawl/internal/feed"
)

// Each case pairs a sample feed with the structure it must unmarshal into. The
// expected values live in a .go file named after the .xml file it describes.
var feedCases = []struct {
	path string
	want feed.Rss
}{
	{"rss-2.0-sample.xml", rss20Sample},
}

func TestFeeds(t *testing.T) {
	for _, tc := range feedCases {
		t.Run(tc.path, func(t *testing.T) {
			f, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("open %s: %v", tc.path, err)
			}
			defer f.Close()

			got, err := feed.Load(f)
			if err != nil {
				t.Fatalf("load %s: %v", tc.path, err)
			}
			diff(t, "Rss", reflect.ValueOf(*got), reflect.ValueOf(tc.want))
		})
	}
}

// diff walks got and want in parallel, reporting every difference it finds
// rather than stopping at the first. path names the position of the current
// values, so failures read like Rss.Channel.Items[1].PubDate.
//
// The walk is driven entirely by reflection, so fields added to the feed types
// are compared without touching this function.
func diff(t *testing.T, path string, got, want reflect.Value) {
	t.Helper()

	if !got.IsValid() || !want.IsValid() {
		if got.IsValid() != want.IsValid() {
			t.Errorf("%s: got valid=%t, want valid=%t", path, got.IsValid(), want.IsValid())
		}
		return
	}
	if got.Type() != want.Type() {
		t.Errorf("%s: got type %s, want type %s", path, got.Type(), want.Type())
		return
	}

	// A type that knows how to compare itself gets to. This is what lets
	// times compare by instant rather than by fabricated location, and it
	// keeps the walk out of unexported fields it could not read anyway.
	if equalMethod(want.Type()) {
		if !want.MethodByName("Equal").Call([]reflect.Value{got})[0].Bool() {
			t.Errorf("%s: got %v, want %v", path, got, want)
		}
		return
	}

	switch got.Kind() {
	case reflect.Pointer, reflect.Interface:
		if got.IsNil() || want.IsNil() {
			if got.IsNil() != want.IsNil() {
				t.Errorf("%s: got %s, want %s", path, nilness(got), nilness(want))
			}
			return
		}
		diff(t, path, got.Elem(), want.Elem())

	case reflect.Struct:
		for i := range got.NumField() {
			field := got.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			diff(t, path+"."+field.Name, got.Field(i), want.Field(i))
		}

	case reflect.Slice:
		if got.IsNil() != want.IsNil() {
			t.Errorf("%s: got %s, want %s", path, nilness(got), nilness(want))
			return
		}
		fallthrough

	case reflect.Array:
		if got.Len() != want.Len() {
			t.Errorf("%s: got length %d, want length %d", path, got.Len(), want.Len())
			return
		}
		for i := range got.Len() {
			diff(t, fmt.Sprintf("%s[%d]", path, i), got.Index(i), want.Index(i))
		}

	case reflect.Map:
		if got.IsNil() != want.IsNil() {
			t.Errorf("%s: got %s, want %s", path, nilness(got), nilness(want))
			return
		}
		if got.Len() != want.Len() {
			t.Errorf("%s: got %d keys, want %d keys", path, got.Len(), want.Len())
			return
		}
		for _, key := range want.MapKeys() {
			gotValue := got.MapIndex(key)
			if !gotValue.IsValid() {
				t.Errorf("%s[%v]: missing, want %v", path, key, want.MapIndex(key))
				continue
			}
			diff(t, fmt.Sprintf("%s[%v]", path, key), gotValue, want.MapIndex(key))
		}

	default:
		if !reflect.DeepEqual(got.Interface(), want.Interface()) {
			t.Errorf("%s: got %#v, want %#v", path, got.Interface(), want.Interface())
		}
	}
}

// equalMethod reports whether t has a method `Equal(t) bool` it can be
// compared with in place of a field-by-field walk.
func equalMethod(t reflect.Type) bool {
	m, ok := t.MethodByName("Equal")
	if !ok {
		return false
	}
	// MethodByName on a type includes the receiver as the first argument.
	return m.Type.NumIn() == 2 && t.AssignableTo(m.Type.In(1)) &&
		m.Type.NumOut() == 1 && m.Type.Out(0).Kind() == reflect.Bool
}

// nilness describes v for a report where one side is nil and the other is not.
// Pointers are dereferenced, since the address alone says nothing useful.
func nilness(v reflect.Value) string {
	if v.IsNil() {
		return "nil"
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		return fmt.Sprintf("%v", v.Elem())
	}
	return fmt.Sprintf("%v", v)
}
