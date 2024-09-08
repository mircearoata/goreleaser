package extrafiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// RunTemplate generates files from templates.
func RunTemplate(ctx *context.Context, files []config.TemplatedExtraFile) (map[string]string, error) {
	t := tmpl.New(ctx)
	result := map[string]string{}
	for _, extra := range files {
		bts, err := os.ReadFile(extra.Src)
		if err != nil {
			return result, fmt.Errorf("failed to read file %q: %w", extra.Src, err)
		}
		content, err := t.Apply(string(bts))
		if err != nil {
			return result, fmt.Errorf("failed to apply template to content of %q: %w", extra.Src, err)
		}
		if content == "" {
			log.Warn("ignoring empty content")
			continue
		}

		file := filepath.Join(ctx.Config.Dist, extra.Dst)
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return result, fmt.Errorf("failed to write file %q: %w", file, err)
		}

		name := filepath.Base(file)
		if old, ok := result[name]; ok {
			log.Warnf("overriding %s with %s for name %s", old, file, name)
		}
		result[name] = file
	}
	return result, nil
}
