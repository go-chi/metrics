package metrics

import (
	"reflect"
	"testing"
)

func TestSimpleStringLabels(t *testing.T) {
	type simpleLabels struct {
		Name   string `label:"name"`
		Status string `label:"status"`
		Type   string `label:"type"`
	}

	// Pre-warm the cache
	getLabelKeys[simpleLabels]()

	labels := simpleLabels{
		Name:   "test-service",
		Status: "active",
		Type:   "http",
	}

	result := getLabelValues(labels)

	expected := map[string]string{
		"name":   "test-service",
		"status": "active",
		"type":   "http",
	}

	for key, expectedValue := range expected {
		if result[key] != expectedValue {
			t.Errorf("Expected %s to be %q, got %q", key, expectedValue, result[key])
		}
	}

	// Test with empty strings
	emptyLabels := simpleLabels{}
	emptyResult := getLabelValues(emptyLabels)

	for key := range expected {
		if emptyResult[key] != "" {
			t.Errorf("Expected empty %s to be empty string, got %q", key, emptyResult[key])
		}
	}
}

func TestEmptyLabels(t *testing.T) {
	type emptyLabels struct{}

	// Empty struct should not panic - it just returns an empty slice of keys
	keys := getLabelKeys[emptyLabels]()

	if len(keys) != 0 {
		t.Errorf("Expected empty struct to return 0 keys, got %d", len(keys))
	}

	// Test labelValues with empty struct
	labels := emptyLabels{}
	result := getLabelValues(labels)

	if len(result) != 0 {
		t.Errorf("Expected empty struct to return 0 labels, got %d", len(result))
	}
}

func TestComplexLabels(t *testing.T) {
	type complexLabels struct {
		Name     string `label:"name"`
		Status   string `label:"status"`
		Duration string `label:"duration"`
		Count    string `label:"count"`
		Value    string `label:"value"`
		Active   string `label:"active"`
	}

	// Pre-warm the cache
	getLabelKeys[*complexLabels]()

	labels := &complexLabels{
		Name:     "complex-service",
		Status:   "active",
		Duration: "5s",
		Count:    "42",
		Value:    "3.14",
		Active:   "true",
	}

	result := getLabelValues(labels)

	t.Logf("Name (string): %q", result["name"])
	t.Logf("Status (string): %q", result["status"])
	t.Logf("Duration (string): %q", result["duration"])
	t.Logf("Count (string): %q", result["count"])
	t.Logf("Value (string): %q", result["value"])
	t.Logf("Active (string): %q", result["active"])

	if result["name"] != "complex-service" {
		t.Errorf("Expected name to be 'complex-service', got %q", result["name"])
	}
	if result["status"] != "active" {
		t.Errorf("Expected status to be 'active', got %q", result["status"])
	}
	if result["duration"] != "5s" {
		t.Errorf("Expected duration to be '5s', got %q", result["duration"])
	}
	if result["count"] != "42" {
		t.Errorf("Expected count to be '42', got %q", result["count"])
	}
	if result["value"] != "3.14" {
		t.Errorf("Expected value to be '3.14', got %q", result["value"])
	}
	if result["active"] != "true" {
		t.Errorf("Expected active to be 'true', got %q", result["active"])
	}
}

func TestNilReflectValue(t *testing.T) {
	// Test what happens when we call String() on a nil reflect.Value
	var v reflect.Value
	t.Logf("Nil reflect.Value.String(): %q", v.String())

	// Test what happens with a zero value
	var s string
	v = reflect.ValueOf(s)
	t.Logf("Zero string reflect.Value.String(): %q", v.String())

	var i int
	v = reflect.ValueOf(i)
	t.Logf("Zero int reflect.Value.String(): %q", v.String())

	var ptr *string
	v = reflect.ValueOf(ptr)
	t.Logf("Nil pointer reflect.Value.String(): %q", v.String())
}

func TestStructValidation(t *testing.T) {
	// Test that getLabelKeys panics for non-struct types
	testCases := []struct {
		name string
		fn   func()
	}{
		{
			name: "string type",
			fn:   func() { getLabelKeys[string]() },
		},
		{
			name: "int type",
			fn:   func() { getLabelKeys[int]() },
		},
		{
			name: "slice type",
			fn:   func() { getLabelKeys[[]string]() },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for non-struct type, but none occurred")
				} else {
					t.Logf("Expected panic occurred: %v", r)
				}
			}()
			tc.fn()
		})
	}
}

func TestUnsupportedFieldTypes(t *testing.T) {
	testCases := []struct {
		name string
		fn   func()
	}{
		{
			name: "pointer field",
			fn: func() {
				type testStruct struct {
					Name string `label:"name"`
					Ptr  *int   `label:"ptr"`
				}
				getLabelKeys[testStruct]()
			},
		},
		{
			name: "slice field",
			fn: func() {
				type testStruct struct {
					Name  string   `label:"name"`
					Items []string `label:"items"`
				}
				getLabelKeys[testStruct]()
			},
		},
		{
			name: "map field",
			fn: func() {
				type testStruct struct {
					Name string            `label:"name"`
					Data map[string]string `label:"data"`
				}
				getLabelKeys[testStruct]()
			},
		},
		{
			name: "array field",
			fn: func() {
				type testStruct struct {
					Name string    `label:"name"`
					Data [3]string `label:"data"`
				}
				getLabelKeys[testStruct]()
			},
		},
		{
			name: "channel field",
			fn: func() {
				type testStruct struct {
					Name string      `label:"name"`
					Chan chan string `label:"chan"`
				}
				getLabelKeys[testStruct]()
			},
		},
		{
			name: "interface field",
			fn: func() {
				type testStruct struct {
					Name string      `label:"name"`
					Data interface{} `label:"data"`
				}
				getLabelKeys[testStruct]()
			},
		},
		{
			name: "unexported field",
			fn: func() {
				type testStruct struct {
					Name string `label:"name"`
					data string `label:"data"` // unexported field
				}
				getLabelKeys[testStruct]()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for unsupported field type, but none occurred")
				} else {
					t.Logf("Expected panic occurred: %v", r)
				}
			}()
			tc.fn()
		})
	}
}
