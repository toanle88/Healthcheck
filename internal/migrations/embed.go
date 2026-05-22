package migrations

import "embed"

// FS embeds the migrations directory
//
//go:embed *.sql
var FS embed.FS
