package mock

import (
	"errors"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
)

// Idler is a mock implementation of idler service
type Idler struct {
	idlerAPI   string
	IdlerState clients.PodState
	throwError bool // Used to test error scenarios
}

// NewMockIdler is a constructor for mock.Idler
func NewMockIdler(idlerAPI string, state clients.PodState, throwError bool) (idler *Idler) {
	return &Idler{
		idlerAPI:   idlerAPI,
		IdlerState: state,
		throwError: throwError,
	}
}

// State just returns the value set in Idler.state
func (i *Idler) State(tenant string, openShiftAPIURL string) (clients.PodState, error) {
	if i.throwError == true {
		return clients.UnknownState, errors.New("This error is invoked for mocking error scenarios")
	}
	if openShiftAPIURL == "Valid_OpenShift_API_URL" {
		return i.IdlerState, nil
	}

	return clients.UnknownState, errors.New("Invalid API URL")
}

// UnIdle always unidles (mock)
func (i *Idler) UnIdle(tenant string, openShiftAPIURL string) (int, error) {
	return 200, nil
}

// Clusters returns a map which maps the OpenShift API URL to the application DNS for this cluster. An empty map together with
// an error is returned if an error occurs.
func (i *Idler) Clusters() (map[string]string, error) {
	clusters := map[string]string{
		"https://api.free-stg.openshift.com/":           "1b7d.free-stg.openshiftapps.com",
		"https://api.starter-us-east-2a.openshift.com/": "b542.starter-us-east-2a.openshiftapps.com",
	}

	return clusters, nil
}
