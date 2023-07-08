package templates

import _ "embed"

//go:embed 404.html
var NotFound string

//go:embed status.html
var Status string

//go:embed profile.html
var Profile string

//go:embed explore.html
var Explore string
