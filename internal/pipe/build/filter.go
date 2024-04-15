package build

import (
	"fmt"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

func filter(ctx *context.Context, targets []string) []string {
	if !ctx.Partial && !ctx.Split {
		return targets
	}

	var target string

	if ctx.Split {
		target = ctx.SplitTarget
	} else {
		target = ctx.PartialTarget
	}

	log.WithField("match", fmt.Sprintf("target=%s", target)).Infof("partial build")

	var result []string
	for _, t := range targets {
		if !strings.HasPrefix(t, target) {
			continue
		}
		result = append(result, t)
		break
	}
	return result
}
