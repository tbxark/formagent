package formagent

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

func ApplyRFC6902[T any](current T, ops []PatchOperation, allowedPaths map[string]bool) (T, error) {
	var zero T

	if err := ValidatePatchOperations(ops, allowedPaths); err != nil {
		return zero, fmt.Errorf("patch validation failed: %w", err)
	}
	if len(ops) == 0 {
		return current, nil
	}

	currentJSON, err := json.Marshal(current)
	if err != nil {
		return zero, fmt.Errorf("failed to marshal current state: %w", err)
	}

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
