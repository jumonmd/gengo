// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package chat

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed modelcatalog.json
var modelCatalog []byte

type Options struct {
	Streamer     Streamer
	ModelCatalog ModelCatalog
}

type Option func(o *Options)

func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	if o.ModelCatalog == nil {
		o.ModelCatalog = defaultModelCatalog()
	}
	return o
}

func WithStream(streamer Streamer) Option {
	return func(o *Options) {
		o.Streamer = streamer
	}
}

func WithModelCatalog(catalog ModelCatalog) Option {
	return func(o *Options) {
		o.ModelCatalog = catalog
	}
}

func defaultModelCatalog() ModelCatalog {
	var catalog ModelCatalog
	if err := json.Unmarshal(modelCatalog, &catalog); err != nil {
		panic(fmt.Sprintf("unmarshal model catalog: %v", err))
	}
	return catalog
}
