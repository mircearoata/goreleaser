package merge

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

var pipe = Pipe{}

func TestString(t *testing.T) {
	require.NotEmpty(t, pipe.String())
}

func TestSkip(t *testing.T) {
	t.Run("merge", func(t *testing.T) {
		ctx := testctx.New(testctx.Merge)
		require.False(t, pipe.Skip(ctx))
	})

	t.Run("regular", func(t *testing.T) {
		require.True(t, pipe.Skip(testctx.New()))
	})
}

func getCtx(tmp string) *context.Context {
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        tmp,
			ProjectName: "name",
			Metadata: config.ProjectMetadata{
				ModTimestamp: "{{.Env.MOD_TS}}",
			},
		},
		testctx.WithPreviousTag("v1.2.2"),
		testctx.WithCurrentTag("v1.2.3"),
		testctx.WithCommit("aef34a"),
		testctx.WithVersion("1.2.3"),
		testctx.WithDate(time.Date(2022, 0o1, 22, 10, 12, 13, 0, time.UTC)),
		testctx.WithFakeRuntime,
	)
	return ctx
}

func TestRunWithError(t *testing.T) {
	t.Run("no dist", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:        "testadata/nope",
			ProjectName: "foo",
		})
		require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
	})
	t.Run("missing artifacts.json", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		require.NoError(t, os.Mkdir(filepath.Join(tmp, "linux"), 0o755))
		require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
	})
	t.Run("missing binaries", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		require.NoError(t, os.Mkdir(filepath.Join(tmp, "linux"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "linux", "artifacts.json"), []byte(`[{"path":"foo.txt"}]`), 0o644))
		require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
	})
}

func TestRun(t *testing.T) {
	t.Run("artifacts", func(t *testing.T) {
		tmp := t.TempDir()
		dist := filepath.Join(tmp, "dist")
		require.NoError(t, os.Mkdir(dist, 0o755))
		ctx := getCtx(dist)

		prevDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmp))
		t.Cleanup(func() { require.NoError(t, os.Chdir(prevDir)) })

		require.NoError(t, os.Mkdir(filepath.Join(dist, "linux"), 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(dist, "linux", "other"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dist, "linux", "other", "foo.txt"), []byte("foo"), 0o644))
		require.NoError(t,
			os.WriteFile(
				filepath.Join(dist, "linux", "artifacts.json"),
				[]byte(`[{"name":"otherfoo","path":"other/foo.txt","goos":"linux","goarch":"arm64","goarm":"7","internal_type":4,"type":"Binary","extra":{"foo":"bar"}}]`),
				0o644,
			),
		)

		require.NoError(t, os.Mkdir(filepath.Join(dist, "darwin"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dist, "darwin", "foo.txt"), []byte("foo"), 0o644))
		require.NoError(t,
			os.WriteFile(
				filepath.Join(dist, "darwin", "artifacts.json"),
				[]byte(`[{"name":"foo","path":"foo.txt","goos":"darwin","goarch":"amd64","goarm":"7","internal_type":4,"type":"Binary","extra":{"foo":"bar"}}]`),
				0o644,
			),
		)

		require.NoError(t, Pipe{}.Run(ctx))
		require.NoDirExists(t, filepath.Join(dist, "linux"))
		require.NoDirExists(t, filepath.Join(dist, "darwin"))
		require.FileExists(t, filepath.Join(dist, "foo.txt"))
		require.FileExists(t, filepath.Join(dist, "other", "foo.txt"))

		require.Equal(t, []*artifact.Artifact{
			{
				Name:   "foo",
				Path:   "dist/foo.txt",
				Goos:   "darwin",
				Goarch: "amd64",
				Goarm:  "7",
				Type:   artifact.Binary,
				TypeS:  "Binary",
				Extra:  map[string]interface{}{"foo": "bar"},
			},
			{
				Name:   "otherfoo",
				Path:   "dist/other/foo.txt",
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
