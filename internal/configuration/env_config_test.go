package configuration

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func Test_without_required_parameters_new_configuration_returns_error(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	os.Clearenv()
	defer os.Clearenv()

	config, err := NewConfiguration()
	assert.Error(t, err, "There should have been an error.")
	assert.Nil(t, config, "No configuration should be returned.")
}

func Test_setting_required_parameters_lets_you_create_configuration(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	os.Clearenv()
	defer os.Clearenv()

	// set the required variables
	os.Setenv("JC_POSTGRES_HOST", "myhost")
	os.Setenv("JC_POSTGRES_PORT", "4532")
	os.Setenv("JC_POSTGRES_USER", "postgres")
	os.Setenv("JC_POSTGRES_PASSWORD", "mypass")
	os.Setenv("JC_POSTGRES_DATABASE", "foo")

	os.Setenv("JC_F8TENANT_API_URL", "http://localhost:1234")
	os.Setenv("JC_AUTH_URL", "http://localhost:1235")
	os.Setenv("JC_WIT_API_URL", "http://localhost:1236")
	os.Setenv("JC_KEYCLOAK_URL", "http://localhost:1237")
	os.Setenv("JC_IDLER_API_URL", "http://localhost:1238")

	os.Setenv("JC_AUTH_TOKEN", "snafu")
	os.Setenv("JC_REDIRECT_URL", "http://localhost")

	config, err := NewConfiguration()
	assert.NoError(t, err, "There should have been no error.")
	assert.NotNil(t, config, "There should be a syntactically valid configuration.")
}
