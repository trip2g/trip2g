package frontmatteredit

import (
	"fmt"
	"maps"
	"slices"

	yamlv2 "gopkg.in/yaml.v2"
)

// verify reads the edited note back the way the edit read the original, and the
// way goldmark-meta reads it for trip2g, and refuses any result where a key
// other than the named ones changed, a named one did not get its value, or the
// body moved.
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
		node, encErr := valueNode(value)
		if encErr != nil {
			return encErr
		}
		want[key], err = decodeValue(node)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUnsafeEdit, err)
		}
	}

	got, err := readValues(now.yaml())
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsafeEdit, err)
	}
	err = compareValues(want, got)
	if err != nil {
		return err
	}

	return compareGoldmarkKeys(now.yaml(), got)
}

func readValues(text string) (map[string]string, error) {
	root, err := parseRoot(text)
	if err != nil || root == nil {
		return map[string]string{}, err
	}
	values := make(map[string]string, len(root.Content)/2)
	for i := 0; i+1 < len(root.Content); i += 2 {
		name := root.Content[i].Value
		_, dup := values[name]
		if dup {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateKey, name)
		}
		values[name], err = decodeValue(root.Content[i+1])
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func compareValues(want, got map[string]string) error {
	keys := slices.Sorted(maps.Keys(want))
	for _, key := range keys {
		value, ok := got[key]
		if !ok {
			return fmt.Errorf("%w: key %s would be lost", ErrUnsafeEdit, key)
		}
		if value != want[key] {
			return fmt.Errorf("%w: key %s would read differently", ErrUnsafeEdit, key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(got)) {
		_, ok := want[key]
		if !ok {
			return fmt.Errorf("%w: key %s would appear", ErrUnsafeEdit, key)
		}
	}
	return nil
}

// compareGoldmarkKeys checks the edit with gopkg.in/yaml.v2 into a map, which is
// what goldmark-meta does when trip2g reads a note's meta.
func compareGoldmarkKeys(text string, want map[string]string) error {
	meta := map[string]any{}
	err := yamlv2.Unmarshal([]byte(text), &meta)
	if err != nil {
		return fmt.Errorf("%w: goldmark-meta could not read it: %w", ErrUnsafeEdit, err)
	}
	got := slices.Sorted(maps.Keys(meta))
	if !slices.Equal(got, slices.Sorted(maps.Keys(want))) {
		return fmt.Errorf("%w: goldmark-meta would read keys %v", ErrUnsafeEdit, got)
	}
	return nil
}
