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
	URLPrefix := "/" + api.APIPrefix()
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
	okBody := "200 OK"
	createdBody := "201 Created"
	testKey := "somekey"
	testNamespace := "somenamespace"
	testData := rest.ObjectV1{Type: rest.TypeKey, Value: "123"}
	t.Run("POST(json)", func(t *testing.T) {
		marshalled, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testKey),
			bytes.NewReader(marshalled))
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
		if string(b) != createdBody {
			t.Errorf(".Body got %q, want %q", string(b), createdBody)
		}
		dbValue, dbErr := App.DB.Get(testNamespace, testKey)
		if dbErr != nil {
			t.Errorf("Error Reading from db %v", dbErr)
		}
		if testData.Value != dbValue {
			t.Errorf("data in database %v does not match posed value %v", dbValue, testData.Value)
		}
	})
	testKey = "somenewkey"
	testNamespace = "hello"
	t.Run("POST(www-form)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testKey),
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
		if string(b) != createdBody {
			t.Errorf(".Body got %q, want %q", string(b), createdBody)
		}
		dbValue, dbErr := App.DB.Get(testNamespace, testKey)
		if dbErr != nil {
			t.Errorf("Error Reading from db %v", dbErr)
		}
		if testData.Value != dbValue {
			t.Errorf("data in database %v does not match posed value %v", dbValue, testData.Value)
		}
	})
	testKey = "someotherkey"
	t.Run("POST(raw)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testKey),
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
		if string(b) != createdBody {
			t.Errorf(".Body got %q, want %q", string(b), createdBody)
		}
		dbValue, dbErr := App.DB.Get(testNamespace, testKey)
		if dbErr != nil {
			t.Errorf("Error Reading from db %v", dbErr)
		}
		if testData.Value != dbValue {
			t.Errorf("data in database %v does not match posed value %v", dbValue, testData.Value)
		}
	})
	testKey = "somedifferentkey"
	testNamespace = "somenamespace"
	t.Run("PUT(raw)", func(t *testing.T) {
		request, _ := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testKey),
			strings.NewReader(testData.Value))
		//request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
		if string(b) != createdBody {
			t.Errorf(".Body got %q, want %q", string(b), createdBody)
		}
		dbValue, dbErr := App.DB.Get(testNamespace, testKey)
		if dbErr != nil {
			t.Errorf("Error Reading from db %v", dbErr)
		}
		if testData.Value != dbValue {
			t.Errorf("data in database %v does not match posed value %v", dbValue, testData.Value)
		}

	})
	testData = rest.ObjectV1{Type: rest.TypeGenerate, Value: ""}
	testGeneratedKey := AuthGenerateRandomString(8)
	t.Run("UPDATE(Generate) New", func(t *testing.T) {
		marshalled, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(
			"UPDATE",
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testGeneratedKey),
			bytes.NewReader(marshalled))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusCreated {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusCreated)
		}
		var replyPair rest.KVPairV2
		err = json.Unmarshal(response.Body.Bytes(), &replyPair)
		if err != nil {
			t.Error(err)
		}
		if replyPair.Key != testGeneratedKey {
			t.Errorf(".Key got %q, want %q", replyPair.Key, testGeneratedKey)
		}
		dbValue, dbErr := App.DB.Get(testNamespace, testGeneratedKey)
		if dbErr != nil {
			t.Errorf("Error Reading from db %v", dbErr)
		}
		if dbValue == "" {
			t.Errorf("data in database not set")
		}
		if dbValue != replyPair.Value {
			t.Errorf("data in database %v not matching reply data %v", dbValue, replyPair.Value)
		}

	})
	t.Run("UPDATE(Generate) repeat", func(t *testing.T) {
		marshalled, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(
			"UPDATE",
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testGeneratedKey),
			bytes.NewReader(marshalled))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusBadRequest {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusBadRequest)
		}
	})
	testData = rest.ObjectV1{Type: rest.TypeRoll, Value: ""}
	t.Run("UPDATE(Roll) Repeat", func(t *testing.T) {
		marshalled, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(
			"UPDATE",
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testGeneratedKey),
			bytes.NewReader(marshalled))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusCreated {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusCreated)
		}
		var replyPair rest.KVPairV2
		err = json.Unmarshal(response.Body.Bytes(), &replyPair)
		if err != nil {
			t.Error(err)
		}
		if replyPair.Key != testGeneratedKey {
			t.Errorf(".Key got %q, want %q", replyPair.Key, testGeneratedKey)
		}
		dbValue, dbErr := App.DB.Get(testNamespace, testGeneratedKey)
		if dbErr != nil {
			t.Errorf("Error Reading from db %v", dbErr)
		}
		if dbValue == "" {
			t.Errorf("data in database not set")
		}
		if dbValue != replyPair.Value {
			t.Errorf("data in database %v not matching reply data %v", dbValue, replyPair.Value)
		}

	})
	testGeneratedKey = AuthGenerateRandomString(8)
	t.Run("UPDATE(Roll) New", func(t *testing.T) {
		marshalled, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(
			"UPDATE",
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testGeneratedKey),
			bytes.NewReader(marshalled))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusBadRequest {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusBadRequest)
		}
	})
	t.Run("List keys", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%v/%v", URLPrefix, testNamespace),
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

		if !slices.Contains(listReply, testKey) {
			t.Errorf("list should contain: %v", testKey)
		}
		if slices.Contains(listReply, "counter") {
			t.Errorf("list should not contain: %v", "counter")
		}
	})
	testData = rest.ObjectV1{Type: rest.TypeKey, Value: "123"}
	t.Run("Get", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testKey),
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

		if replyPair.Key != testKey {
			t.Errorf(".Key got %q, want %q", replyPair.Key, testKey)
		}
		if replyPair.Value != testData.Value {
			t.Errorf(".Value got %q, want %q", replyPair.Value, testData.Value)
		}
	})
	t.Run("ListAll keys", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%v/%v/*", URLPrefix, testNamespace),
			nil)
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusOK)
		}
		var listReply rest.KVPairListV1
		err := json.Unmarshal(response.Body.Bytes(), &listReply)
		if err != nil {
			t.Error(err)
		}
		foundTestKey := false
		foundSystemKey := false
		for _, data := range listReply {
			if data.Key == testKey {
				foundTestKey = true
			}
			if data.Key == "counter" {
				foundSystemKey = true
			}
		}
		if !foundTestKey {
			t.Errorf("list should contain: %v", testKey)
		}
		if foundSystemKey {
			t.Errorf("list should not contain: %v", "counter")
		}
	})
	t.Run("Get (not-Existing)", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%v/%v/fake", URLPrefix, testNamespace),
			nil)
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusNotFound {
			t.Errorf(".Code got %v, want %v", response.Code, http.StatusNotFound)
		}
	})
	testCreateNamespace := rest.ObjectV1{Type: rest.TypeNamespace, Value: "newNamespace"}
	t.Run("Create Namespace POST(json)", func(t *testing.T) {
		marshalled, err := json.Marshal(testCreateNamespace)
		if err != nil {
			t.Fatalf("impossible to marshall teacher: %s", err)
		}
		request, _ := http.NewRequest(http.MethodPost, URLPrefix, bytes.NewReader(marshalled))
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
		if string(b) != createdBody {
			t.Errorf(".Body got %q, want %q", string(b), createdBody)
		}
	})
	t.Run("List namespaces", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodGet, URLPrefix, nil)
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

		if !slices.Contains(listReply, testNamespace) {
			t.Errorf("list should contain: %v", testNamespace)
		}
		if !slices.Contains(listReply, testCreateNamespace.Value) {
			t.Errorf("list should contain: %v", testCreateNamespace.Value)
		}
		if slices.Contains(listReply, App.DB.GetSystemNS()) {
			t.Errorf("list should not contain: %v", App.DB.GetSystemNS())
		}
	})

	t.Run("Delete Namespace", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("%v/%v", URLPrefix, testCreateNamespace.Value),
			nil)
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
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusOK)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	t.Run("List namespaces Delete Test", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodGet, URLPrefix, nil)
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

		if slices.Contains(listReply, testCreateNamespace.Value) {
			t.Errorf("list should not contain: %v", testCreateNamespace.Value)
		}
	})
	config := ConfigType{}
	ConfigRead("example-config", &config)
	App.Auth.Init(config)
	TestUsername := "test"
	TestPassword := "testpassword"
	t.Run("List namespaces full", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, URLPrefix+"/*", nil)
		request.SetBasicAuth(TestUsername, TestPassword)
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		App.Auth.Authentication(requestParameters)
		t.Logf("%+v", requestParameters)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusOK)
		}
		var listReply rest.NamespaceListV1
		err := json.Unmarshal(response.Body.Bytes(), &listReply)
		if err != nil {
			t.Error(err)
		}
		foundTestNS := false
		foundSystemNS := false
		accessableNamespace := "hello"
		deniedNamespace := "somenamespace"
		for _, data := range listReply {
			if data.Name == accessableNamespace {
				foundTestNS = true
				if data.Access != true {
					t.Errorf(".Access for %v shoud be true", data.Name)
				}
				if data.Size != 2 {
					t.Errorf(".Size for %v shoud be 1", data.Name)
				}
			}
			if data.Name == deniedNamespace {
				foundTestNS = true
				if data.Access == true {
					t.Errorf(".Access for %v shoud be true", data.Name)
				}
				if data.Size != 3 {
					t.Errorf(".Size for %v shoud be 3 was %v", data.Name, data.Size)
				}
			}
			if data.Name == App.DB.GetSystemNS() {
				foundSystemNS = true
			}
		}
		if !foundTestNS {
			t.Errorf("list should contain: %v", testNamespace)
		}
		if foundSystemNS {
			t.Errorf("list should not contain: %v", App.DB.GetSystemNS())
		}
	})

	t.Run("Delete key", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testKey),
			nil)
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
		if response.Code != http.StatusOK {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusOK)
		}
		if string(b) != okBody {
			t.Errorf(".Body got %q, want %q", string(b), okBody)
		}
	})
	t.Run("Get - Post delete test", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%v/%v/%v", URLPrefix, testNamespace, testKey),
			nil)
		response := httptest.NewRecorder()
		requestParameters := GetRequestParameters(request, requestsCount)
		requestsCount += 1
		api.ApiController(response, requestParameters)
		if response.Code != http.StatusNotFound {
			t.Errorf(".Code got %q, want %q", response.Code, http.StatusNotFound)
			var replyPair rest.KVPairV2
			err := json.Unmarshal(response.Body.Bytes(), &replyPair)
			if err != nil {
				t.Error(err)
			}

			if replyPair.Key != testKey {
				t.Errorf(".Key got %q, want %q", replyPair.Key, testKey)
			}
			if replyPair.Value != testData.Value {
				t.Errorf(".Value got %q, want %q", replyPair.Value, testData.Value)
			}
		}

	})
}
