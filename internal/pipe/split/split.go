package split

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/x/exp/ordered"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string                 { return "split" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Split }

func (Pipe) Run(ctx *context.Context) error {
	ctx.SplitTarget = getFilter()
	ctx.Config.Dist = filepath.Join(ctx.Config.Dist, ctx.SplitTarget)
	return nil
}

func getFilter() string {
	goos := ordered.First(os.Getenv("GGOOS"), os.Getenv("GOOS"), runtime.GOOS)
	return goos
}

type ArtifactsPipe struct{}

func (ArtifactsPipe) String() string                 { return "split-artifacts" }
func (ArtifactsPipe) Skip(ctx *context.Context) bool { return !ctx.Split }

func (ArtifactsPipe) Run(ctx *context.Context) error {
	return ctx.Artifacts.Visit(func(a *artifact.Artifact) error {
		var err error
		a.Path, err = filepath.Rel(ctx.Config.Dist, a.Path)
		a.Path = filepath.ToSlash(a.Path)
		return err
	})
}
