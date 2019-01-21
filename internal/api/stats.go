package api

import (
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api/app"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/goadesign/goa"
	log "github.com/sirupsen/logrus"
)

// StatsController implements the stats resource.
type StatsController struct {
	*goa.Controller
	storageService storage.Store
}

// NewStatsController creates a stats controller.
func NewStatsController(service *goa.Service,
	store storage.Store) *StatsController {
	return &StatsController{
		Controller:     service.NewController("StatsController"),
		storageService: store}
}

// Clear runs the clear action.
func (c *StatsController) Clear(ctx *app.ClearStatsContext) error {
	// StatsController_Clear: start_implement

	// Put your logic here

	res := &app.Stats{}
	return ctx.OK(res)
	// StatsController_Clear: end_implement
}

// Info runs the info action.
func (c *StatsController) Info(ctx *app.InfoStatsContext) error {
	// StatsController_Info: start_implement

	// Put your logic here
	ns := ctx.Namespace
	s, notFound, err := c.storageService.GetStatisticsUser(ns)
	if err != nil {
		if notFound {
			log.Debugf("Did not find data for %s", ns)
		} else {
			log.Error(err) //FIXME
			return ctx.InternalServerError(goa.ErrInternal(err))
		}
	}
	count, err := c.storageService.GetRequestsCount(ns)
	if err != nil {
		log.Error(err) //FIXME
		return ctx.InternalServerError(goa.ErrInternal(err))
	}

	res := &app.Stats{
		Namespace:   ns,
		Requests:    count,
		LastRequest: time.Unix(s.LastBufferedRequest, 0),
		LastVisit:   time.Unix(s.LastAccessed, 0),
	}
	return ctx.OK(res)
	// StatsController_Info: end_implement
}
