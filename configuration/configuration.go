package configuration

import (
	"fmt"
	"strings"
	"time"
	"net/url"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	varPostgresHost                    = "postgres.host"
	varPostgresPort                    = "postgres.port"
	varPostgresUser                    = "postgres.user"
	varPostgresDatabase                = "postgres.database"
	varPostgresPassword                = "postgres.password"
	varPostgresSSLMode                 = "postgres.sslmode"
	varPostgresConnectionTimeout       = "postgres.connection.timeout"
	varPostgresConnectionRetrySleep    = "postgres.connection.retrysleep"
	varPostgresConnectionMaxIdle       = "postgres.connection.maxidle"
	varPostgresConnectionMaxOpen       = "postgres.connection.maxopen"

	varIdlerURL                        = "idler.api.url"
	varAuthToken                       = "auth.token"
	varAuthURL                         = "auth.url"
	varTenantURL                       = "f8tenant.api.url"
	varWitURL                          = "wit.api.url"
	varRedirectURL                     = "redirect.url"
	varKeycloakURL                     = "keycloak.url"
	varMaxRequestRetry                 = "max.request.retry"

	varIndexPath                       = "index.path"

	varLocalDevEnv                     = "local.dev.env"
)

// Data encapsulates the Viper configuration object which stores the configuration data in-memory.
type Data struct {
	v *viper.Viper
}

// NewData creates a configuration reader object 
func NewData() (*Data, error) {
	c := Data{
		v: viper.New(),
	}
	c.v.SetEnvPrefix("JC")
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()

	return &c, nil
}

func (c *Data) setConfigDefaults() {
	//---------
	// Postgres
	//---------
	c.v.SetTypeByDefaultValue(true)
	c.v.SetDefault(varPostgresHost, "localhost")
	c.v.SetDefault(varPostgresPort, 5432)
	c.v.SetDefault(varPostgresUser, "postgres")
	c.v.SetDefault(varPostgresDatabase, "tenant")
	c.v.SetDefault(varPostgresPassword, "mysecretpassword")
	c.v.SetDefault(varPostgresSSLMode, "disable")
	c.v.SetDefault(varPostgresConnectionTimeout, 5)
	c.v.SetDefault(varPostgresConnectionMaxIdle, -1)
	c.v.SetDefault(varPostgresConnectionMaxOpen, -1)

	// Number of seconds to wait before trying to connect again
	c.v.SetDefault(varPostgresConnectionRetrySleep, time.Duration(time.Second))

	c.v.SetDefault(varMaxRequestRetry, 10)

	c.v.SetDefault(varIndexPath, "/opt/fabric8-jenkins-proxy/index.html")
}

func (c *Data) VerifyConfig() {
	missingParam := false
	apiURL := c.GetIdlerURL()
	_, err := url.ParseRequestURI(apiURL)
	if len(apiURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide URL to Idler API endpoint in JC_IDLER_API_URL environment variable")
	}

	authToken := c.GetAuthToken()
	if len(authToken) == 0 {
		missingParam = true
		log.Error("You need to provide fabric8-auth token")
	}
	tenantApiURL := c.GetTenantURL()
	_, err = url.ParseRequestURI(tenantApiURL)
	if len(tenantApiURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide fabric8-tenant service URL")
	}

	witApiURL := c.GetWitURL()
	_, err = url.ParseRequestURI(witApiURL)
	if len(witApiURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide WIT API service URL")
	}

	redirURL := c.GetRedirectURL()
	_, err = url.ParseRequestURI(redirURL)
	if len(redirURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide redirect URL")
	}

	keycloakURL := c.GetKeycloakURL()
	_, err = url.ParseRequestURI(keycloakURL)
	if len(keycloakURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide Keycloak URL")
	}

	authURL := c.GetAuthURL()
	_, err = url.ParseRequestURI(authURL)
	if len(authURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide Auth service URL")
	}


	if missingParam {
		log.Fatal("A value for envinronment variable(s) is missing")
	}
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func (c *Data) GetPostgresConfigString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		c.GetPostgresHost(),
		c.GetPostgresPort(),
		c.GetPostgresUser(),
		c.GetPostgresPassword(),
		c.GetPostgresDatabase(),
		c.GetPostgresSSLMode(),
		c.GetPostgresConnectionTimeout(),
	)
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func (c *Data) GetPostgresHost() string {
	return c.v.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func (c *Data) GetPostgresPort() int {
	return c.v.GetInt(varPostgresPort)	
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func (c *Data) GetPostgresUser() string {
	return c.v.GetString(varPostgresUser)
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func (c *Data) GetPostgresDatabase() string {
	return c.v.GetString(varPostgresDatabase)
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func (c *Data) GetPostgresPassword() string {
	return c.v.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func (c *Data) GetPostgresSSLMode() string {
	return c.v.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable
func (c *Data) GetPostgresConnectionTimeout() int {
	return c.v.GetInt(varPostgresConnectionTimeout)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func (c *Data) GetPostgresConnectionRetrySleep() time.Duration {
	return c.v.GetDuration(varPostgresConnectionRetrySleep)
}

// GetPostgresConnectionMaxIdle returns the number of connections that should be keept alive in the database connection pool at
// any given time. -1 represents no restrictions/default behavior
func (c *Data) GetPostgresConnectionMaxIdle() int {
	return c.v.GetInt(varPostgresConnectionMaxIdle)
}

// GetPostgresConnectionMaxOpen returns the max number of open connections that should be open in the database connection pool.
// -1 represents no restrictions/default behavior
func (c *Data) GetPostgresConnectionMaxOpen() int {
	return c.v.GetInt(varPostgresConnectionMaxOpen)
}

// GetIdlerURL returns the Idler API URL as set via default, config file, or environment variable
func (c *Data) GetIdlerURL() string {
	return c.v.GetString(varIdlerURL)
}

// GetAuthURL returns the Auth API URL as set via default, config file, or environment variable
func (c *Data) GetAuthURL() string {
	return c.v.GetString(varAuthURL)
}

// GetTenantURL returns the F8 Tenant API URL as set via default, config file, or environment variable
func (c *Data) GetTenantURL() string {
	return c.v.GetString(varTenantURL)
}

// GetWitURL returns the WIT API URL as set via default, config file, or environment variable
func (c *Data) GetWitURL() string {
	return c.v.GetString(varWitURL)
}

// GetKeycloakURL returns the Keycloak API URL as set via default, config file, or environment variable
func (c *Data) GetKeycloakURL() string {
	return c.v.GetString(varKeycloakURL)
}

// GetAuthToken returns the Auth token as set via default, config file, or environment variable
func (c *Data) GetAuthToken() string {
	return c.v.GetString(varAuthToken)
}

// GetRedirectURL returns the redirect url to be passed to Auth as set via default, config file, or environment variable
func (c *Data) GetRedirectURL() string {
	return c.v.GetString(varRedirectURL)
}

// GetIndexPath returns the path to loading page template as set via default, config file, or environment variable
func (c *Data) GetIndexPath() string {
	return c.v.GetString(varIndexPath)
}

// GetMaxRequestretry returns the number of retries for webhook request forwarding as set via default, config file,
// or environment variable
func (c *Data) GetMaxRequestretry() int {
	return c.v.GetInt(varMaxRequestRetry)
}

// GetLocalDevEnv returns if it is local development env as set via default, config file, or environment variable
func (c *Data) GetLocalDevEnv() bool {
	return c.v.GetBool(varLocalDevEnv)
}