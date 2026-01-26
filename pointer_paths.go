package formagent

import (
	"reflect"
	"strings"
)

func AllJSONPointerPaths[T any]() []string {
	var zero T
	typ := reflect.TypeOf(zero)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return []string{}
	}

	paths := make([]string, 0)
	visited := make(map[reflect.Type]bool)
	collectPaths(typ, "", &paths, visited)
	return paths
}

func collectPaths(typ reflect.Type, prefix string, paths *[]string, visited map[reflect.Type]bool) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if visited[typ] {
		return
	}

	switch typ.Kind() {
	case reflect.Struct:
		visited[typ] = true
		defer func() { delete(visited, typ) }()

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}

			jsonName := getJSONFieldName(field)
			if jsonName == "" || jsonName == "-" {
				continue
			}

			fieldPath := prefix + "/" + jsonName
			*paths = append(*paths, fieldPath)
			collectPaths(field.Type, fieldPath, paths, visited)
		}

	case reflect.Slice, reflect.Array:
		elemType := typ.Elem()
		arrayPath := prefix + "/-"
		*paths = append(*paths, arrayPath)

		if elemType.Kind() == reflect.Struct ||
			(elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct) {
			collectPaths(elemType, arrayPath, paths, visited)
		}

	case reflect.Map:
		valueType := typ.Elem()
		mapPath := prefix + "/*"
		*paths = append(*paths, mapPath)

		if valueType.Kind() == reflect.Struct ||
			(valueType.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct) {
			collectPaths(valueType, mapPath, paths, visited)
		}
	default:
		break
	}
}

func getJSONFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}

	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return field.Name
}
