/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"csjhuang.net/nixos-sops-confgen/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// cfgFile for nixos-sops-confgen
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "nixos-sops-confgen",
		Short: "Generate .sops.yaml file are easier to manage by our nixos workflow",
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Past argument/flag validation, so any error from here on is a
			// runtime error — print just the error, not the usage block.
			cmd.SilenceUsage = true
			return initializeConfig()
		},
	}

	Cfg config.Config
)

func initializeConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("unmarshal config error: %w", err)
	}

	return nil
}

// rootCmd represents the base command when called without any subcommands

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nixos-sops-confgen.yaml)")
}
