package patch

import (
	"fmt"
	"strings"
)

func ValidatePatchOperations(ops []Operation, allowedPaths map[string]bool) error {
	if len(ops) == 0 {
		return nil
	}
	for i, op := range ops {
		if err := validatePathAllowed(op.Path, allowedPaths); err != nil {
			return fmt.Errorf("operation %d: %w", i, err)
		}
	}
	return nil
}

func validatePathAllowed(path string, allowedPaths map[string]bool) error {
	if len(allowedPaths) == 0 {
		return nil
	}
	if allowedPaths[path] {
		return nil
	}
	if isPathMatchedByWildcard(path, allowedPaths) {
		return nil
	}
	return fmt.Errorf("path %q is not in the allowed paths set", path)
}

func isPathMatchedByWildcard(path string, allowedPaths map[string]bool) bool {
	segments := strings.Split(path, "/")
	return matchWildcardRecursive(segments, 0, allowedPaths, false)
}

func matchWildcardRecursive(segments []string, index int, allowedPaths map[string]bool, hasWildcard bool) bool {
	if index >= len(segments) {
		if !hasWildcard {
			return false
		}
		pattern := strings.Join(segments, "/")
		return allowedPaths[pattern]
	}

	if index == 0 {
		return matchWildcardRecursive(segments, index+1, allowedPaths, hasWildcard)
	}

	original := segments[index]

	segments[index] = "-"
	if matchWildcardRecursive(segments, index+1, allowedPaths, true) {
		segments[index] = original
		return true
	}

	segments[index] = "*"
	if matchWildcardRecursive(segments, index+1, allowedPaths, true) {
		segments[index] = original
		return true
	}

	segments[index] = original
	return matchWildcardRecursive(segments, index+1, allowedPaths, hasWildcard)
}
