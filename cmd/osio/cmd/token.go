package cmd

import (
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
	"net/url"
)

var (
	cmdToken = &cobra.Command{
		Use:   "token",
		Short: "Prints the JSON access token for a user.",
		Long:  `Prints the OpenShift IO JSON (JWT) accesss token for the provided Red Hat Developers credentials.`,
		Run:   runCreateToken,
	}
	targetEnv string
	username  string
	password  string
	encode    bool
)

func init() {
	cmdToken.Flags().StringVarP(&targetEnv, "target", "t", "stage", "Target environment OpenShift.io stage vs prod.")
	cmdToken.Flags().StringVarP(&username, "username", "u", "", "Red Hat Developer username.")
	cmdToken.Flags().StringVarP(&password, "password", "p", "", "Red Hat Developer password.")
	cmdToken.Flags().BoolVarP(&encode, "encode", "e", false, "Whether or not the output should be URL encoded.")
}

func runCreateToken(cmd *cobra.Command, args []string) {
	log.Debugf("Username: %s", username)
	log.Debugf("Password: %s", password)

	token, err := util.CreateAccessToken(targetEnv, username, password)
	if err != nil {
		log.Fatal(err)
	}

	if encode {
		token = url.QueryEscape(token)
	}
	fmt.Println(token)
}
