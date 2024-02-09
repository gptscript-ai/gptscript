package static

import (
	"embed"
	_ "embed"
)

//go:embed * ui/_nuxt/*
var UI embed.FS
