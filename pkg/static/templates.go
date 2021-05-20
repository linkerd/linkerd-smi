//go:generate go run generate.go
// +build !production

package static

import (
	"net/http"
	"path"
	"path/filepath"
	"runtime"
)

// Templates that will be rendered by `linkerd smi install`. This is only used on
// dev builds.
var Templates http.FileSystem = http.Dir(path.Join(getRepoRoot(), "charts"))

// GetRepoRoot returns the full path to the root of the repo. We assume this
// function is only called from the `Templates` var above, and that this source
// file lives at `pkg/charts/static`, relative to the root of the repo.
func getRepoRoot() string {
	// /foo/bar/linkerd-smi/pkg/charts/static/templates.go
	_, filename, _, _ := runtime.Caller(0)

	// /foo/bar/linkerd-smi/pkg/charts/static
	dir := filepath.Dir(filename)

	// filepath.Dir returns the parent directory, so that combined with joining
	// ".." walks 1 level up the tree:
	// /foo/linkerd-smi
	return filepath.Dir(path.Join(dir, ".."))
}
