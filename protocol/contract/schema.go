package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func RepoProtocolDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "protocol"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func SchemaDir() string {
	return filepath.Join(RepoProtocolDir(), "schemas", "json")
}

func ProtoPath() string {
	return filepath.Join(RepoProtocolDir(), "schemas", "proto", "oncode_tool.proto")
}

func FixtureDir() string {
	return filepath.Join(RepoProtocolDir(), "tests", "fixtures")
}

func LoadJSON(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func ValidateRequired(schemaPath string, instance any) error {
	schema, err := LoadJSON(schemaPath)
	if err != nil {
		return err
	}
	return validate(schema, instance, schemaPath)
}

func validate(schema map[string]any, instance any, schemaPath string) error {
	if ref, ok := schema["$ref"].(string); ok {
		dir := filepath.Dir(schemaPath)
		refSchema, err := LoadJSON(filepath.Join(dir, ref))
		if err != nil {
			return err
		}
		return validate(refSchema, instance, filepath.Join(dir, ref))
	}
	obj, ok := instance.(map[string]any)
	if !ok {
		if instance == nil {
			return fmt.Errorf("instance is null")
		}
		return nil
	}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			name, _ := r.(string)
			if _, exists := obj[name]; !exists {
				return fmt.Errorf("missing required field %q", name)
			}
		}
	}
	props, _ := schema["properties"].(map[string]any)
	for k, v := range obj {
		ps, _ := props[k].(map[string]any)
		if ps == nil {
			continue
		}
		if enum, ok := ps["enum"].([]any); ok {
			if !inEnum(v, enum) {
				return fmt.Errorf("field %q value %v not in enum", k, v)
			}
		}
		if nested, ok := ps["properties"]; ok && nested != nil {
			if err := validate(ps, v, schemaPath); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
		}
		if ref, ok := ps["$ref"].(string); ok {
			dir := filepath.Dir(schemaPath)
			if err := validate(map[string]any{"$ref": ref}, v, filepath.Join(dir, "placeholder")); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
		}
	}
	return nil
}

func inEnum(v any, enum []any) bool {
	s := fmt.Sprint(v)
	for _, e := range enum {
		if fmt.Sprint(e) == s {
			return true
		}
	}
	return false
}
