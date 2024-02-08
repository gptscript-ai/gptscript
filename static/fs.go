package static

import (
	"embed"
	_ "embed"
)

// /go:embed ui/** ui/**/_*
var UI embed.FS
