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

var requestsCount int

func TestGETGreeting(t *testing.T) {
	app := new(Application)
	app.Config = ConfigType{Debug: true}
	t.Run("Initialize DB for Tests", func(t *testing.T) {
		fileName := "testdb.yaml"
		err := os.Remove(fileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		app.DB = &YamlDatabase{DatabaseName: fileName, Config: &ConfigType{Debug: true}}
		app.DB.Init()
		app.Count = &Counter{Config: &ConfigType{Debug: true}}
		app.Count.Init(app.DB)
	})
	t.Run("Greetings", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/system/greeting", nil)
		response := httptest.NewRecorder()

		app.GreetingController(response, request)
		requestsCount++
		var greetingReply rest.GreetingV1
		err := json.Unmarshal(response.Body.Bytes(), &greetingReply)
		if err != nil {
			t.Error(err)
		}
		greetinWanted := rest.GreetingV1{0, "Hello, World!"}

		if greetingReply.Id != greetinWanted.Id {
			t.Errorf(".id got %q, want %q", greetingReply.Id, greetinWanted.Id)
		}
		if greetingReply.Content != greetinWanted.Content {
			t.Errorf(".content got %q, want %q", greetingReply.Content, greetinWanted.Content)
		}
	})
	t.Run("Health", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/system/health", nil)
		response := httptest.NewRecorder()

		app.HealthActuator(response, request)
		var healthReply rest.HealthV1
		err := json.Unmarshal(response.Body.Bytes(), &healthReply)
		if err != nil {
			t.Error(err)
		}
		greetinWanted := rest.HealthV1{"UP", requestsCount}

		if healthReply.Status != greetinWanted.Status {
			t.Errorf(".Status got %q, want %q", healthReply.Status, greetinWanted.Status)
		}
		if healthReply.Requests != greetinWanted.Requests {
			t.Errorf(".Requests got %q, want %q", healthReply.Requests, greetinWanted.Requests)
		}
	})
	okBody := "OK"
	testData := rest.KVPairV1{Key: "somekey", Value: "123"}
	t.Run("POST(json)", func(t *testing.T) {
		marshalled, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader(marshalled))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		app.RootController(response, request)
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
			fmt.Sprintf("/%v", testData.Key),
			strings.NewReader(
				fmt.Sprintf("value=%v", testData.Value)))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()

		app.RootController(response, request)
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
	t.Run("POST(raw)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/%v", testData.Key),
			strings.NewReader(testData.Value))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()

		app.RootController(response, request)
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
	t.Run("PUT(raw)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/%v", testData.Key),
			strings.NewReader(testData.Value))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()

		app.RootController(response, request)
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
	t.Run("List", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/system/list", nil)
		response := httptest.NewRecorder()

		app.ListController(response, request)
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
		if !slices.Contains(listReply, "counter") {
			t.Errorf("list should contain: %v", "counter")
		}
	})
	t.Run("Get", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/"+testData.Key, nil)
		response := httptest.NewRecorder()

		app.RootController(response, request)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusOK)
		}
		var replyPair rest.KVPairV1
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
		request, _ := http.NewRequest(http.MethodGet, "/fake", nil)
		response := httptest.NewRecorder()

		app.RootController(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusNotFound)
		}
	})
}
