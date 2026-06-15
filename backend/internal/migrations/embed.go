// Package migrations embarque les fichiers SQL de migration dans le binaire.
package migrations

import "embed"

// FS contient les fichiers de migration (NNNN_name.up.sql / .down.sql),
// appliqués dans l'ordre lexicographique par le store.
//
//go:embed *.sql
var FS embed.FS
