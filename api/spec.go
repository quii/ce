// Package api embeds the OpenAPI contract so it can be served without
// depending on a file being present next to the running binary (the
// production image only ships the compiled binary - see Dockerfile).
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
