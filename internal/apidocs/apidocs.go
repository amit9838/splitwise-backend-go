// Package apidocs serves the embedded OpenAPI spec and the
// Swagger UI / ReDoc pages.
package apidocs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var spec []byte

//go:embed swagger.html
var swaggerHTML []byte

//go:embed redoc.html
var redocHTML []byte

// Spec handles GET /docs/openapi.yaml
func Spec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Write(spec)
}

// SwaggerUI handles GET /docs
func SwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(swaggerHTML)
}

// ReDoc handles GET /redoc
func ReDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(redocHTML)
}
