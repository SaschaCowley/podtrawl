package feed

import (
	"reflect"
	"strings"
	"testing"
)

// checkNamespace asserts that every element field of T is bound to the name space ns.
// Attribute, chardata, innerxml and comment fields have no name space, and are skipped.
func checkNamespace[T any](t *testing.T, ns string) {
	t.Helper()
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Struct {
		t.Fatalf("checkNamespace: %s is a %s, want a struct", typ, typ.Kind())
	}
	for field := range typ.NumField() {
		f := typ.Field(field)
		if !f.IsExported() {
			continue
		}
		tag, ok := f.Tag.Lookup("xml")
		if !ok {
			t.Errorf("%s.%s has no xml tag, want one naming the %q name space", typ.Name(), f.Name, ns)
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if name == "-" || isTextOption(opts) {
			continue
		}
		got, _, found := strings.Cut(name, " ")
		if !found {
			t.Errorf("%s.%s is in no name space (xml:%q), want %q", typ.Name(), f.Name, tag, ns)
			continue
		}
		if got != ns {
			t.Errorf("%s.%s is in name space %q (xml:%q), want %q", typ.Name(), f.Name, got, tag, ns)
		}
	}
}

// isTextOption reports whether opts mark the field as referring to something other than an element, and thus having no name space	.
func isTextOption(opts string) bool {
	for opt := range strings.SplitSeq(opts, ",") {
		switch opt {
		case "attr", "chardata", "innerxml", "comment":
			return true
		}
	}
	return false
}
