package workflow

import "strings"

func parseNamespace(namespace string) (string, []string, bool) {
	trimmed := strings.TrimSpace(namespace)
	if !strings.HasPrefix(trimmed, "$.steps.") {
		return "", nil, false
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) < 3 {
		return "", nil, false
	}
	return parts[2], parts[3:], true
}

func validateWorkflowNamespace(path []string) error {
	if !containsPathToken(path, "execute") || !containsPathToken(path, "inputs") {
		return NamespaceResolutionError{
			Message: "unsupported sub-workflow namespace path",
			Hint: "use execute inputs like " +
				"$.steps.<step>.execute.inputs.items.item",
		}
	}
	return nil
}

func validatePluginNamespace(path []string) error {
	if !containsPathToken(path, "inputs") {
		return NamespaceResolutionError{
			Message: "unsupported plugin namespace path",
			Hint: "use starting inputs like " +
				"$.steps.<step>.starting.inputs.input",
		}
	}
	return nil
}

func containsPathToken(path []string, token string) bool {
	for _, part := range path {
		if part == token {
			return true
		}
	}
	return false
}
