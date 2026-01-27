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
		if err := validateOp(op.Op); err != nil {
			return fmt.Errorf("operation %d: %w", i, err)
		}

		if err := validateJSONPointer(op.Path); err != nil {
			return fmt.Errorf("operation %d: %w", i, err)
		}

		if err := validatePathAllowed(op.Path, allowedPaths); err != nil {
			return fmt.Errorf("operation %d: %w", i, err)
		}

		if (op.Op == "add" || op.Op == "replace") && op.Value == nil {
			return fmt.Errorf("operation %d: %s operation requires a value", i, op.Op)
		}
	}

	return nil
}

func validateOp(op string) error {
	switch op {
	case "add", "replace", "remove":
		return nil
	default:
		return fmt.Errorf("invalid operation type %q: must be one of 'add', 'replace', or 'remove'", op)
	}
}

func validateJSONPointer(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty: must start with '/'")
	}
	if path[0] != '/' {
		return fmt.Errorf("invalid JSON Pointer %q: must start with '/'", path)
	}

	tokens := strings.Split(path[1:], "/")
	for i, token := range tokens {
		if strings.Contains(token, "~") {
			if err := validateEscapeSequences(token); err != nil {
				return fmt.Errorf("invalid JSON Pointer %q at token %d: %w", path, i, err)
			}
		}
	}

	return nil
}

func validateEscapeSequences(token string) error {
	i := 0
	for i < len(token) {
		if token[i] == '~' {
			if i+1 >= len(token) {
				return fmt.Errorf("invalid escape sequence: '~' at end of token")
			}

			next := token[i+1]
			if next != '0' && next != '1' {
				return fmt.Errorf("invalid escape sequence: '~%c' (must be '~0' or '~1')", next)
			}
			i += 2
			continue
		}
		i++
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
