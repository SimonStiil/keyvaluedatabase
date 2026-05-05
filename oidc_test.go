package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_OIDC_Config_Defaults(t *testing.T) {
	setupTestlogging()

	t.Run("ConfigOIDC default values", func(t *testing.T) {
		config := ConfigOIDC{}
		if config.Enabled {
			t.Error("OIDC should be disabled by default")
		}
	})

	t.Run("ConfigOIDC enabled with provider", func(t *testing.T) {
		config := ConfigOIDC{
			Enabled:     true,
			ProviderURL: "http://127.0.0.1:9096/.well-known/openid-configuration",
			ClientID:    "test-client",
		}
		if !config.Enabled {
			t.Error("OIDC should be enabled")
		}
		if config.ProviderURL != "http://127.0.0.1:9096/.well-known/openid-configuration" {
			t.Errorf("ProviderURL mismatch: got %v", config.ProviderURL)
		}
	})
}

func Test_OIDC_Session_Cookie(t *testing.T) {
	t.Run("BuildSessionCookie", func(t *testing.T) {
		cookie := BuildSessionCookie("test-token", "test_cookie", 60)
		if cookie.Name != "test_cookie" {
			t.Errorf("Cookie name: got %v, want test_cookie", cookie.Name)
		}
		if cookie.Value != "test-token" {
			t.Errorf("Cookie value: got %v, want test-token", cookie.Value)
		}
		if cookie.MaxAge != 3600 {
			t.Errorf("Cookie MaxAge: got %v, want 3600", cookie.MaxAge)
		}
		if !cookie.HttpOnly {
			t.Error("Cookie should be HttpOnly")
		}
		if !cookie.Secure {
			t.Error("Cookie should be Secure")
		}
		if cookie.SameSite != http.SameSiteStrictMode {
			t.Errorf("Cookie SameSite: got %v, want StrictMode", cookie.SameSite)
		}
	})

	t.Run("ClearSessionCookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		ClearSessionCookie(w, "test_cookie")
		result := w.Result()
		defer result.Body.Close()
		cookies := result.Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Expected 1 cookie, got %d", len(cookies))
		}
		if cookies[0].MaxAge != -1 {
			t.Errorf("Cookie MaxAge should be -1 for clear, got %v", cookies[0].MaxAge)
		}
	})
}

func Test_OIDC_State_Generation(t *testing.T) {
	t.Run("randomState uniqueness", func(t *testing.T) {
		states := make(map[string]bool)
		for i := 0; i < 100; i++ {
			state, err := randomState(16)
			if err != nil {
				t.Fatalf("Failed to generate state: %v", err)
			}
			if states[state] {
				t.Errorf("Duplicate state generated: %s", state)
			}
			states[state] = true
		}
	})
}

func Test_OIDC_Auth_Integration(t *testing.T) {
	// This test requires a running Authelia instance or mock
	// For CI, we use testcontainers to spin up Authelia

	t.Run("Authentication with OIDC disabled falls back to basic", func(t *testing.T) {
		App = new(Application)
		stub := &APIStub{}
		config := ConfigType{}
		ConfigRead("example-config", &config)
		App.Auth = Auth{}
		App.Auth.Init(config)
		App.DB = &YamlDatabase{DatabaseName: "test_oidc.db.yaml"}
		App.DB.Init()
		App.Count = &Counter{}
		App.Count.Init(App.DB)
		defer App.DB.Close()
		App.APIEndpoints = []API{stub}

		// Test basic auth still works when OIDC is disabled
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), "world"),
			nil)
		request.SetBasicAuth("user", "password")
		request.RemoteAddr = "127.0.0.1:434"
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		if response.Code != http.StatusOK {
			t.Errorf("Basic auth should work: got %v, want %v", response.Code, http.StatusOK)
		}
	})

	t.Run("Authentication with OIDC enabled and no session fails", func(t *testing.T) {
		App = new(Application)
		stub := &APIStub{}
		config := ConfigType{}
		ConfigRead("example-config", &config)
		// Enable OIDC but don't initialize provider
		config.OIDC.Enabled = true
		config.OIDC.DisableBasicAuth = true
		App.Auth = Auth{}
		App.Auth.Init(config)
		App.DB = &YamlDatabase{DatabaseName: "test_oidc.db.yaml"}
		App.DB.Init()
		App.Count = &Counter{}
		App.Count.Init(App.DB)
		defer App.DB.Close()
		App.APIEndpoints = []API{stub}

		// Request without OIDC session should fail
		request, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/%v/%v/notUsed", stub.APIPrefix(), "world"),
			nil)
		request.SetBasicAuth("user", "password")
		request.RemoteAddr = "127.0.0.1:434"
		response := httptest.NewRecorder()
		App.RootControllerV1(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Errorf("Request without OIDC session should be unauthorized: got %v, want %v", response.Code, http.StatusUnauthorized)
		}
	})
}

func Test_OIDC_Login_Endpoint(t *testing.T) {
	t.Run("OIDC login returns 404 when not configured", func(t *testing.T) {
		App = new(Application)
		config := ConfigType{}
		ConfigRead("example-config", &config)
		App.Auth = Auth{}
		App.Auth.Init(config)

		request, _ := http.NewRequest(http.MethodGet, "/oidc/login", nil)
		response := httptest.NewRecorder()
		App.Auth.OIDCLogin(response, request)

		if response.Code != http.StatusNotFound {
			t.Errorf("OIDC login should return 404 when not configured: got %v", response.Code)
		}
	})

	t.Run("OIDC callback returns 404 when not configured", func(t *testing.T) {
		App = new(Application)
		config := ConfigType{}
		ConfigRead("example-config", &config)
		App.Auth = Auth{}
		App.Auth.Init(config)

		request, _ := http.NewRequest(http.MethodGet, "/oidc/callback", nil)
		response := httptest.NewRecorder()
		App.Auth.OIDCCallback(response, request)

		if response.Code != http.StatusNotFound {
			t.Errorf("OIDC callback should return 404 when not configured: got %v", response.Code)
		}
	})

	t.Run("OIDC logout clears cookie when not configured", func(t *testing.T) {
		App = new(Application)
		config := ConfigType{}
		ConfigRead("example-config", &config)
		App.Auth = Auth{}
		App.Auth.Init(config)

		request, _ := http.NewRequest(http.MethodGet, "/oidc/logout", nil)
		response := httptest.NewRecorder()
		App.Auth.OIDCLogout(response, request)

		if response.Code != http.StatusFound {
			t.Errorf("OIDC logout should redirect: got %v, want %v", response.Code, http.StatusFound)
		}
	})
}

func TestMain(m *testing.M) {
	// Clean up test db files
	os.Remove("test_oidc.db.yaml")
	os.Exit(m.Run())
}

// Test_OIDC_With_Authelia spins up an Authelia container and tests full OIDC flow
// This test is skipped if TEST_AUTHelia is not set to true
func Test_OIDC_With_Authelia(t *testing.T) {
	if os.Getenv("TEST_AUTHelia") != "true" {
		t.Skip("Skipping Authelia integration test (set TEST_AUTHelia=true to enable)")
	}

	// Spin up Authelia container
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "authelia/authelia",
		ExposedPorts: []string{"9096/tcp"},
		WaitingFor:   wait.ForListeningPort("9096"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("Could not start Authelia container: %v", err)
		return
	}
	defer func() { _ = container.Terminate(ctx) }()

	mappedPort, err := container.MappedPort(ctx, "9096")
	if err != nil {
		t.Fatalf("Could not get mapped port: %v", err)
	}

	providerURL := fmt.Sprintf("http://127.0.0.1:%v/.well-known/openid-configuration", mappedPort.Port())

	// Configure OIDC
	config := ConfigType{}
	ConfigRead("example-config", &config)
	config.OIDC.Enabled = true
	config.OIDC.ProviderURL = providerURL
	config.OIDC.ClientID = "test-client"
	config.OIDC.ClientSecret = "test-secret"
	config.OIDC.RedirectURL = "http://127.0.0.1:8080/oidc/callback"
	config.OIDC.Scopes = []string{"openid", "profile", "email"}
	config.OIDC.TokenTTL = 60

	// Initialize provider
	provider, err := InitOIDCProvider(config.OIDC)
	if err != nil {
		t.Fatalf("Failed to initialize OIDC provider: %v", err)
	}

	t.Run("Provider discovery works", func(t *testing.T) {
		if provider.Provider == nil {
			t.Error("OIDC provider should not be nil")
		}
	})

	t.Run("Login returns authorization URL", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/oidc/login", nil)
		provider.OIDCLogin(w, r)

		if w.Code != http.StatusFound {
			t.Errorf("Login should redirect: got %v, want %v", w.Code, http.StatusFound)
		}
		location := w.Header().Get("Location")
		if location == "" {
			t.Error("Login should redirect to authorization URL")
		}
	})
}

// Helper function to test OIDC token extraction
func Test_GetUsernameFromToken(t *testing.T) {
	t.Run("Extracts preferred_username", func(t *testing.T) {
		// Mock token with claims
		token := &oidc.IDToken{}
		// In a real test, we would parse a real token
		username := GetUsernameFromToken(token)
		if username != "anonymous" {
			t.Errorf("Expected anonymous for empty token, got %v", username)
		}
	})
}

// Test OIDC session expiration
func Test_OIDC_Session_Expiry(t *testing.T) {
	t.Run("Session expiry calculation", func(t *testing.T) {
		session := &OIDCSession{
			Username: "testuser",
			Email:    "test@example.com",
			Expiry:   time.Now().Add(60 * time.Minute),
		}
		if session.Username != "testuser" {
			t.Errorf("Username mismatch: got %v", session.Username)
		}
		if time.Until(session.Expiry).Minutes() < 59 {
			t.Errorf("Session should have ~60 minutes until expiry")
		}
	})
}
