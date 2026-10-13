// Package schemas embeds Specmetry's exchange contracts (JSON Schema
// 2020-12). The JSON files in this directory are the normative source;
// internal/contract compiles and applies them.
package schemas

import "embed"

// FS holds every *.schema.json file in this directory.
//
//go:embed *.schema.json
var FS embed.FS

// BaseURL is the common prefix of every schema's $id.
const BaseURL = "https://github.com/XKHoshizora/specmetry/schemas/"
