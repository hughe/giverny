package giverny

import "embed"

// Source holds the embedded source code needed to build the innie Docker image.
// This allows giverny to build the innie image without requiring
// access to the source code directory at runtime.
//
//go:embed cmd internal go.mod source.go
var Source embed.FS
