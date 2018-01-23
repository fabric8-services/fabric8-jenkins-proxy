package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

const (
	envPrefix = "OIO"
)

var RootCmd *cobra.Command

func init() {
	RootCmd = &cobra.Command{
		Use:   "osio",
		Short: "Osio is a helper tool for OpenShift.io.",
	}

	RootCmd.AddCommand(cmdToken)
	cobra.OnInitialize(initConfig)

}

func initConfig() {
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
