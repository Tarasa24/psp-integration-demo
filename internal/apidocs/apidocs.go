// Package apidocs embeds the OpenAPI spec and Swagger UI for serving via HTTP.
package apidocs

import _ "embed"

// OpenAPISpec is the raw OpenAPI 3.0 YAML specification.
//
//go:embed openapi.yaml
var OpenAPISpec []byte

// SwaggerUI is the standalone Swagger UI HTML page.
//
//go:embed swagger-ui.html
var SwaggerUI []byte
