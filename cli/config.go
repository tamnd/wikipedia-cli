package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect resolved configuration and paths",
	}
	cmd.AddCommand(newConfigShowCmd(app))
	return cmd
}

func newConfigShowCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"get"},
		Short:   "Print the effective configuration",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := app.Cfg
			site, host := "", ""
			if s, err := cfg.Site(); err == nil {
				site = s.Project
				host = s.Host
			}
			rows := [][2]string{
				{"lang", cfg.Lang},
				{"project", site},
				{"host", host},
				{"data-dir", cfg.DataDir},
				{"cache-dir", cfg.CacheDir},
				{"download-dir", cfg.DownloadDir()},
				{"config-dir", wiki.ConfigDir()},
				{"timeout", cfg.Timeout.String()},
				{"rate", cfg.Delay.String()},
				{"retries", itoa(cfg.Retries)},
				{"maxlag", itoa(cfg.Maxlag)},
				{"user-agent", cfg.UserAgent},
				{"allow-any-host", boolWord(cfg.AllowAnyHost, "true", "false")},
			}
			return emitKV(app, rows, configValue(cfg, host))
		},
	}
	return cmd
}

func configValue(cfg wiki.Config, host string) map[string]any {
	return map[string]any{
		"lang":           cfg.Lang,
		"project":        cfg.Project,
		"host":           host,
		"data_dir":       cfg.DataDir,
		"cache_dir":      cfg.CacheDir,
		"download_dir":   cfg.DownloadDir(),
		"config_dir":     wiki.ConfigDir(),
		"timeout":        cfg.Timeout.String(),
		"rate":           cfg.Delay.String(),
		"retries":        cfg.Retries,
		"maxlag":         cfg.Maxlag,
		"user_agent":     cfg.UserAgent,
		"allow_any_host": cfg.AllowAnyHost,
	}
}
