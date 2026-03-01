package triggers

import "fmt"

// getString extracts a required string param.
func getString(params map[string]interface{}, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", fmt.Errorf("missing required param %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("param %q must be a string, got %T", key, v)
	}
	return s, nil
}

// getStringOptional extracts an optional string param.
func getStringOptional(params map[string]interface{}, key string) string {
	v, ok := params[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// getInt extracts a required integer param.
// Handles JSON's float64 and BSON's int32/int64 representations.
func getInt(params map[string]interface{}, key string) (int, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("missing required param %q", key)
	}
	return toInt(v, key)
}

// getStringSlice extracts a required []string param.
func getStringSlice(params map[string]interface{}, key string) ([]string, error) {
	v, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("missing required param %q", key)
	}
	raw, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("param %q must be an array, got %T", key, v)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("param %q must be a non-empty array", key)
	}
	result := make([]string, 0, len(raw))
	for i, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("param %q[%d] must be a string, got %T", key, i, item)
		}
		result = append(result, s)
	}
	return result, nil
}

// toInt converts various numeric types to int.
func toInt(v interface{}, key string) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case float32:
		return int(n), nil
	case int:
		return n, nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("param %q must be a number, got %T", key, v)
	}
}
