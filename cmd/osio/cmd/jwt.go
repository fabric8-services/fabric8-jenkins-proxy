package cmd

import (
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
	"net/url"
)

var (
	cmdJWT = &cobra.Command{
		Use:   "jwt",
		Short: "Prints the JSON access token (JWT) for a user.",
		Long:  `Prints the OpenShift JSON Web Token for the provided Red Hat Developers credentials.`,
		Run:   runCreateJWT,
	}
	username string
	password string
	encode   bool
)

func init() {
	cmdJWT.Flags().StringVarP(&username, "username", "u", "", "Red Hat Developer username.")
	cmdJWT.Flags().StringVarP(&password, "password", "p", "", "Red Hat Developer password.")
	cmdJWT.Flags().BoolVarP(&encode, "encode", "e", false, "Whether or not the output should be URL encoded.")
}

func runCreateJWT(cmd *cobra.Command, args []string) {
	log.Debugf("Username: %s", username)
	log.Debugf("Password: %s", password)

	token, err := util.CreateJWTToken(targetEnv, username, password)
	if err != nil {
		log.Fatal(err)
	}

	if encode {
		token = url.QueryEscape(token)
	}
	fmt.Println(token)
}
