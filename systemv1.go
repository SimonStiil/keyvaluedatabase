package main

import (
	"encoding/json"
	"net/http"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

type Systemv1 struct {
	Count             *Counter
	PrometheusHandler http.Handler
}

func (Api *Systemv1) ApiController(w http.ResponseWriter, request *RequestParameters) {
	switch request.Namespace {
	case "metrics":
		if Api.PrometheusHandler != nil {
			Api.PrometheusHandler.ServeHTTP(w, request.orgRequest)
			return
		}
	case "health":
		if Api.PrometheusHandler != nil {
			requests.WithLabelValues(request.orgRequest.URL.EscapedPath(), request.Method).Inc()
		}
		reply := rest.HealthV1{Status: "UP", Requests: int(Api.Count.PeakCount())}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reply)
		return
	}
	http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
}

func (api *Systemv1) Permissions(request *RequestParameters) *ConfigPermissions {
	switch request.Namespace {
	case "metrics":
		return &ConfigPermissions{}
	case "health":
		return &ConfigPermissions{}
	default:
		return &ConfigPermissions{Read: true, Write: true, List: true}
	}
}
func InitSystemv1(Prometheus http.Handler) *Systemv1 {
	return &Systemv1{PrometheusHandler: Prometheus}
}
