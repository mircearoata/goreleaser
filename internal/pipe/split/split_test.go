package split

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

var pipe = Pipe{}

func TestString(t *testing.T) {
	require.NotEmpty(t, pipe.String())
}

func TestSkip(t *testing.T) {
	t.Run("split", func(t *testing.T) {
		ctx := testctx.New(testctx.Split)
		require.False(t, pipe.Skip(ctx))
	})

	t.Run("full", func(t *testing.T) {
		require.True(t, pipe.Skip(testctx.New()))
	})
}

func TestRun(t *testing.T) {
	t.Run("target", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Split)
		t.Setenv("GOOS", "windows")
		t.Setenv("GOARCH", "arm64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "windows", ctx.SplitTarget)
		require.Equal(t, filepath.Join("dist", "windows"), ctx.Config.Dist)
	})
	t.Run("using GGOOS and GGOARCH", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Split)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "amd64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "linux", ctx.SplitTarget)
		require.Equal(t, filepath.Join("dist", "linux"), ctx.Config.Dist)
	})
	t.Run("using runtime", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Split)
		require.NoError(t, pipe.Run(ctx))
		target := runtime.GOOS
		require.Equal(t, target, ctx.SplitTarget)
		require.Equal(t, filepath.Join("dist", target), ctx.Config.Dist)
	})
}

func TestStringArtifacts(t *testing.T) {
	require.NotEmpty(t, ArtifactsPipe{}.String())
}

func TestSkipArtifacts(t *testing.T) {
	t.Run("split", func(t *testing.T) {
		ctx := testctx.New(testctx.Split)
		require.False(t, ArtifactsPipe{}.Skip(ctx))
	})

	t.Run("full", func(t *testing.T) {
		require.True(t, ArtifactsPipe{}.Skip(testctx.New()))
	})
}

func TestRunArtifacts(t *testing.T) {
	t.Run("target", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist/darwin",
		}, testctx.Split)
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "foo",
			Path:   "dist/darwin/foo.txt",
			Goos:   "darwin",
			Goarch: "amd64",
			Goarm:  "7",
			Type:   artifact.Binary,
			TypeS:  "Binary",
			Extra:  map[string]interface{}{"foo": "bar"},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "otherfoo",
			Path:   "dist/darwin/other/foo.txt",
			Goos:   "linux",
			Goarch: "arm64",
			Goarm:  "7",
			Type:   artifact.Binary,
			TypeS:  "Binary",
			Extra:  map[string]interface{}{"foo": "bar"},
		})
		require.NoError(t, ArtifactsPipe{}.Run(ctx))
		require.Equal(t, []*artifact.Artifact{
			{
				Name:   "foo",
				Path:   "foo.txt",
				Goos:   "darwin",
				Goarch: "amd64",
				Goarm:  "7",
				Type:   artifact.Binary,
				TypeS:  "Binary",
				Extra:  map[string]interface{}{"foo": "bar"},
			},
			{
				Name:   "otherfoo",
				Path:   "other/foo.txt",
				Goos:   "linux",
				Goarch: "arm64",
				Goarm:  "7",
				Type:   artifact.Binary,
				TypeS:  "Binary",
				Extra:  map[string]interface{}{"foo": "bar"},
			},
		}, ctx.Artifacts.List())
	})
}
