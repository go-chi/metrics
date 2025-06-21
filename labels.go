package metrics

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	labelCache sync.Map // map[reflect.Type][]string
)

// isValidLabelName checks if a label name matches the Prometheus format [a-z_][a-z0-9_]*.
func isValidLabelName(s string) bool {
	if len(s) == 0 {
		return false
	}

	// First character must be [a-z_]
	if s[0] != '_' && (s[0] < 'a' || s[0] > 'z') {
		return false
	}

	// Remaining characters must be [a-z0-9_]
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c != '_' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}

	return true
}

// mustValidMetricName validates a metric name and panics if it doesn't match the required Prometheus format.
func mustValidMetricName(s string) string {
	if !isValidLabelName(s) {
		panic("invalid metric name: " + s + " (must match [a-z_][a-z0-9_]*)")
	}
	return s
}

// getLabelKeys returns the list of labels defined as struct tags, e.g. `label:"some_name"`,
// and panics if any of the labels are empty or invalid.
// This function implements memoization - it computes the keys once per type and caches them.
func getLabelKeys[T any]() []string {
	var zero T
	t := reflect.TypeOf(zero)

	cached, ok := labelCache.Load(t)
	if ok {
		return cached.([]string)
	}

	structType := t
	if t.Kind() == reflect.Ptr {
		structType = t.Elem()
	}

	if structType.Kind() != reflect.Struct {
		panic("invalid label type: " + structType.Kind().String() + " (must be struct)")
	}

	var keys []string
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		labelName := field.Tag.Get("label")
		if labelName == "" {
			panic(fmt.Sprintf("missing `label` struct tag (package %v):\ntype %s %s {\n\t%s %s `label:\"add_label_here\"`\n}", structType.PkgPath(), structType.Name(), structType.Kind(), field.Name, field.Type))
		}
		if !isValidLabelName(labelName) {
			panic(fmt.Sprintf("invalid `label` name (package %v):\ntype %s %s {\n\t%s %s `label:\"%s\"` // <-- label must match [a-z_][a-z0-9_]*\n}", structType.PkgPath(), structType.Name(), structType.Kind(), field.Name, field.Type, labelName))
		}
		if !field.IsExported() {
			panic(fmt.Sprintf("label struct fields must be exported (package %v):\ntype %s %s {\n\t%s %s `label:\"%v\"` // <-- field must be exported\n}", structType.PkgPath(), structType.Name(), structType.Kind(), field.Name, field.Type, labelName))
		}

		ft := field.Type
		if ft.Kind() != reflect.String {
			panic(fmt.Sprintf("label struct fields must be strings (package %v):\ntype %s %s {\n\t%s %s `label:\"%v\"` // <-- field must be string\n}", structType.PkgPath(), structType.Name(), structType.Kind(), field.Name, field.Type, labelName))
		}
		keys = append(keys, labelName)
	}

	labelCache.Store(t, keys)

	return keys
}

// getLabelValues extracts label values from a struct instance using the cached label keys.
// This function assumes that getLabelKeys[T]() has been called first to populate the cache.
func getLabelValues[T any](labelStruct T) prometheus.Labels {
	v := reflect.ValueOf(labelStruct)
	t := v.Type()

	cached, ok := labelCache.Load(t)
	if !ok {
		panic("unreachable, getLabelKeys should have cached the keys")
	}
	keys := cached.([]string)

	structValue := v
	if v.Kind() == reflect.Ptr {
		structValue = v.Elem()
	}

	labels := make(prometheus.Labels, len(keys))
	for i, key := range keys {
		valField := structValue.Field(i)
		labels[key] = valField.String()
	}

	return labels
}
