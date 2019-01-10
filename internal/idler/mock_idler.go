package idler

import (
	"errors"
)

// Mock is a mock implementation of idler service
type Mock struct {
	idlerAPI   string
	IdlerState PodState
	throwError bool // Used to test error scenarios
}

// NewMock is a constructor for mock.Idler
func NewMock(idlerAPI string, state PodState, throwError bool) (idler *Mock) {
	return &Mock{
		idlerAPI:   idlerAPI,
		IdlerState: state,
		throwError: throwError,
	}
}

// State just returns the value set in Idler.state
func (i *Mock) State(tenant string, openShiftAPIURL string) (PodState, error) {
	if i.throwError == true {
		return UnknownState, errors.New("This error is invoked for mocking error scenarios")
	}
	if openShiftAPIURL == "Valid_OpenShift_API_URL" {
		return i.IdlerState, nil
	}

	return UnknownState, errors.New("Invalid API URL")
}

// UnIdle always unidles (mock)
func (i *Mock) UnIdle(tenant string, openShiftAPIURL string) (int, error) {
	return 200, nil
}

// Clusters returns a map which maps the OpenShift API URL to the application DNS for this cluster. An empty map together with
// an error is returned if an error occurs.
func (i *Mock) Clusters() (map[string]string, error) {
	clusters := map[string]string{
		"https://api.free-stg.openshift.com/":           "1b7d.free-stg.openshiftapps.com",
		"https://api.starter-us-east-2a.openshift.com/": "b542.starter-us-east-2a.openshiftapps.com",
	}

	return clusters, nil
}
