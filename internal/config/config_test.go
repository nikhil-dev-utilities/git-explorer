package config

import (
	"reflect"
	"strings"
	"testing"
)

// TestConfig_HasNoCredentialShapedField is a structural regression guard for ADR-0004:
// Config must have no field, at any nesting depth, whose name suggests it could hold a
// credential. This is deliberately not a one-time review — it walks the type with
// reflection so a future field addition (e.g. a "token" convenience flag) fails the
// build rather than being missed in review.
func TestConfig_HasNoCredentialShapedField(t *testing.T) {
	suspicious := []string{"token", "secret", "password", "passwd", "apikey", "api_key", "credential"}

	var walk func(rt reflect.Type, path string)
	walk = func(rt reflect.Type, path string) {
		switch rt.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array:
			walk(rt.Elem(), path)
			return
		case reflect.Struct:
			// fall through to field walk below
		default:
			return
		}
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			fieldPath := path + "." + f.Name
			lower := strings.ToLower(f.Name)
			for _, s := range suspicious {
				if strings.Contains(lower, s) {
					t.Fatalf("Config field %s looks credential-shaped (matches %q) — ADR-0004 requires Config to structurally have no field capable of holding a secret", fieldPath, s)
				}
			}
			walk(f.Type, fieldPath)
		}
	}

	walk(reflect.TypeOf(Config{}), "Config")
}
