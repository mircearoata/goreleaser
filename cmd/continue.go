package cmd

import (
	"fmt"
	"runtime"
	"time"

	"github.com/caarlos0/ctrlc"
	"github.com/caarlos0/log"
	"github.com/spf13/cobra"

	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipe/git"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type continueCmd struct {
	cmd  *cobra.Command
	opts continueOpts
}

type continueOpts struct {
	config            string
	releaseNotesFile  string
	releaseNotesTmpl  string
	releaseHeaderFile string
	releaseHeaderTmpl string
	releaseFooterFile string
	releaseFooterTmpl string
	autoSnapshot      bool
	snapshot          bool
	draft             bool
	failFast          bool
	merge             bool
	parallelism       int
	timeout           time.Duration
	skips             []string
}

func newContinueCmd() *continueCmd {
	root := &continueCmd{}
	// nolint: dupl
	cmd := &cobra.Command{
		Use:               "continue",
		Aliases:           []string{"r"},
		Short:             "Continues the current project",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: timedRunE("continue", func(_ *cobra.Command, _ []string) error {
			ctx, err := continueProject(root.opts)
			if err != nil {
				return err
			}
			deprecateWarn(ctx)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&root.opts.config, "config", "f", "", "Load configuration from file")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.Flags().StringVar(&root.opts.releaseNotesFile, "release-notes", "", "Load custom release notes from a markdown file (will skip GoReleaser changelog generation)")
	_ = cmd.MarkFlagFilename("release-notes", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseHeaderFile, "release-header", "", "Load custom release notes header from a markdown file")
	_ = cmd.MarkFlagFilename("release-header", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseFooterFile, "release-footer", "", "Load custom release notes footer from a markdown file")
	_ = cmd.MarkFlagFilename("release-footer", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseNotesTmpl, "release-notes-tmpl", "", "Load custom release notes from a templated markdown file (overrides --release-notes)")
	_ = cmd.MarkFlagFilename("release-notes-tmpl", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseHeaderTmpl, "release-header-tmpl", "", "Load custom release notes header from a templated markdown file (overrides --release-header)")
	_ = cmd.MarkFlagFilename("release-header-tmpl", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseFooterTmpl, "release-footer-tmpl", "", "Load custom release notes footer from a templated markdown file (overrides --release-footer)")
	_ = cmd.MarkFlagFilename("release-footer-tmpl", "md", "mkd", "markdown")
	cmd.Flags().BoolVar(&root.opts.autoSnapshot, "auto-snapshot", false, "Automatically sets --snapshot if the repository is dirty")
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts (implies --skip=announce,publish,validate)")
	cmd.Flags().BoolVar(&root.opts.draft, "draft", false, "Whether to set the release to draft. Overrides release.draft in the configuration file")
	cmd.Flags().BoolVar(&root.opts.failFast, "fail-fast", false, "Whether to abort the release publishing on the first error")
	cmd.Flags().BoolVar(&root.opts.merge, "merge", false, "Merge a release that has been split into multiple steps")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Amount tasks to run concurrently (default: number of CPUs)")
	_ = cmd.RegisterFlagCompletionFunc("parallelism", cobra.NoFileCompletions)
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire release process")
	_ = cmd.RegisterFlagCompletionFunc("timeout", cobra.NoFileCompletions)
	cmd.Flags().StringSliceVar(
		&root.opts.skips,
		"skip",
		nil,
		fmt.Sprintf("Skip the given options (valid options are %s)", skips.Release.String()),
	)
	_ = cmd.RegisterFlagCompletionFunc("skip", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return skips.Release.Complete(toComplete), cobra.ShellCompDirectiveDefault
	})

	root.cmd = cmd
	return root
}

func continueProject(options continueOpts) (*context.Context, error) {
	cfg, err := loadConfig(options.config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.NewWithTimeout(cfg, options.timeout)
	defer cancel()
	if err := setupContinueContext(ctx, options); err != nil {
		return nil, err
	}
	return ctx, ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range pipeline.ContinueCmdPipeline {
			if err := skip.Maybe(
				pipe,
				logging.Log(
					pipe.String(),
					errhandler.Handle(pipe.Run),
				),
			)(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func setupContinueContext(ctx *context.Context, options continueOpts) error {
	ctx.Action = context.ActionContinue
	ctx.Parallelism = runtime.GOMAXPROCS(0)
	if options.parallelism > 0 {
		ctx.Parallelism = options.parallelism
	}
	log.Debugf("parallelism: %v", ctx.Parallelism)
	ctx.ReleaseNotesFile = options.releaseNotesFile
	ctx.ReleaseNotesTmpl = options.releaseNotesTmpl
	ctx.ReleaseHeaderFile = options.releaseHeaderFile
	ctx.ReleaseHeaderTmpl = options.releaseHeaderTmpl
	ctx.ReleaseFooterFile = options.releaseFooterFile
	ctx.ReleaseFooterTmpl = options.releaseFooterTmpl
	ctx.Snapshot = options.snapshot
	ctx.FailFast = options.failFast
	ctx.Clean = false
	ctx.Merge = options.merge
	if options.autoSnapshot && git.CheckDirty(ctx) != nil {
		log.Info("git repository is dirty and --auto-snapshot is set, implying --snapshot")
		ctx.Snapshot = true
	}

	ctx.Config.Release.Draft = options.draft

	if err := skips.SetRelease(ctx, options.skips...); err != nil {
		return err
	}

	if ctx.Snapshot {
		skips.Set(ctx, skips.Publish, skips.Announce, skips.Validate)
	}
	if skips.Any(ctx, skips.Publish) {
		skips.Set(ctx, skips.Announce)
	}

	if skips.Any(ctx, skips.Release...) {
		log.Warnf(
			logext.Warning("skipping %s..."),
			skips.String(ctx),
		)
	}
	return nil
}
