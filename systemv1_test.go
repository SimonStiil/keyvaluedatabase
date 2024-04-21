package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

func TestSystemV1Api(t *testing.T) {
	api := new(Systemv1)
	App = new(Application)
	setupTestlogging()
	t.Run("Initialize DB for Tests", func(t *testing.T) {
		fileName := "testdb.yaml"
		err := os.Remove(fileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		App.DB = &YamlDatabase{DatabaseName: fileName}
		App.DB.Init()
		App.Count = &Counter{}
		App.Count.Init(App.DB)
	})
	requestsCount := 0
	t.Run("Health", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/system/health", nil)
		requestParameters := GetRequestParameters(request, 0)
		response := httptest.NewRecorder()

		api.ApiController(response, requestParameters)
		if response.Result().StatusCode != 200 {
			t.Errorf(".StatusCode got %v, want %v", response.Result().StatusCode, 200)
		}
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
