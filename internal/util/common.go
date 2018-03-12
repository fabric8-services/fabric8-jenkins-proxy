package util

const (
	stage = "stage"
	prod  = "prod"
)

type environment struct {
	osioURL      string
	apiURL       string
	authURL      string
	redirectURL  string
	privateKeyID string
}

var (
	environments = make(map[string]environment, 2)
)

func init() {
	environments[stage] = environment{
		osioURL:      "https://prod-preview.openshift.io",
		apiURL:       "https://api.openshift.io",
		authURL:      "https://auth.prod-preview.openshift.io",
		redirectURL:  "https://jenkins.prod-preview.openshift.io",
		privateKeyID: "PE6-BEECZZpPZIVxLR6NinbthOHJcGqYrfl8v7v6BVA", // key id for serviceaccount.privatekey
	}

	environments[prod] = environment{
		osioURL:      "https://openshift.io",
		apiURL:       "https://api.prod-preview.openshift.io",
		authURL:      "https://auth.openshift.io",
		redirectURL:  "https://jenkins.openshift.io",
		privateKeyID: "quzUZlR_ollAUoAGgm165tYDTU3xtKon8O1RghJZ4TU", // key id for serviceaccount.privatekey
	}
}
