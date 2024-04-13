package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

func TestSystemV1Api(t *testing.T) {
	api := new(Systemv1)
	var requestsCount int
	t.Run("Health", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/system/health", nil)
		requestParameters := GetRequestParameters(request)
		response := httptest.NewRecorder()

		api.ApiController(response, requestParameters)
		var healthReply rest.HealthV1
		err := json.Unmarshal(response.Body.Bytes(), &healthReply)
		if err != nil {
			t.Error(err)
		}
		greetinWanted := rest.HealthV1{Status: "UP", Requests: requestsCount}

		if healthReply.Status != greetinWanted.Status {
			t.Errorf(".Status got %q, want %q", healthReply.Status, greetinWanted.Status)
		}
		if healthReply.Requests != greetinWanted.Requests {
			t.Errorf(".Requests got %q, want %q", healthReply.Requests, greetinWanted.Requests)
		}
	})
}
