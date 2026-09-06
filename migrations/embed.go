// Package migrations embeds the ordered SQL migration files applied at startup.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
