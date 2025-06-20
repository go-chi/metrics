package metrics

import (
	"fmt"
	"reflect"
	"regexp"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	labelCache sync.Map // map[reflect.Type][]string

	labelValidator = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
)

// mustBeValidMetricName validates a metric name and panics if it doesn't match the required Prometheus format.
func mustBeValidMetricName(s string) string {
	if !labelValidator.MatchString(s) {
		panic("invalid metric name: " + s + " (must match [a-z_][a-z0-9_]*)")
	}
	return s
}

// mustStructLabelKeys returns the list of labels defined as struct tags,
// e.g. `label:"some_name"`, and panics if any of the labels are empty or invalid.
func mustStructLabelKeys[T any]() []string {
	var keys []string
	var zero T
	v := reflect.ValueOf(zero)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return keys
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		labelName := field.Tag.Get("label")
		if labelName == "" {
			panic(fmt.Sprintf("missing `label` struct tag in %v:\ntype %s %s {\n\t%s %s `label:\"add_label_here\"`\n}", t.PkgPath(), t.Name(), t.Kind(), field.Name, field.Type))
		}
		if !labelValidator.MatchString(labelName) {
			panic(fmt.Sprintf("invalid `label` name in %v:\ntype %s %s {\n\t%s %s `label:\"%s\"` // <-- must match [a-z_][a-z0-9_]*\n}", t.PkgPath(), t.Name(), t.Kind(), field.Name, field.Type, labelName))
		}
		keys = append(keys, labelName)
	}
	labelCache.Store(t, keys)

	return keys
}

func mustStructLabels[T any](labelStruct T) prometheus.Labels {
	labels := make(prometheus.Labels)
	v := reflect.ValueOf(labelStruct)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return labels
	}

	t := v.Type()
	var keys []string
	if cached, ok := labelCache.Load(t); ok {
		keys = cached.([]string)
	} else {
		panic("unreachable, mustStructLabelKeys should have cached the keys")
	}

	for i, key := range keys {
		valField := v.Field(i)
		if valField.IsValid() && valField.CanInterface() {
			val := valField.Interface()
			switch v := val.(type) {
			case string:
				labels[key] = v
			default:
				labels[key] = reflect.ValueOf(val).String()
			}
		}
	}

	return labels
}
