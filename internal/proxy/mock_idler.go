package proxy

type mockIdler struct {
	idlerAPI string
	isIdle   bool
}

// IsIdle returns mockIdler.isIdle
func (i *mockIdler) IsIdle(tenant string, openShiftAPIURL string) (bool, error) {
	return i.isIdle, nil
}

// UnIdle always unidles (mock)
func (i *mockIdler) UnIdle(tenant string, openShiftAPIURL string) error {
	return nil
}

// Clusters returns a map which maps the OpenShift API URL to the application DNS for this cluster. An empty map together with
// an error is returned if an error occurs.
func (i *mockIdler) Clusters() (map[string]string, error) {
	clusters := map[string]string{
		"https://api.free-stg.openshift.com/":           "1b7d.free-stg.openshiftapps.com",
		"https://api.starter-us-east-2a.openshift.com/": "b542.starter-us-east-2a.openshiftapps.com",
	}

	return clusters, nil
}
