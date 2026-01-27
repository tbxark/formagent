package patch

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func GeneratePatchesFromInitial[T any](current, initial T) ([]Operation, error) {
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal current state: %w", err)
	}

	initialJSON, err := json.Marshal(initial)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initial state: %w", err)
	}

	var currentMap map[string]interface{}
	if err := json.Unmarshal(currentJSON, &currentMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal current state: %w", err)
	}

	var initialMap map[string]interface{}
	if err := json.Unmarshal(initialJSON, &initialMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal initial state: %w", err)
	}

	patches := make([]Operation, 0)
	generatePatchesFromMap("", currentMap, initialMap, &patches)
	return patches, nil
}

func generatePatchesFromMap(prefix string, current, initial map[string]interface{}, patches *[]Operation) {
	for key, initialValue := range initial {
		if initialValue == nil {
			continue
		}

		path := prefix + "/" + escapeJSONPointer(key)
		currentValue, existsInCurrent := current[key]

		if isZeroValue(initialValue) {
			continue
		}

		if initialMap, ok := initialValue.(map[string]interface{}); ok {
			if currentMap, ok := currentValue.(map[string]interface{}); ok {
				generatePatchesFromMap(path, currentMap, initialMap, patches)
			} else {
				*patches = append(*patches, Operation{Op: "replace", Path: path, Value: initialValue})
			}
			continue
		}

		if initialArray, ok := initialValue.([]interface{}); ok {
			if !existsInCurrent || !reflect.DeepEqual(currentValue, initialValue) {
				if len(initialArray) > 0 {
					*patches = append(*patches, Operation{Op: "replace", Path: path, Value: initialValue})
				}
			}
			continue
		}

		if !existsInCurrent {
			*patches = append(*patches, Operation{Op: "add", Path: path, Value: initialValue})
		} else if !reflect.DeepEqual(currentValue, initialValue) {
			*patches = append(*patches, Operation{Op: "replace", Path: path, Value: initialValue})
		}
	}
}

func escapeJSONPointer(token string) string {
	result := ""
	for _, ch := range token {
		switch ch {
		case '~':
			result += "~0"
		case '/':
			result += "~1"
		default:
			result += string(ch)
		}
	}
	return result
}

func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case float64:
		return val == 0
	case bool:
		return !val
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	default:
		return false
	}
}
