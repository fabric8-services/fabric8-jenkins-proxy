// Code generated by goagen v1.3.1, DO NOT EDIT.
//
// API "fabric8-jenkins-proxy-api": Application Media Types
//
// Command:
// $ goagen
// --design=github.com/fabric8-services/fabric8-jenkins-proxy/internal/design
// --out=$(GOPATH)/src/github.com/fabric8-services/fabric8-jenkins-proxy/internal/api
// --version=v1.3.1

package app

import (
	"github.com/goadesign/goa"
	"time"
)

// Response from Fabric8-Jenkins-Proxy (default view)
//
// Identifier: application/vnd.stats+json; view=default
type Stats struct {
	LastRequest time.Time `form:"last_request" json:"last_request" yaml:"last_request" xml:"last_request"`
	LastVisit   time.Time `form:"last_visit" json:"last_visit" yaml:"last_visit" xml:"last_visit"`
	// Unique Namespace
	Namespace string `form:"namespace" json:"namespace" yaml:"namespace" xml:"namespace"`
	Requests  int    `form:"requests" json:"requests" yaml:"requests" xml:"requests"`
}

// Validate validates the Stats media type instance.
func (mt *Stats) Validate() (err error) {
	if mt.Namespace == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "namespace"))
	}

	return
}
