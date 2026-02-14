package triggers

import (
	"testing"
)

func TestGetString(t *testing.T) {
	params := map[string]interface{}{"name": "fire", "count": 42}

	s, err := getString(params, "name")
	if err != nil || s != "fire" {
		t.Errorf("expected 'fire', got %q, err=%v", s, err)
	}

	_, err = getString(params, "missing")
	if err == nil {
		t.Error("expected error for missing key")
	}

	_, err = getString(params, "count")
	if err == nil {
		t.Error("expected error for non-string value")
	}
}

func TestGetStringOptional(t *testing.T) {
	params := map[string]interface{}{"name": "fire"}

	if got := getStringOptional(params, "name"); got != "fire" {
		t.Errorf("expected 'fire', got %q", got)
	}
	if got := getStringOptional(params, "missing"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGetInt_Float64(t *testing.T) {
	params := map[string]interface{}{"amount": float64(42)}
	n, err := getInt(params, "amount")
	if err != nil || n != 42 {
		t.Errorf("expected 42, got %d, err=%v", n, err)
	}
}

func TestGetInt_Int32(t *testing.T) {
	params := map[string]interface{}{"amount": int32(7)}
	n, err := getInt(params, "amount")
	if err != nil || n != 7 {
		t.Errorf("expected 7, got %d, err=%v", n, err)
	}
}

func TestGetInt_Int64(t *testing.T) {
	params := map[string]interface{}{"amount": int64(100)}
	n, err := getInt(params, "amount")
	if err != nil || n != 100 {
		t.Errorf("expected 100, got %d, err=%v", n, err)
	}
}

func TestGetInt_Missing(t *testing.T) {
	params := map[string]interface{}{}
	_, err := getInt(params, "amount")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestGetInt_WrongType(t *testing.T) {
	params := map[string]interface{}{"amount": "not a number"}
	_, err := getInt(params, "amount")
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestGetStringSlice(t *testing.T) {
	params := map[string]interface{}{
		"conditions": []interface{}{"poisoned", "blinded"},
	}

	s, err := getStringSlice(params, "conditions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 2 || s[0] != "poisoned" || s[1] != "blinded" {
		t.Errorf("expected [poisoned blinded], got %v", s)
	}
}

func TestGetStringSlice_Missing(t *testing.T) {
	params := map[string]interface{}{}
	_, err := getStringSlice(params, "conditions")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestGetStringSlice_Empty(t *testing.T) {
	params := map[string]interface{}{
		"conditions": []interface{}{},
	}
	_, err := getStringSlice(params, "conditions")
	if err == nil {
		t.Error("expected error for empty array")
	}
}

func TestGetStringSlice_WrongElementType(t *testing.T) {
	params := map[string]interface{}{
		"conditions": []interface{}{"poisoned", 42},
	}
	_, err := getStringSlice(params, "conditions")
	if err == nil {
		t.Error("expected error for non-string element")
	}
}

func TestGetStringSlice_NotArray(t *testing.T) {
	params := map[string]interface{}{
		"conditions": "poisoned",
	}
	_, err := getStringSlice(params, "conditions")
	if err == nil {
		t.Error("expected error for non-array value")
	}
}
