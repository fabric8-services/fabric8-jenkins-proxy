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
	wit                clients.WITService
	tenant             clients.TenantService
	repositoryCloneURL string
	logger             *log.Entry
}

// NewCodebase gets an instance of
func NewCodebase(wit clients.WITService, tenant clients.TenantService, repositoryCloneURL string, logger *log.Entry) *Codebase {
	return &Codebase{
		wit:                wit,
		tenant:             tenant,
		repositoryCloneURL: repositoryCloneURL,
		logger:             logger,
	}
}

// Namespace gives us details of user who owns given repository
func (c *Codebase) Namespace() (clients.Namespace, error) {
	wi, err := c.wit.SearchCodebase(c.repositoryCloneURL)
	if err != nil {
		return clients.Namespace{}, err
	}

	if len(strings.TrimSpace(wi.OwnedBy)) == 0 {
		return clients.Namespace{}, fmt.Errorf("unable to determine tenant id for repository %s", c.repositoryCloneURL)
	}

	c.logger.Infof("Found id %s for repo %s", wi.OwnedBy, c.repositoryCloneURL)
	ti, err := c.tenant.GetTenantInfo(wi.OwnedBy)
	if err != nil {
		return clients.Namespace{}, err
	}

	n, err := clients.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return clients.Namespace{}, err
	}

	return n, nil
}
