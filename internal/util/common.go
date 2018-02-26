package util

const (
	stage = "stage"
	prod  = "prod"
)

type environment struct {
	osioUrl      string
	apiUrl       string
	authURL      string
	redirectURL  string
	privateKeyId string
}

var (
	environments = make(map[string]environment, 2)
)

func init() {
	environments[stage] = environment{
		osioUrl:      "https://prod-preview.openshift.io",
		apiUrl:       "https://api.openshift.io",
		authURL:      "https://auth.prod-preview.openshift.io",
		redirectURL:  "https://jenkins.prod-preview.openshift.io",
		privateKeyId: "PE6-BEECZZpPZIVxLR6NinbthOHJcGqYrfl8v7v6BVA", // key id for serviceaccount.privatekey
	}

	environments[prod] = environment{
		osioUrl:      "https://openshift.io",
		apiUrl:       "https://api.prod-preview.openshift.io",
		authURL:      "https://auth.openshift.io",
		redirectURL:  "https://jenkins.openshift.io",
		privateKeyId: "quzUZlR_ollAUoAGgm165tYDTU3xtKon8O1RghJZ4TU", // key id for serviceaccount.privatekey
	}
}
