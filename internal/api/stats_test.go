package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api/app"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/goadesign/goa"
	"github.com/golang/mock/gomock"
)

func TestNewStatsController(t *testing.T) {
	type args struct {
		service *goa.Service
		store   storage.Store
	}
	tests := []struct {
		name string
		args args
		want *StatsController
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStatsController(tt.args.service, tt.args.store); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStatsController() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatsController_Clear(t *testing.T) {

}

func TestStatsController_Info(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := storage.NewMockStore(ctrl)

	service := goa.New("fabric8-jenkins-proxy-api")

	statsCtrl := NewStatsController(service, store)

	app.MountStatsController(service, statsCtrl)

	ns := "test"
	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/info/%v", ns),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	prms["namespace"] = []string{fmt.Sprintf("%v", ns)}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "StatsTest"), rw, req, prms)
	infoCtx, err := app.NewInfoStatsContext(goaCtx, req, service)
	if err != nil {
		t.Errorf("Error fail to create info stat context = %v", err)
	}

	tests := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "Not Found From Mock",
			wantErr: false,
			mock: func() {
				store.EXPECT().GetStatisticsUser(ns).Return(&storage.Statistics{}, true, errors.New("Not Found"))
				store.EXPECT().GetRequestsCount(ns).Return(5, nil)

			},
		},
		{
			name:    "Error from GetStatisticsUser",
			wantErr: false,
			mock: func() {
				store.EXPECT().GetStatisticsUser(ns).Return(&storage.Statistics{}, false, errors.New("Internal Error"))
			},
		},
		{
			name:    "Error from GetRequestsCount",
			wantErr: false,
			mock: func() {
				store.EXPECT().GetStatisticsUser(ns).Return(&storage.Statistics{}, false, nil)
				store.EXPECT().GetRequestsCount(ns).Return(0, errors.New("Failed to Get RC"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			if err := statsCtrl.Info(infoCtx); (err != nil) != tt.wantErr {
				t.Errorf("StatsController.Info() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
