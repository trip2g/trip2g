package logger

import "fmt"

// ConvertToFields converts key-value pairs into a map.
func ConvertToFields(kv []interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			key, ok := kv[i].(string)
			if !ok {
				key = fmt.Sprintf("%v (non-string key)", kv[i])
			}

			m[key] = kv[i+1]
		}
	}
	return m
}
