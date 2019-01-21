package api

import (
	"reflect"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/goadesign/goa"
	"github.com/golang/mock/gomock"
)

func TestNewStatsController(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := storage.NewMockStore(ctrl)

	service := goa.New("fabric8-jenkins-proxy-api")

	want := &StatsController{
		storageService: store}

	got := NewStatsController(service, store)
	if !reflect.DeepEqual(got.storageService, want.storageService) {
		t.Errorf("NewStatsController() = %v, want %v", got, want)
	}
}
