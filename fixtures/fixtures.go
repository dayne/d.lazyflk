// Package fixtures embeds real flk output from the reference host (see
// README.md) so --demo and the tests run without flk.
package fixtures

import "embed"

//go:embed *.json *.txt *.toml
var FS embed.FS
