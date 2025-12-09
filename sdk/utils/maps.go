package utils

// MergeConfig defines how specific fields (arrays of maps) should be merged.
// Key: the field name (e.g., "files"), Value: the key to match inside each map (e.g., "name").
type MergeConfig map[string]string

// MergeMaps merges two maps (map1 and map2) giving precedence to map2.
// If both maps contain the same key and the value is also a map, it merges recursively.
// If the value is a slice of maps and a MergeConfig is provided, it merges by key.
func MergeMaps(map1, map2 map[string]interface{}, cfg MergeConfig) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy map1 into result
	for k, v := range map1 {
		result[k] = v
	}

	// Merge map2 into result
	for k, v2 := range map2 {
		v1, exists := result[k]

		switch {
		// Both are nested maps → merge recursively
		case exists && isMap(v1) && isMap(v2):
			result[k] = MergeMaps(v1.(map[string]interface{}), v2.(map[string]interface{}), cfg)

		// Both are arrays of maps and merge config is defined
		case exists && isSlice(v1) && isSlice(v2) && cfg != nil:
			arr1 := v1.([]interface{})
			arr2 := v2.([]interface{})
			if mergeKey, ok := cfg[k]; ok && looksLikeArrayOfMaps(arr1) && looksLikeArrayOfMaps(arr2) {
				result[k] = mergeArrayOfMapsByKey(arr1, arr2, mergeKey, cfg)
			} else {
				result[k] = v2
			}

		// Default case → overwrite
		default:
			result[k] = v2
		}
	}

	return result
}

// mergeArrayOfMapsByKey merges two arrays of maps using a specified unique key (e.g., "name")
func mergeArrayOfMapsByKey(arr1, arr2 []interface{}, key string, cfg MergeConfig) []interface{} {
	index := make(map[interface{}]map[string]interface{})

	// Index elements of arr1 by key
	for _, item := range arr1 {
		if m, ok := item.(map[string]interface{}); ok {
			if id, exists := m[key]; exists {
				index[id] = m
			}
		}
	}

	// Merge elements from arr2 into the index
	for _, item := range arr2 {
		if m, ok := item.(map[string]interface{}); ok {
			if id, exists := m[key]; exists {
				if existing, found := index[id]; found {
					index[id] = MergeMaps(existing, m, cfg)
				} else {
					index[id] = m
				}
			}
		}
	}

	// Convert map back to slice
	result := make([]interface{}, 0, len(index))
	for _, m := range index {
		result = append(result, m)
	}
	return result
}

// looksLikeArrayOfMaps returns true if all elements are maps
func looksLikeArrayOfMaps(arr []interface{}) bool {
	for _, item := range arr {
		if _, ok := item.(map[string]interface{}); !ok {
			return false
		}
	}
	return true
}

// isMap checks if v is a map[string]interface{}
func isMap(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

// isSlice checks if v is a []interface{}
func isSlice(v interface{}) bool {
	_, ok := v.([]interface{})
	return ok
}
