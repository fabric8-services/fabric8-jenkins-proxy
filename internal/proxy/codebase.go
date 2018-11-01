package proxy

import (
	"fmt"
	"strings"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	log "github.com/sirupsen/logrus"
)

// CodebaseService contains methods that deals with code repository and
// code hosting services
type CodebaseService interface {
	Namespace(repositoryCloneURL string) (clients.Namespace, error)
}

// Codebase is an implementation of the codebase interface
type Codebase struct {
	wit    clients.WIT
	tenant clients.TenantService

	logger *log.Entry
}

// NewCodebase gets an instance of
func NewCodebase(wit clients.WIT, tenant clients.TenantService, logger *log.Entry) *Codebase {
	return &Codebase{
		wit:    wit,
		tenant: tenant,
		logger: logger,
	}
}

// Namespace gives us details of user who owns given repository
func (codebase *Codebase) Namespace(repositoryCloneURL string) (clients.Namespace, error) {
	wi, err := codebase.wit.SearchCodebase(repositoryCloneURL)
	if err != nil {
		return clients.Namespace{}, err
	}

	if len(strings.TrimSpace(wi.OwnedBy)) == 0 {
		return clients.Namespace{}, fmt.Errorf("unable to determine tenant id for repository %s", repositoryCloneURL)
	}

	codebase.logger.Infof("Found id %s for repo %s", wi.OwnedBy, repositoryCloneURL)
	ti, err := codebase.tenant.GetTenantInfo(wi.OwnedBy)
	if err != nil {
		return clients.Namespace{}, err
	}

	n, err := clients.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return clients.Namespace{}, err
	}

	return n, nil
}
