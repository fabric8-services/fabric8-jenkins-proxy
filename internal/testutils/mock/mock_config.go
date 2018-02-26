package mock

type Config struct {
	PostgresHost              string
	PostgresPort              int
	PostgresUser              string
	PostgresDatabase          string
	PostgresPassword          string
	PostgresSSLMode           string
	PostgresConnectionTimeout int
	PostgresConnectionMaxIdle int
	PostgresConnectionMaxOpen int
	IdlerURL                  string
	AuthURL                   string
	TenantURL                 string
	WitURL                    string
	KeycloakURL               string
	AuthToken                 string
	RedirectURL               string
	IndexPath                 string
	MaxRequestRetry           int
	DebugMode                 bool
	Clusters                  map[string]string
}

func NewConfig() Config {
	c := Config{}

	// set some defaults which can be overridden as needed
	c.PostgresHost = "localhost"
	c.PostgresPort = 5432
	c.PostgresUser = "postgres"
	c.PostgresPassword = "postgres"
	c.PostgresDatabase = "tenant"
	c.PostgresSSLMode = "disable"
	c.PostgresConnectionMaxIdle = -1
	c.PostgresConnectionMaxOpen = -1
	c.PostgresConnectionTimeout = 5

	c.KeycloakURL = "https://sso.prod-preview.openshift.io"

	c.RedirectURL = "https://localhost:8443/"
	c.IndexPath = "static/html/index.html"
	c.Clusters = make(map[string]string)
	c.Clusters["https://api.free-stg.openshift.com/"] = "1b7d.free-stg.openshiftapps.com"

	return c
}

func (c *Config) GetPostgresHost() string {
	return c.PostgresHost
}

func (c *Config) GetPostgresPort() int {
	return c.PostgresPort
}

func (c *Config) GetPostgresUser() string {
	return c.PostgresUser
}

func (c *Config) GetPostgresDatabase() string {
	return c.PostgresDatabase
}

func (c *Config) GetPostgresPassword() string {
	return c.PostgresPassword
}

func (c *Config) GetPostgresSSLMode() string {
	return c.PostgresSSLMode
}

func (c *Config) GetPostgresConnectionTimeout() int {
	return c.PostgresConnectionTimeout
}

func (c *Config) GetPostgresConnectionMaxIdle() int {
	return c.PostgresConnectionMaxIdle
}

func (c *Config) GetPostgresConnectionMaxOpen() int {
	return c.PostgresConnectionMaxOpen
}

func (c *Config) GetIdlerURL() string {
	return c.IdlerURL
}

func (c *Config) GetAuthURL() string {
	return c.AuthURL
}

func (c *Config) GetTenantURL() string {
	return c.TenantURL
}

func (c *Config) GetWitURL() string {
	return c.WitURL
}

func (c *Config) GetKeycloakURL() string {
	return c.KeycloakURL
}

func (c *Config) GetAuthToken() string {
	return c.AuthToken
}

func (c *Config) GetRedirectURL() string {
	return c.RedirectURL
}

func (c *Config) GetIndexPath() string {
	return c.IndexPath
}

func (c *Config) GetMaxRequestRetry() int {
	return c.MaxRequestRetry
}

func (c *Config) GetDebugMode() bool {
	return c.DebugMode
}

func (c *Config) GetClusters() map[string]string {
	return c.Clusters
}

func (c *Config) String() string {
	return "mockConfig"
}
