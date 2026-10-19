package frontmatteredit

import (
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"

	yamlv2 "gopkg.in/yaml.v2"
)

// verify reads the edited note back the way the edit read the original, and the
// way goldmark-meta reads it for trip2g, and refuses any result where a key
// other than the named ones changed, a named one did not get its value, or the
// body moved. A set key's value is checked against the value asked for, not
// against what the encoder made of it, so an encoder quirk is a refusal.
func verify(before, after string, set map[string]any, unset []string) error {
	was, err := split(before)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsafeEdit, err)
	}
	now, err := split(after)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsafeEdit, err)
	}
	if now.body != was.body {
		return fmt.Errorf("%w: the note body would change", ErrUnsafeEdit)
	}

	want, err := readValues(was.yaml())
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsafeEdit, err)
	}
	for _, key := range unset {
		delete(want, key)
	}
	for key, value := range set {
		want[key] = normalize(value)
	}

	got, err := readValues(now.yaml())
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsafeEdit, err)
	}
	err = compareValues(want, got)
	if err != nil {
		return err
	}

	return compareGoldmark(was.yaml(), now.yaml(), slices.Sorted(maps.Keys(got)), set)
}

func readValues(text string) (map[string]any, error) {
	root, err := parseRoot(text)
	if err != nil || root == nil {
		return map[string]any{}, err
	}
	values := make(map[string]any, len(root.Content)/2)
	for i := 0; i+1 < len(root.Content); i += 2 {
		name := root.Content[i].Value
		_, dup := values[name]
		if dup {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateKey, name)
		}
		var value any
		err = root.Content[i+1].Decode(&value)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidYAML, err)
		}
		values[name] = value
	}
	return values, nil
}

func compareValues(want, got map[string]any) error {
	for _, key := range slices.Sorted(maps.Keys(want)) {
		value, ok := got[key]
		if !ok {
			return fmt.Errorf("%w: key %q would be lost", ErrUnsafeEdit, key)
		}
		if !sameValue(want[key], value) {
			return fmt.Errorf("%w: key %q would read differently", ErrUnsafeEdit, key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(got)) {
		_, ok := want[key]
		if !ok {
			return fmt.Errorf("%w: key %q would appear", ErrUnsafeEdit, key)
		}
	}
	return nil
}

// compareGoldmark reads both notes with gopkg.in/yaml.v2 into a map, which is
// what goldmark-meta does when trip2g reads a note's meta: it must see the keys
// the edit left, each set key must hold the value asked for, and each other key
// must read as it did before the edit.
func compareGoldmark(before, after string, keys []string, set map[string]any) error {
	now := map[string]any{}
	err := yamlv2.Unmarshal([]byte(after), &now)
	if err != nil {
		return fmt.Errorf("%w: goldmark-meta could not read it: %w", ErrUnsafeEdit, err)
	}
	got := slices.Sorted(maps.Keys(now))
	if !slices.Equal(got, keys) {
		return fmt.Errorf("%w: goldmark-meta would read keys %q", ErrUnsafeEdit, got)
	}

	was := map[string]any{}
	err = yamlv2.Unmarshal([]byte(before), &was)
	if err != nil {
		return fmt.Errorf("%w: goldmark-meta could not read the note before the edit: %w", ErrUnsafeEdit, err)
	}
	for _, key := range keys {
		want, isSet := set[key]
		if isSet {
			want = normalize(want)
		} else {
			want, isSet = was[key]
		}
		if !isSet || !sameValue(want, now[key]) {
			return fmt.Errorf("%w: goldmark-meta would read key %q differently", ErrUnsafeEdit, key)
		}
	}
	return nil
}

// sameValue reports whether a reader sees a and b as one value: numbers compare
// by what they are worth whatever their Go type, NaN equals NaN, and maps
// compare by key whatever their key type.
func sameValue(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	x, y := reflect.ValueOf(a), reflect.ValueOf(b)
	if isNumber(x) && isNumber(y) {
		return sameNumber(x, y)
	}
	switch {
	case x.Kind() == reflect.String && y.Kind() == reflect.String:
		return x.String() == y.String()
	case isList(x) && isList(y):
		if x.Len() != y.Len() {
			return false
		}
		for i := range x.Len() {
			if !sameValue(x.Index(i).Interface(), y.Index(i).Interface()) {
				return false
			}
		}
		return true
	case x.Kind() == reflect.Map && y.Kind() == reflect.Map:
		return sameMap(x, y)
	default:
		return reflect.DeepEqual(a, b)
	}
}

func isList(v reflect.Value) bool {
	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

func sameMap(x, y reflect.Value) bool {
	if x.Len() != y.Len() {
		return false
	}
	other := make(map[string]any, y.Len())
	for _, key := range y.MapKeys() {
		other[fmt.Sprint(key.Interface())] = y.MapIndex(key).Interface()
	}
	if len(other) != y.Len() {
		return false
	}
	for _, key := range x.MapKeys() {
		value, ok := other[fmt.Sprint(key.Interface())]
		if !ok || !sameValue(x.MapIndex(key).Interface(), value) {
			return false
		}
	}
	return true
}

func isNumber(v reflect.Value) bool {
	return v.CanInt() || v.CanUint() || v.CanFloat()
}

func sameNumber(x, y reflect.Value) bool {
	if x.CanFloat() || y.CanFloat() {
		a, b := toFloat(x), toFloat(y)
		return a == b || (math.IsNaN(a) && math.IsNaN(b))
	}
	if x.CanInt() && y.CanInt() {
		return x.Int() == y.Int()
	}
	if x.CanUint() && y.CanUint() {
		return x.Uint() == y.Uint()
	}
	if x.CanUint() {
		x, y = y, x
	}
	return x.Int() >= 0 && uint64(x.Int()) == y.Uint()
}

func toFloat(v reflect.Value) float64 {
	switch {
	case v.CanFloat():
		return v.Float()
	case v.CanInt():
		return float64(v.Int())
	default:
		return float64(v.Uint())
	}
}
