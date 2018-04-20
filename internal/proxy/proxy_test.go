package proxy

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils/mock"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	dockertest "gopkg.in/ory-am/dockertest.v3"
)

func Test_constructRoute(t *testing.T) {
	proxy := Proxy{
		clusters: map[string]string{
			"https://api.free-stg.openshift.com/":           "1b7d.free-stg.openshiftapps.com",
			"https://api.starter-us-east-2a.openshift.com/": "b542.starter-us-east-2a.openshiftapps.com",
		},
	}

	// Successful case
	expectedRoute := "jenkins-sampleNamespace.1b7d.free-stg.openshiftapps.com"
	retrievedRoute, scheme, err := proxy.constructRoute("https://api.free-stg.openshift.com/", "sampleNamespace")

	assert.Nil(t, err)
	assert.Equal(t, expectedRoute, retrievedRoute)
	assert.Equal(t, "https", scheme)

	// Failing case
	retrievedRoute, scheme, err = proxy.constructRoute("some_invalid_url", "sampleNamespace")

	assert.NotNil(t, err)
	assert.Equal(t, "could not find entry for cluster some_invalid_url", err.Error())
	assert.Equal(t, "", retrievedRoute)
	assert.Equal(t, "", scheme)
}

func Test_recordStatistics(t *testing.T) {
	proxy, pool, resource := setupDependencyServices(true)
	assert.NotNil(t, proxy)
	defer pool.Purge(resource)
	// Let's try to record a statistics that doesn't exist in our database,
	// i.e., given namespace doesn't exist in the db
	err := proxy.recordStatistics("namespace_that_doesnt_exist", 123, 123)
	assert.Nil(t, err)
	// Check if data has been saved in db
	statistics, notFound, err := proxy.storageService.GetStatisticsUser("namespace_that_doesnt_exist")
	assert.Nil(t, err)
	assert.False(t, notFound)
	assert.Equal(t, int64(123), statistics.LastAccessed)
	assert.Equal(t, int64(123), statistics.LastBufferedRequest)

	// It should update if namespace of statistics is found, but lastAccessed or LastBufferedRequest are different
	err = proxy.recordStatistics("namespace_that_doesnt_exist", int64(123), int64(165))
	assert.Nil(t, err)
	// Check if data has been saved in db
	statistics, notFound, err = proxy.storageService.GetStatisticsUser("namespace_that_doesnt_exist")
	assert.Nil(t, err)
	assert.False(t, notFound)
	assert.Equal(t, int64(123), statistics.LastAccessed)
	assert.Equal(t, int64(165), statistics.LastBufferedRequest)
}

func Test_checkCookies(t *testing.T) {
	proxy, pool, resource := setupDependencyServices(true)
	assert.NotNil(t, proxy)
	defer pool.Purge(resource)

	req := httptest.NewRequest("GET", "/doesnt_matter", nil)
	w := httptest.NewRecorder()
	proxyLogger := log.WithFields(log.Fields{"component": "proxy"})

	// Scenario 1: User info is not cached , i.e., cookie not found needs to relogin
	proxy.loginInstance = &mockLogin{}
	cacheKey, namespace, noProxy, needsAuth := proxy.checkCookies(w, req, proxyLogger)
	assert.True(t, needsAuth)
	assert.True(t, noProxy)
	assert.Empty(t, cacheKey)
	assert.Empty(t, namespace)

	// Scenario 2: only jenkins idle cookie is found, user needs to login, and info will be available
	cacheItem := CacheItem{
		ClusterURL: "https://api.free-stg.openshift.com/",
		NS:         "someNameSpace",
		Route:      "1b7d.free-stg.openshiftapps.com",
		Scheme:     "",
	}

	c := &http.Cookie{}
	id := uuid.NewV4().String()
	c.Name = CookieJenkinsIdled
	c.Value = id
	// Store proxyCacheItem at id in cache
	proxy.ProxyCache.SetDefault(id, cacheItem)
	req.AddCookie(c)
	// lets mock login user
	proxy.loginInstance = &mockLogin{}
	cacheKey, namespace, noProxy, needsAuth = proxy.checkCookies(w, req, proxyLogger)
	assert.False(t, needsAuth)
	assert.True(t, noProxy)
	assert.Equal(t, id, cacheKey)
	assert.Equal(t, cacheItem.NS, namespace)

	// Scenario 3: If we find session cookie in the cache, user does'nt need to login, info is available
	c.Name = SessionCookie
	req = httptest.NewRequest("GET", "/doesnt_matter", nil)
	req.AddCookie(c)
	proxy.loginInstance = &login{}
	cacheKey, namespace, noProxy, needsAuth = proxy.checkCookies(w, req, proxyLogger)
	assert.False(t, needsAuth)
	assert.False(t, noProxy)
	assert.Equal(t, id, cacheKey)
	assert.Equal(t, cacheItem.NS, namespace)

}

func Test_processAuthenticatedRequest(t *testing.T) {
	proxy, pool, resource := setupDependencyServices(true)
	assert.NotNil(t, proxy)
	defer pool.Purge(resource)

	req := httptest.NewRequest("GET", "/doesnt_matter", nil)
	w := httptest.NewRecorder()
	proxyLogger := log.WithFields(log.Fields{"component": "proxy"})

	// Scenario 1: fails with empty tokenJSON
	noProxy := true
	tokenJSON := []string{}
	namespace := proxy.processAuthenticatedRequest(w, req, proxyLogger, tokenJSON, &noProxy)
	assert.Empty(t, namespace)
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "could not read JWT token from URL")

	// Scenario 2: fails if tokenJSON has a an invalid token
	tokenJSON = []string{"someinvalid_token"}
	w = httptest.NewRecorder()
	namespace = proxy.processAuthenticatedRequest(w, req, proxyLogger, tokenJSON, &noProxy)
	assert.Empty(t, namespace)
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "invalid character 's' looking for beginning of value")

	// Scenario 3: With Valid tokenJSON and Jenkins is idle, jenkins idle cookie is set
	// Mock processToken
	proxy.loginInstance = &mockLogin{
		isLoggedIn:   true,
		isTokenValid: true,
	}
	proxy.idler = &mockIdler{
		isIdle: true,
	}
	tokenJSON = []string{"valid_token"}
	w = httptest.NewRecorder()
	namespace = proxy.processAuthenticatedRequest(w, req, proxyLogger, tokenJSON, &noProxy)
	assert.Equal(t, "someNameSpace", namespace)
	assert.Equal(t, 302, w.Code)
	assert.Contains(t, w.Body.String(), "Found")
	// How do I test if cookies are set?

	// Scenario 4: With Valid tokenJSON and Jenkins is not idle, jenkins idle cookie is set
	// Mock processToken, GetOSOToken, loginJenkins
	proxy.loginInstance = &mockLogin{
		isLoggedIn:   true,
		isTokenValid: true,
		giveOSOToken: true,
	}
	proxy.idler = &mockIdler{
		isIdle: false,
	}
	tokenJSON = []string{"valid_token"}
	w = httptest.NewRecorder()
	namespace = proxy.processAuthenticatedRequest(w, req, proxyLogger, tokenJSON, &noProxy)
	assert.Equal(t, "someNameSpace", namespace)
	assert.Equal(t, 302, w.Code)
	assert.Contains(t, w.Body.String(), "Found")
}

func setupTestDatabase(mockConfig *mock.Config) (*dockertest.Pool, *dockertest.Resource, error) {

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Errorf("Could not connect to docker: %s", err)
		return nil, nil, err
	}

	resource, err := pool.Run("postgres", "9.6", []string{"POSTGRES_PASSWORD=" + mockConfig.GetPostgresPassword(), "POSTGRES_DB=" + mockConfig.GetPostgresDatabase()})
	if err != nil {
		log.Errorf("Could not start resource: %s", err)
		return nil, nil, err
	}

	if err = pool.Retry(func() error {
		var err error
		db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", mockConfig.GetPostgresUser(), mockConfig.GetPostgresPassword(), resource.GetPort("5432/tcp"), mockConfig.GetPostgresDatabase()))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Errorf("Could not connect to docker: %s", err)
		err1 := pool.Purge(resource)
		if err != nil {
			log.Fatalf("Could not remove resource: %s", err1)
		}
		return nil, nil, err
	}

	port, _ := strconv.Atoi(resource.GetPort("5432/tcp"))
	mockConfig.PostgresPort = port

	return pool, resource, nil
}

func setupDependencyServices(needDB bool) (Proxy, *dockertest.Pool, *dockertest.Resource) {
	configuration := mock.NewConfig()
	var pool *dockertest.Pool
	var resource *dockertest.Resource
	var err error
	if needDB {
		pool, resource, err = setupTestDatabase(&configuration)
		if err != nil {
			panic("invalid test " + err.Error())
		}
	}

	configuration.IdlerURL = "doesnt_matter"

	// Connect to DB
	db, err := storage.Connect(&configuration)
	if err != nil {
		panic("invalid test " + err.Error())
	}

	store := storage.NewDBStorage(db)

	// Create tenant client
	tenant := clients.NewTenant(configuration.GetTenantURL(), configuration.GetAuthToken())

	// Create WorkItemTracker client
	wit := clients.NewWIT(configuration.GetWitURL(), configuration.GetAuthToken())

	// Create Idler client
	idler := mockIdler{
		idlerAPI: configuration.IdlerURL,
		isIdle:   false,
	}

	// Get the cluster view from the Idler
	clusters := map[string]string{
		"https://api.free-stg.openshift.com/":           "1b7d.free-stg.openshiftapps.com",
		"https://api.starter-us-east-2a.openshift.com/": "b542.starter-us-east-2a.openshiftapps.com",
	}

	proxy, err := NewProxy(&tenant, &wit, &idler, store, &configuration, clusters)
	if err != nil {
		panic(err.Error())
	}

	return proxy, pool, resource
}
