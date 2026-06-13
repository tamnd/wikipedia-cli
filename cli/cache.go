package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCacheCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Inspect or clear the on-disk response cache",
	}
	cmd.AddCommand(newCachePathCmd(app), newCacheInfoCmd(app), newCacheClearCmd(app))
	return cmd
}

func newCachePathCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the cache directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmdOut, app.Cache.Dir())
			return err
		},
	}
}

func newCacheInfoCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show cache entry count and total size",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			count, bytes := app.Cache.Info()
			rows := [][2]string{
				{"dir", app.Cache.Dir()},
				{"entries", itoa(count)},
				{"size", humanSize(bytes)},
			}
			return emitKV(app, rows, map[string]any{
				"dir": app.Cache.Dir(), "entries": count, "bytes": bytes,
			})
		},
	}
}

func newCacheClearCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove every cached response",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := app.Cache.Clear()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmdOut, "removed %d cache entries\n", n)
			return err
		},
	}
}
