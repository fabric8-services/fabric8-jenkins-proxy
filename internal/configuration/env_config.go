package configuration

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	defaultPostgresSSLMode           = "disable"
	defaultPostgresConnectionTimeout = "5"
	defaultPostgresConnectionMaxIdle = "-1"
	defaultPostgresConnectionMaxOpen = "-1"
	defaultIndexPath                 = "/opt/fabric8-jenkins-proxy/index.html"
	defaultMaxRequestRetry           = "10"
	defaultDebugMode                 = "false"
	defaultHTTPSEnabled              = "false"
	defaultGatewayTimeout            = "25s"
	defaultAllowedOrigins            = "https://*openshift.io,https://localhost:*,http://localhost:*"
)

var (
	settings = map[string]Setting{}
	logger   = log.WithFields(log.Fields{"component": "configuration"})
)

func init() {
	// Postgres
	settings["GetPostgresHost"] = Setting{"JC_POSTGRES_HOST", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetPostgresPort"] = Setting{"JC_POSTGRES_PORT", "", []func(interface{}, string) error{util.IsInt}}
	settings["GetPostgresDatabase"] = Setting{"JC_POSTGRES_DATABASE", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetPostgresUser"] = Setting{"JC_POSTGRES_USER", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetPostgresPassword"] = Setting{"JC_POSTGRES_PASSWORD", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetPostgresSSLMode"] = Setting{"JC_POSTGRES_SSL_MODE", defaultPostgresSSLMode, []func(interface{}, string) error{}}
	settings["GetPostgresConnectionTimeout"] = Setting{"JC_POSTGRES_CONNECTION_TIMEOUT", defaultPostgresConnectionTimeout, []func(interface{}, string) error{util.IsInt}}
	settings["GetPostgresConnectionMaxIdle"] = Setting{"JC_POSTGRES_CONNECTION_MAX_IDLE", defaultPostgresConnectionMaxIdle, []func(interface{}, string) error{util.IsInt}}
	settings["GetPostgresConnectionMaxOpen"] = Setting{"JC_POSTGRES_CONNECTION_MAX_OPEN", defaultPostgresConnectionMaxOpen, []func(interface{}, string) error{util.IsInt}}

	// Service URLs
	settings["GetIdlerURL"] = Setting{"JC_IDLER_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetAuthURL"] = Setting{"JC_AUTH_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetAuthToken"] = Setting{"JC_AUTH_TOKEN", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetTenantURL"] = Setting{"JC_F8TENANT_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetWitURL"] = Setting{"JC_WIT_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetKeycloakURL"] = Setting{"JC_KEYCLOAK_URL", "", []func(interface{}, string) error{util.IsURL}}

	// Misc
	settings["GetRedirectURL"] = Setting{"JC_REDIRECT_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetIndexPath"] = Setting{"JC_INDEX_PATH", defaultIndexPath, []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetMaxRequestRetry"] = Setting{"JC_MAX_REQUEST_RETRY", defaultMaxRequestRetry, []func(interface{}, string) error{util.IsInt}}
	settings["GetDebugMode"] = Setting{"JC_DEBUG_MODE", defaultDebugMode, []func(interface{}, string) error{util.IsBool}}
	settings["GetHTTPSEnabled"] = Setting{"JC_ENABLE_HTTPS", defaultHTTPSEnabled, []func(interface{}, string) error{util.IsBool}}
	settings["GetGatewayTimeout"] = Setting{"JC_GATEWAY_TIMEOUT", defaultGatewayTimeout, []func(interface{}, string) error{util.IsDuration}}
	settings["GetAllowedOrigins"] = Setting{"JC_ALLOWED_ORIGINS", defaultAllowedOrigins, []func(interface{}, string) error{util.IsNotEmpty}}
}

// Setting is an element in the proxy configuration. It contains the environment
// variable from which the setting is retrieved, its default value as well as a list
// of validations which the value of this setting needs to pass.
type Setting struct {
	key          string
	defaultValue string
	validations  []func(interface{}, string) error
}

// EnvConfig is a Configuration implementation which reads the configuration from the process environment.
type EnvConfig struct {
	clusters map[string]string
}

// NewConfiguration creates a configuration instance.
func NewConfiguration() (Configuration, error) {
	// Check if we have all we need.
	multiError := verifyEnv()
	if !multiError.Empty() {
		for _, err := range multiError.Errors {
			logger.Error(err)
		}
		return nil, errors.New("one or more required environment variables are missing or invalid")
	}

	config := EnvConfig{}
	return &config, nil
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable.
func (c *EnvConfig) GetPostgresHost() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable.
func (c *EnvConfig) GetPostgresPort() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable.
func (c *EnvConfig) GetPostgresUser() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable.
func (c *EnvConfig) GetPostgresDatabase() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable.
func (c *EnvConfig) GetPostgresPassword() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable.
func (c *EnvConfig) GetPostgresSSLMode() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable.
func (c *EnvConfig) GetPostgresConnectionTimeout() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetPostgresConnectionMaxIdle returns the number of connections that should be keept alive in the database connection pool at
// any given time. -1 represents no restrictions/default behavior.
func (c *EnvConfig) GetPostgresConnectionMaxIdle() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetPostgresConnectionMaxOpen returns the max number of open connections that should be open in the database connection pool.
// -1 represents no restrictions/default behavior.
func (c *EnvConfig) GetPostgresConnectionMaxOpen() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetIdlerURL returns the Idler API URL as set via default, config file, or environment variable.
func (c *EnvConfig) GetIdlerURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetAuthURL returns the Auth API URL as set via default, config file, or environment variable.
func (c *EnvConfig) GetAuthURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetTenantURL returns the F8 Tenant API URL as set via default, config file, or environment variable.
func (c *EnvConfig) GetTenantURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetWitURL returns the WIT API URL as set via default, config file, or environment variable.
func (c *EnvConfig) GetWitURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetKeycloakURL returns the Keycloak API URL as set via default, config file, or environment variable.
func (c *EnvConfig) GetKeycloakURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetAuthToken returns the Auth token as set via default, config file, or environment variable.
func (c *EnvConfig) GetAuthToken() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetRedirectURL returns the redirect url to be passed to Auth as set via default, config file, or environment variable.
func (c *EnvConfig) GetRedirectURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetIndexPath returns the path to loading page template as set via default, config file, or environment variable.
func (c *EnvConfig) GetIndexPath() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetMaxRequestRetry returns the number of retries for webhook request forwarding as set via default, config file,
// or environment variable.
func (c *EnvConfig) GetMaxRequestRetry() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetDebugMode returns if debug mode should be enabled as set via default, config file, or environment variable.
func (c *EnvConfig) GetDebugMode() bool {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	b, _ := strconv.ParseBool(value)
	return b
}

// GetHTTPSEnabled returns if https should be enabled as set via default, config file, or environment variable.
func (c *EnvConfig) GetHTTPSEnabled() bool {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	b, _ := strconv.ParseBool(value)
	return b
}

// GetGatewayTimeout returns the duration within which the reverse-proxy expects
// a response.
func (c *EnvConfig) GetGatewayTimeout() time.Duration {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	d, _ := time.ParseDuration(value)
	return d
}

// GetAllowedOrigins returns string containing allowed origins separated with ", "
func (c *EnvConfig) GetAllowedOrigins() []string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return strings.Split(value, ",")
}

func (c *EnvConfig) String() string {
	config := map[string]interface{}{}
	for key, setting := range settings {
		value := getConfigValueFromEnv(key)
		// don't echo tokens or passwords
		if strings.Contains(setting.key, "TOKEN") && len(value) > 0 {
			value = "***"
		}
		if strings.Contains(setting.key, "PASSWORD") && len(value) > 0 {
			value = "***"
		}
		config[key] = value

	}
	return fmt.Sprintf("%v", config)
}

// Verify checks whether all needed config options are set.
func verifyEnv() util.MultiError {
	var errors util.MultiError
	for key, setting := range settings {
		value := getConfigValueFromEnv(key)

		for _, validateFunc := range setting.validations {
			errors.Collect(validateFunc(value, setting.key))
		}
	}

	return errors
}

func getConfigValueFromEnv(funcName string) string {
	setting := settings[funcName]

	value, ok := os.LookupEnv(setting.key)
	if !ok {
		value = setting.defaultValue
	}
	return value
}
