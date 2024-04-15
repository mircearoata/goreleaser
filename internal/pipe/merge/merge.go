package merge

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string                 { return "merge" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Merge }

func (Pipe) Run(ctx *context.Context) error {
	items, err := os.ReadDir(ctx.Config.Dist)
	if err != nil {
		return err
	}
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		splitDir := filepath.Join(ctx.Config.Dist, item.Name())

		artifactsFile := filepath.Join(splitDir, "artifacts.json")
		artifacts, err := os.ReadFile(artifactsFile)
		if err != nil {
			return err
		}
		var a []*artifact.Artifact
		err = json.Unmarshal(artifacts, &a)
		if err != nil {
			return err
		}

		for _, a1 := range a {
			if a1.Type == artifact.Metadata {
				continue
			}
			oldArtifactPath := filepath.Join(splitDir, a1.Path)
			newArtifactPath := filepath.Join(ctx.Config.Dist, a1.Path)
			a1.Path = newArtifactPath
			ctx.Artifacts.Add(a1)
			err = os.MkdirAll(filepath.Dir(newArtifactPath), 0o755)
			if err != nil {
				return err
			}
			err = os.Rename(oldArtifactPath, newArtifactPath)
			if err != nil {
				return err
			}
		}
		err = os.RemoveAll(splitDir)
		if err != nil {
			return err
		}
	}

	return nil
}
