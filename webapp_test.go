package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGETGreeting(t *testing.T) {
	setupTestlogging()
	App = new(Application)
	stub := &APIStub{}
	t.Run("Initialize DB for Tests", func(t *testing.T) {
		fileName := "testdb.yaml"
		err := os.Remove(fileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		config := ConfigType{}
		ConfigRead("example-config", &config)
		App.Auth = Auth{}
		App.Auth.Init(config)
		App.DB = &YamlDatabase{DatabaseName: fileName}
		App.DB.Init()
		App.Count = &Counter{}
		App.Count.Init(App.DB)
		App.APIEndpoints = []API{stub}

	})
	okBody := "OK"
	remoteAddr := "127.0.0.1:434"
	ExampleUsername := "user"
	ExamplePassword := "password"
	namespace := "readall"
	t.Run("Get readall", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), namespace),
			nil)
		request.SetBasicAuth(ExampleUsername, ExamplePassword)
		request.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusOK)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	TestPassword := "testpassword"
	UnauthorizedBody := "Unauthorized\n"
	t.Run("GET readall wrong password", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), namespace),
			nil)
		request.SetBasicAuth(ExampleUsername, TestPassword)
		request.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusUnauthorized {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusUnauthorized)
		}
		if string(b) != UnauthorizedBody {
			t.Errorf(".Body got %q, want %q", string(b), UnauthorizedBody)
		}
	})
	TestUsername := "test"
	remoteAddr = "172.17.0.6:434"
	t.Run("GET readall wrong namespace", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), namespace),
			nil)
		request.SetBasicAuth(TestUsername, TestPassword)
		request.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusUnauthorized {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusUnauthorized)
		}
		if string(b) != UnauthorizedBody {
			t.Errorf(".Body got %q, want %q", string(b), UnauthorizedBody)
		}
	})
	namespace = "world"
	t.Run("GET world", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), namespace),
			nil)
		request.SetBasicAuth(TestUsername, TestPassword)
		request.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusOK)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	namespace = "world"
	t.Run("DELETE world denied", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), namespace),
			nil)
		request.SetBasicAuth(TestUsername, TestPassword)
		request.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusUnauthorized {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusUnauthorized)
		}
		if string(b) != UnauthorizedBody {
			t.Errorf(".Body got %q, want %q", string(b), UnauthorizedBody)
		}
	})
	remoteAddr = "127.0.0.1:434"
	t.Run("DELETE world allowed", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), namespace),
			nil)
		request.SetBasicAuth(ExampleUsername, ExamplePassword)
		request.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("Error Reading body %v", err)
		}
		//t.Logf("Body: %v, Status: %v", string(b), response.Code)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusOK)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
}

type APIStub struct{}

func (Api *APIStub) APIPrefix() string {
	return "stub"
}

func (api *APIStub) Permissions(request *RequestParameters) *ConfigPermissions {
	switch request.Namespace {
	case "readall":
		return &ConfigPermissions{List: true, Read: true}
	case "list":
		return &ConfigPermissions{List: true}
	case "world":
		switch request.Method {
		case "GET":
			return &ConfigPermissions{Read: true}
		case "POST", "PUT", "UPDATE", "PATCH", "DELETE":
			return &ConfigPermissions{Write: true}
		}
	}
	return &ConfigPermissions{Write: true, Read: true, List: true}
}

func (api *APIStub) ApiController(w http.ResponseWriter, request *RequestParameters) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
