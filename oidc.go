package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCProvider struct {
	Provider     *oidc.Provider
	Verifier     *oidc.IDTokenVerifier
	OAuth2Config *oauth2.Config
	Config       *ConfigOIDC
	sessions     sync.Map
}

type OIDCSession struct {
	Username string
	Email    string
	Name     string
	Expiry   time.Time
}

func randomState(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func InitOIDCProvider(config ConfigOIDC) (*OIDCProvider, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, config.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	oidcConfig := &OIDCProvider{
		Provider: provider,
		Verifier: verifier,
		Config:   &config,
	}

	oidcConfig.OAuth2Config = &oauth2.Config{
		Endpoint:     provider.Endpoint(),
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Scopes:       config.Scopes,
		RedirectURL:  config.RedirectURL,
	}

	logger.Info("OIDC provider initialized successfully",
		"function", "InitOIDCProvider", "provider", config.ProviderURL, "clientID", config.ClientID)
	return oidcConfig, nil
}

func (provider *OIDCProvider) VerifyJWT(tokenString string) (*oidc.IDToken, error) {
	token, err := provider.Verifier.Verify(context.Background(), tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}
	return token, nil
}

func GetUsernameFromToken(token *oidc.IDToken) string {
	var claims struct {
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		return "anonymous"
	}
	if claims.PreferredUsername != "" {
		return claims.PreferredUsername
	}
	if claims.Email != "" {
		return strings.Split(claims.Email, "@")[0]
	}
	return "anonymous"
}

func GetSessionFromToken(token *oidc.IDToken) (*OIDCSession, error) {
	var claims struct {
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	session := &OIDCSession{
		Username: GetUsernameFromToken(token),
		Email:    claims.Email,
		Name:     claims.Name,
		Expiry:   token.Expiry,
	}
	return session, nil
}

func StartAuthorization(provider *OIDCProvider) (string, string) {
	state, err := randomState(16)
	if err != nil {
		logger.Error("Failed to generate state", "error", err)
		return "", ""
	}
	authURL := provider.OAuth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("prompt", "login"))
	return authURL, state
}

func (provider *OIDCProvider) HandleCallback(code string, state string) (*oidc.IDToken, error) {
	oauth2Token, err := provider.OAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	idToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	token, err := provider.Verifier.Verify(context.Background(), idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}
	return token, nil
}

func BuildSessionCookie(tokenString string, cookieName string, ttlMinutes int) *http.Cookie {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    tokenString,
		Path:     "/",
		MaxAge:   ttlMinutes * 60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	return cookie
}

func ExtractSessionCookie(r *http.Request, cookieName string) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func ClearSessionCookie(w http.ResponseWriter, cookieName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (provider *OIDCProvider) OIDCLogin(w http.ResponseWriter, r *http.Request) {
	debugLogger := debugLogger.With("function", "OIDCLogin")
	debugLogger.Debug("OIDC login requested")

	authURL, state := StartAuthorization(provider)
	if authURL == "" {
		http.Error(w, "Failed to generate authorization URL", http.StatusInternalServerError)
		return
	}

	// Store state for CSRF protection
	stateKey := "oidc:" + state
	provider.sessions.Store(stateKey, time.Now().Add(10*time.Minute))

	http.SetCookie(w, &http.Cookie{
		Name:  "oidc_state",
		Value: state,
		Path:  "/",
		MaxAge: 600, // 10 minutes
	})

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (provider *OIDCProvider) OIDCCallback(w http.ResponseWriter, r *http.Request) {
	debugLogger := debugLogger.With("function", "OIDCCallback")
	debugLogger.Debug("OIDC callback received")

	state, err := r.Cookie("oidc_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	// Verify state to prevent CSRF
	stateKey := "oidc:" + state.Value
	storedTime, ok := provider.sessions.LoadAndDelete(stateKey)
	if !ok {
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}

	storedTimestamp, ok := storedTime.(time.Time)
	if !ok || storedTimestamp.Before(time.Now()) {
		http.Error(w, "Expired state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	oauth2Token, err := provider.OAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		logger.Error("Failed to exchange token", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	idToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		logger.Error("No id_token in token response")
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	token, err := provider.Verifier.Verify(context.Background(), idToken)
	if err != nil {
		logger.Error("Failed to verify ID token", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Get session info from token
	session, err := GetSessionFromToken(token)
	if err != nil {
		logger.Error("Failed to get session from token", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Store raw JWT in cookie
	cookie := BuildSessionCookie(idToken, provider.Config.CookieName, provider.Config.TokenTTL)
	if provider.Config.CookieDomain != "" {
		cookie.Domain = provider.Config.CookieDomain
	}
	http.SetCookie(w, cookie)

	debugLogger.Debug("OIDC login successful", "username", session.Username, "email", session.Email)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (provider *OIDCProvider) OIDCLogout(w http.ResponseWriter, r *http.Request) {
	debugLogger := debugLogger.With("function", "OIDCLogout")
	debugLogger.Debug("OIDC logout requested")

	ClearSessionCookie(w, provider.Config.CookieName)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (App *Application) OIDCSessionFromRequest(r *http.Request) (*OIDCSession, error) {
	if !App.Config.OIDC.Enabled || App.Auth.OIDCProvider == nil {
		return nil, fmt.Errorf("OIDC not enabled")
	}

	tokenString, err := ExtractSessionCookie(r, App.Config.OIDC.CookieName)
	if err != nil {
		return nil, err
	}

	token, err := App.Auth.OIDCProvider.VerifyJWT(tokenString)
	if err != nil {
		return nil, err
	}

	return GetSessionFromToken(token)
}

func (App *Application) WriteOIDCStatus(w http.ResponseWriter, request *RequestParameters) {
	debugLogger := request.Logger.Ext.With("function", "WriteOIDCStatus")

	if !App.Config.OIDC.Enabled {
		debugLogger.Debug("OIDC not enabled")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "OIDC not enabled",
		})
		return
	}

	session, err := App.OIDCSessionFromRequest(request.orgRequest)
	if err != nil {
		debugLogger.Debug("No valid OIDC session", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "No valid OIDC session",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username": session.Username,
		"email":    session.Email,
		"name":     session.Name,
		"expiry":   session.Expiry,
		"enabled":  App.Config.OIDC.Enabled,
	})
}

func initOIDCClientSecret(config *ConfigOIDC, envPrefix string) {
	if config.EnvVariableName != "" {
		envVarName := fmt.Sprintf("%v_%v", envPrefix, config.EnvVariableName)
		if val := os.Getenv(envVarName); val != "" {
			config.ClientSecret = val
		}
	}
}

func (provider *OIDCProvider) GetIPFromClaims(token *oidc.IDToken) (string, error) {
	var claims struct {
		IP string `json:"ip"`
	}
	if err := token.Claims(&claims); err != nil {
		return "", err
	}
	if claims.IP != "" {
		ip := net.ParseIP(claims.IP)
		if ip != nil {
			return claims.IP, nil
		}
	}
	return "", fmt.Errorf("no valid IP in token claims")
}

func (provider *OIDCProvider) GetTokenExpiry(token *oidc.IDToken) (time.Duration, error) {
	if time.Now().After(token.Expiry) {
		return 0, fmt.Errorf("token expired")
	}
	return time.Until(token.Expiry), nil
}

func (provider *OIDCProvider) CleanupExpiredSessions() {
	debugLogger := debugLogger.With("function", "CleanupExpiredSessions")
	debugLogger.Debug("Cleaning up expired OIDC sessions")

	now := time.Now()
	provider.sessions.Range(func(key, value interface{}) bool {
		if storedTime, ok := value.(time.Time); ok {
			if storedTime.Before(now) {
				provider.sessions.Delete(key)
			}
		}
		return true
	})
}
