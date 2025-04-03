// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Schema is a JSON schema as a map[string]any.
type Schema map[string]any

// ParseJSONString parses a JSONSchema JSON string and returns a Schema.
func ParseJSONString(s string) (Schema, error) {
	var sch Schema
	if err := json.Unmarshal([]byte(s), &sch); err != nil {
		return nil, err
	}
	if !sch.IsValid() {
		return nil, fmt.Errorf("invalid schema")
	}
	return sch, nil
}

// MustParseJSONString parses a JSONSchema JSON string and returns a Schema.
// It panics if the schema is invalid.
func MustParseJSONString(s string) Schema {
	sch, err := ParseJSONString(s)
	if err != nil {
		panic(err)
	}
	return sch
}

// JSON returns the JSON representation of the schema.
func (s Schema) JSON() json.RawMessage {
	js, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return js
}

func (s Schema) PropertiesJSON() json.RawMessage {
	js, err := json.Marshal(s["properties"])
	if err != nil {
		return nil
	}
	return js
}

// IsValid checks if the schema is valid.
func (s Schema) IsValid() bool {
	js := s.JSON()
	sch, err := jsonschema.UnmarshalJSON(bytes.NewReader(js))
	if err != nil {
		return false
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", sch); err != nil {
		return false
	}
	_, err = c.Compile("schema.json")
	return err == nil
}

// Validate validates the data against the schema.
func (s Schema) Validate(data []byte) error {
	return validate(s.JSON(), data)
}

func validate(schema []byte, data []byte) error {
	c := jsonschema.NewCompiler()
	s, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	err = c.AddResource("schema.json", s)
	if err != nil {
		return fmt.Errorf("failed to add schema: %w", err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var instance interface{}
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return sch.Validate(instance)
}
