package registry

// Version is the registry server release identifier.
// Override at link time, e.g.:
//
//	go build -ldflags "-X github.com/getskillpack/registry.Version=1.2.3" -o registry ./cmd/registry
var Version = "0.0.0-dev"
