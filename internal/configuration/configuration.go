package configuration

import "time"

// Configuration declares methods to get configuration of the proxy.
type Configuration interface {
	// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
	GetPostgresHost() string

	// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
	GetPostgresPort() int

	// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
	GetPostgresUser() string

	// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
	GetPostgresDatabase() string

	// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
	GetPostgresPassword() string

	// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
	GetPostgresSSLMode() string

	// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable
	GetPostgresConnectionTimeout() int

	// GetPostgresConnectionMaxIdle returns the number of connections that should be kept alive in the database connection pool at
	// any given time. -1 represents no restrictions/default behavior
	GetPostgresConnectionMaxIdle() int

	// GetPostgresConnectionMaxOpen returns the max number of open connections that should be open in the database connection pool.
	// -1 represents no restrictions/default behavior
	GetPostgresConnectionMaxOpen() int

	// GetIdlerURL returns the Idler API URL as set via default, config file, or environment variable
	GetIdlerURL() string

	// GetAuthURL returns the Auth API URL as set via default, config file, or environment variable
	GetAuthURL() string

	// GetTenantURL returns the F8 Tenant API URL as set via default, config file, or environment variable
	GetTenantURL() string

	// GetWitURL returns the WIT API URL as set via default, config file, or environment variable
	GetWitURL() string

	// GetAuthToken returns the Auth token as set via default, config file, or environment variable
	GetAuthToken() string

	// GetRedirectURL returns the redirect url to be passed to Auth as set via default, config file, or environment variable
	GetRedirectURL() string

	// GetIndexPath returns the path to loading page template as set via default, config file, or environment variable
	GetIndexPath() string

	// GetMaxRequestRetry returns the number of retries for webhook request forwarding as set via default, config file,
	// or environment variable
	GetMaxRequestRetry() int

	// GetDebugMode returns if debug mode should be enabled as set via default, config file, or environment variable
	GetDebugMode() bool

	// GetHTTPSEnabled returns if https should be enabled as set via default, config file, or environment variable
	GetHTTPSEnabled() bool

	// GetGatewayTimeout returns the interval within which reverse proxy expects
	// a response from the underlying jenkins server
	GetGatewayTimeout() time.Duration

	// GetAllowedOrigins returns string containing allowed origins separated with ", "
	GetAllowedOrigins() []string

	// String returns a string representation of the configuration
	String() string
}
