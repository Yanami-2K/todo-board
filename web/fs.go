package web

import "embed"

//go:embed index.html login.html intake.html openapi.yaml css js fonts
var FS embed.FS
