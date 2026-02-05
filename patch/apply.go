package patch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

func ApplyRFC6902[T any](current T, ops []Operation) (T, error) {
	var zero T

	if len(ops) == 0 {
		return current, nil
	}

	currentJSON, err := json.Marshal(current)
	if err != nil {
		return zero, fmt.Errorf("failed to marshal current state: %w", err)
	}

	ops = FixOperation(currentJSON, ops)

	patchJSON, err := json.Marshal(ops)
	if err != nil {
		return zero, fmt.Errorf("failed to marshal patch operations: %w", err)
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return zero, fmt.Errorf("failed to decode patch: %w", err)
	}

	modifiedJSON, err := patch.Apply(currentJSON)
	if err != nil {
		return zero, fmt.Errorf("failed to apply patch: %w", err)
	}

	var result T
	if err := json.Unmarshal(modifiedJSON, &result); err != nil {
		return zero, fmt.Errorf("type mismatch: patch would result in invalid type T: %w", err)
	}

	return result, nil
}

func FixOperation(currentJSON []byte, ops []Operation) []Operation {
	var doc any
	if err := json.Unmarshal(currentJSON, &doc); err != nil {
		return ops
	}

	fixed := make([]Operation, 0, len(ops))
	for _, op := range ops {
		switch op.Op {
		case OperationReplace:
			if !pathExists(doc, op.Path) {
				op.Op = OperationAdd
			}
			fixed = append(fixed, op)
		case OperationRemove:
			if pathExists(doc, op.Path) {
				fixed = append(fixed, op)
			}
		default:
			fixed = append(fixed, op)
		}
	}

	return fixed
}

func pathExists(doc any, path string) bool {
	if path == "" {
		return true
	}
	if !strings.HasPrefix(path, "/") {
		return false
	}

	tokens := strings.Split(path[1:], "/")
	cur := doc
	for _, token := range tokens {
		token = strings.ReplaceAll(token, "~1", "/")
		token = strings.ReplaceAll(token, "~0", "~")
		switch node := cur.(type) {
		case map[string]any:
			value, ok := node[token]
			if !ok {
				return false
			}
			cur = value
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(node) {
				return false
			}
			cur = node[index]
		default:
			return false
		}
	}

	return true
}
