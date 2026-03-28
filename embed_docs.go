package registry

import _ "embed"

//go:embed docs/registry-api.md
var RegistryAPIMarkdown []byte

//go:embed docs/og-card.svg
var OGCardSVG []byte
