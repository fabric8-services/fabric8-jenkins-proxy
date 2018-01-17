package main

import (
	"github.com/fabric8-services/fabric8-jenkins-proxy/cmd/osio/cmd"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

func main() {
	cmd.RootCmd.Execute()
}
