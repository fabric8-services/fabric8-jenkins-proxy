package jenkinsapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/jenkinsapi"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils/mock"

	"github.com/stretchr/testify/assert"
)

func Test_Start(t *testing.T) {
	tenant, idler := setupDependencyServices()
	jenkinsapi := jenkinsapi.NewJenkinsAPI(tenant, idler)

	r := httptest.NewRequest("GET", "/doesntmatter", nil)
	r.Header.Set("Authorization", "Bearer InvalidToken")
	w := httptest.NewRecorder()
	jenkinsapi.Start(w, r, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	r = httptest.NewRequest("GET", "/doesntmatter", nil)
	r.Header.Set("Authorization", "Bearer ValidToken")
	w = httptest.NewRecorder()
	jenkinsapi.Start(w, r, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	fmt.Printf(w.Body.String())
	assert.Equal(t, "{\"data\":{\"state\":\""+string(idler.IdlerState)+"\"}}\n", w.Body.String())
}

func Test_Status(t *testing.T) {
	tenant, idler := setupDependencyServices()
	jenkinsapi := jenkinsapi.NewJenkinsAPI(tenant, idler)

	r := httptest.NewRequest("GET", "/someendpoint", nil)
	r.Header.Set("Authorization", "Bearer InvalidToken")
	w := httptest.NewRecorder()
	jenkinsapi.Status(w, r, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func Test_Status_unauthorized(t *testing.T) {
	tenant, idler := setupDependencyServices()
	jenkinsapi := jenkinsapi.NewJenkinsAPI(tenant, idler)

	r := httptest.NewRequest("GET", "/someendpoint", nil)
	r.Header.Set("Authorization", "Bearer ValidToken")
	w := httptest.NewRecorder()
	jenkinsapi.Status(w, r, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	fmt.Printf(w.Body.String())
	assert.Equal(t, "{\"data\":{\"state\":\""+string(idler.IdlerState)+"\"}}\n", w.Body.String())
}

func Test_Status_bad_idler(t *testing.T) {
	failedTenant, failedIdler := setupBadDependencyServices()
	failedJenkinsAPI := jenkinsapi.NewJenkinsAPI(failedTenant, failedIdler)

	r := httptest.NewRequest("GET", "/someendpoint", nil)
	r.Header.Set("Authorization", "Bearer ValidToken")
	w := httptest.NewRecorder()
	failedJenkinsAPI.Status(w, r, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func setupDependencyServices() (clients.TenantService, *mock.Idler) {
	configuration := mock.NewConfig()

	configuration.IdlerURL = "doesnt_matter"

	// Create tenant client
	tenant := mock.Tenant{}

	// Create Idler client
	idler := mock.NewMockIdler(configuration.IdlerURL, clients.PodState("idled"), false)

	return &tenant, idler
}

func setupBadDependencyServices() (clients.TenantService, *mock.Idler) {
	configuration := mock.NewConfig()

	configuration.IdlerURL = "doesnt_matter"

	// Create tenant client
	tenant := mock.Tenant{}

	// Create Idler client
	idler := mock.NewMockIdler(configuration.IdlerURL, clients.PodState("idled"), true)

	return &tenant, idler
}
