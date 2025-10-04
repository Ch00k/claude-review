package main

import (
	"embed"
)

//go:embed frontend/templates/*.html
var templatesFS embed.FS

//go:embed frontend/static/*.css frontend/static/*.js
var staticFS embed.FS

//go:embed slash-commands/*.md
var slashCommandsFS embed.FS
