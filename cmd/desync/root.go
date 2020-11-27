package main

import (
	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "desync",
		Short: "Content-addressed binary distribution system.",
	}
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.config/desync/config.json)")
	cmd.PersistentFlags().StringVar(&digestAlgorithm, "digest", "sha512-256", "digest algorithm, sha512-256 or sha256")
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level, if log filename or verbose is define (debug, info, warn, error). debug default")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose mode")
	cmd.PersistentFlags().StringVar(&logFileName, "log-filename", "", "log filename, if any. stop verbose mode")
	cmd.PersistentFlags().StringVar(&monitorAddress, "mon-addr", "", "json health monitor bind address")
	cmd.PersistentFlags().IntVar(&monitorFailoverLen, "mon-fo-len", 10, "size of the failovers rotate tab (default 10)")
	return cmd
}
