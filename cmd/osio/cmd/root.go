package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	envPrefix = "OSIO"
)

var (
	// RootCmd is for using Cobra library for setting up command line interactions.
	RootCmd   *cobra.Command
	targetEnv string
)

func init() {
	RootCmd = &cobra.Command{
		Use:   "osio",
		Short: "osio is a helper tool for OpenShift.io.",
	}

	RootCmd.PersistentFlags().StringVarP(&targetEnv, "target", "t", "stage", "Target environment OpenShift.io stage vs prod.")

	RootCmd.AddCommand(cmdJWT)
	RootCmd.AddCommand(cmdToken)
	cobra.OnInitialize(initConfig)

}

func initConfig() {
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
