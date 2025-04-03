// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package chat

import (
	"os"
	"strings"
	"testing"
)

func TestModelCatalog(t *testing.T) {
	f, err := os.Open("../testdata/modelcatalog.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	catalog, err := NewModelCatalog(f)
	if err != nil {
		t.Fatal(err)
	}

	name := "gemini-2.0-flash"
	m := catalog.GetModel(name)
	if m == nil {
		t.Fatalf("model %s not found", name)
	}

	if !strings.Contains(m.Model, name) {
		t.Fatalf("model %s name is not expected: %s", name, m.Model)
	}
}

func TestCalculateCost(t *testing.T) {
	m := &ModelInfo{
		InputTokenCost:  1.5e-7,
		OutputTokenCost: 6e-7,
	}

	usage := &Usage{
		InputTokens:  300,
		OutputTokens: 300,
	}

	cost := calculateCost(m, usage)
	if cost != 0.000225 {
		t.Fatalf("cost is not expected: %f", cost)
	}
}

func TestDefaultModelCatalog(t *testing.T) {
	catalog := defaultModelCatalog()
	if catalog == nil {
		t.Fatal("default model catalog is nil")
	}
}
