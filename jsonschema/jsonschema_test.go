// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseJSONString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Schema
		wantErr bool
	}{
		{
			name:  "valid schema",
			input: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			want: Schema{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{"type": "object", "properties": }`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJSONString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSONString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !cmp.Equal(got, tt.want) {
				t.Errorf("ParseJSONString() diff = %v", cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestSchemaIsValid(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		{
			name:    "valid schema - string",
			schema:  `{"type": "string"}`,
			wantErr: false,
		},
		{
			name:    "valid schema - object",
			schema:  `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			wantErr: false,
		},
		{
			name:    "invalid schema - invalid type",
			schema:  `{"type": "invalid", "properties": {"name": {"type": "string"}}}`,
			wantErr: true,
		},
		{
			name:    "invalid schema - wrong type",
			schema:  `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": "not-a-boolean"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSONString(tt.schema)
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("Schema.IsValid() error = %v, wantErr %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Schema
		data    string
		wantErr bool
	}{
		{
			name: "valid data",
			schema: Schema{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"age": map[string]interface{}{
						"type": "integer",
					},
				},
				"required": []interface{}{"name"},
			},
			data:    `{"name": "John", "age": 30}`,
			wantErr: false,
		},
		{
			name: "invalid data - wrong type",
			schema: Schema{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"age": map[string]interface{}{
						"type": "integer",
					},
				},
				"required": []interface{}{"name"},
			},
			data:    `{"name": "John", "age": "thirty"}`,
			wantErr: true,
		},
		{
			name: "invalid data - missing required field",
			schema: Schema{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"name"},
			},
			data:    `{"age": 30}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Schema.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
