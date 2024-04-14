package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/SimonStiil/keyvaluedatabase/rest"
	"golang.org/x/exp/slices"
)

func TestApiV1(t *testing.T) {
	setupTestlogging()
	var requestsCount uint32 = 0
	App = new(Application)
	api := new(APIv1)
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
	okBody := "OK"
	testData := rest.KVPairV2{Key: "somekey", Namespace: "somenamespace", Value: "123"}
	t.Run("POST(json)", func(t *testing.T) {
		marshalled, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader(marshalled))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusCreated {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusCreated)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	t.Run("POST(www-form)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/v1/%v/%v", testData.Namespace, testData.Key),
			strings.NewReader(
				fmt.Sprintf("value=%v", testData.Value)))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusCreated {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusCreated)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	t.Run("POST(raw)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/v1/%v/%v", testData.Namespace, testData.Key),
			strings.NewReader(testData.Value))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusCreated {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusCreated)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	t.Run("PUT(raw)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/v1/%v/%v", testData.Namespace, testData.Key),
			strings.NewReader(testData.Value))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusCreated {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusCreated)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	t.Run("List", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/v1/%v", testData.Namespace),
			nil)
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusOK)
		}
		var listReply []string
		err := json.Unmarshal(response.Body.Bytes(), &listReply)
		if err != nil {
			t.Error(err)
		}
		//t.Log(listReply)

		if !slices.Contains(listReply, testData.Key) {
			t.Errorf("list should contain: %v", testData.Key)
		}
		if slices.Contains(listReply, "counter") {
			t.Errorf("list should not contain: %v", "counter")
		}
	})
	t.Run("Get", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/v1/%v/%v", testData.Namespace, testData.Key),
			nil)
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusOK)
		}
		var replyPair rest.KVPairV2
		err := json.Unmarshal(response.Body.Bytes(), &replyPair)
		if err != nil {
			t.Error(err)
		}

		if replyPair.Key != testData.Key {
			t.Errorf(".Key got %q, want %q", replyPair.Key, testData.Key)
		}
		if replyPair.Value != testData.Value {
			t.Errorf(".Value got %q, want %q", replyPair.Value, testData.Value)
		}
	})
	t.Run("Get (not-Existing)", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/v1/%v/fake", testData.Namespace),
			nil)
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusNotFound {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusNotFound)
		}
	})
}
