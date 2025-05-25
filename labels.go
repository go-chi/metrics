package metrics

import (
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	labelCache sync.Map // map[reflect.Type][]string
)

type NoLabels struct{}

var labelSanitizer = regexp.MustCompile(`[^a-z0-9_]`)

func sanitizeLabel(s string) string {
	s = labelSanitizer.ReplaceAllString(strings.ToLower(s), "_")
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "_" + s // Prometheus label names can't start with a digit
	}
	return s
}

func labelKeys[T any]() []string {
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
		label := sanitizeLabel(t.Field(i).Name)
		keys = append(keys, label)
	}
	return keys
}

func structToLabels[T any](labelStruct T) prometheus.Labels {
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
		keys = make([]string, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			keys[i] = sanitizeLabel(t.Field(i).Name)
		}
		labelCache.Store(t, keys)
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
