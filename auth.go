package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	b64 "encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/netinternet/remoteaddr"
)

type Auth struct {
	Users       map[string]User
	Config      ConfigType
	HostHeaders []string
	Permissions struct {
		ListFull *ConfigPermissions
		List     *ConfigPermissions
	}
}

// https://www.alexedwards.net/blog/basic-authentication-in-go
// https://medium.com/@matryer/the-http-handler-wrapper-technique-in-golang-updated-bc7fbcffa702
func (Auth *Auth) Authentication(request *RequestParameters) bool {
	user, ok := Auth.Users[request.Basic.Username]
	logger.Debug("Start",
		"function", "Authentication", "struct", "Auth",
		"id", request.ID, "basic.user", request.Basic.Username,
		"basic.ok", request.Basic.Ok, "users.ok", ok)
	if ok {
		request.Authentication.User = &user
		if request.Basic.Ok {
			passwordHash := AuthHash(request.Basic.Password)
			if AuthTest(passwordHash, user.PasswordEnc) {
				request.Authentication.Verified.Password = true
			}
			request.RequestIP, _ = Auth.GetIPHeaderFromRequest(request.orgRequest)
			if user.HostAllowed(request.RequestIP) {
				request.Authentication.Verified.Host = true
			}
			if request.Authentication.Verified.Password && request.Authentication.Verified.Host {
				return true
			}
			return request.Authentication.Verified.Ok()
		}
	}
	return false
}
func (Auth *Auth) NoAuth(permissions *ConfigPermissions) bool {
	return !permissions.List && !permissions.Read && !permissions.Write
}

func (User *User) Autorization(request *RequestParameters, permissions *ConfigPermissions) bool {
	logger.Debug("Start",
		"function", "Autorization", "struct", "Auth",
		"id", request.ID, "api", request.Api)
	if request.Api == "system" {
		return true
	}
	userPermissions, ok := User.Permissions[request.Namespace]
	logger.Debug("Read User Permissions",
		"function", "ServeAuthFailed", "struct", "Auth",
		"id", request.ID, "userPermissions", userPermissions, "expectedPermissions", permissions,
		"forNamespace", ok, "namespace", request.Namespace)
	if ok {
		return AuthTestPermission(userPermissions, *permissions)
	} else {
		return AuthTestPermission(User.GlobalPermissions, *permissions)
	}
}

func (Auth *Auth) ServeAuthFailed(w http.ResponseWriter, request *RequestParameters) {
	logger.Debug("Auth Failed",
		"function", "ServeAuthFailed", "struct", "Auth",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "status", 403)
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func (Auth *Auth) GetIPHeaderFromRequest(r *http.Request) (string, string) {
	address, _ := remoteaddr.Parse().IP(r)
	foundHeaderName := "remoteaddr"

	logger.Debug("Remote address: "+address, "function", "GetIPHeaderFromRequest", "struct", "Auth")
	var trustedProxy bool
	for _, proxyAddress := range Auth.Config.TrustedProxies {
		if proxyAddress == address {
			logger.Debug(address+" is a trusted proxy", "function", "GetIPHeaderFromRequest", "struct", "Auth")
			trustedProxy = true
			break
		}
	}
	if trustedProxy {
		for _, headerName := range Auth.HostHeaders {
			header := r.Header[headerName]
			if len(header) > 0 {
				logger.Debug(fmt.Sprintf("Found address in headder [%v] = %v", headerName, header[0]), "function", "GetIPHeaderFromRequest", "struct", "Auth")
				return header[0], headerName
			}
		}
	} else {
		logger.Debug(address+" is not a trusted proxy - skipping headders", "function", "GetIPHeaderFromRequest", "struct", "Auth")
	}
	return address, foundHeaderName
}

func AuthHash(data string) [32]byte {
	return sha256.Sum256([]byte(data))
}
func AuthTest(test [32]byte, fact [32]byte) bool {
	return (subtle.ConstantTimeCompare(test[:], fact[:]) == 1)
}
func AuthEncode(data [32]byte) string {
	return b64.StdEncoding.EncodeToString(data[:])
}
func AuthDecode(data string) [32]byte {
	bytes, err := b64.StdEncoding.DecodeString(data)

	if err != nil {
		panic(err)
	}
	if len(bytes) != 32 {
		panic(fmt.Sprintf("E wrong datalength %d should be 32", len(bytes)))
	}
	byteArray := [32]byte{}
	copy(byteArray[:], bytes)
	return byteArray
}
func (Auth *Auth) AuthGenerate(generate string, test string) {
	if generate != "" {
		hash := AuthHash(generate)
		encodedHash := AuthEncode(hash)
		logger.Debug("encodedHash: "+encodedHash, "function", "AuthGenerate", "struct", "Auth")
		if test == "" {
			fmt.Println(encodedHash)
		} else {
			testHash := AuthDecode(test)
			success := AuthTest(testHash, hash)
			fmt.Println("Test: ", success)
		}
		os.Exit(0)
	}
}
func (Auth *Auth) Init(config ConfigType) {
	Auth.Config = config
	Auth.Users = make(map[string]User)
	for _, v := range config.Users {
		Auth.Users[v.Username] = AuthUnpack(v)
	}
	logger.Debug(fmt.Sprintf("Auth.Users: %+v", Auth.Users), "function", "Init", "struct", "Auth")
	logger.Debug(fmt.Sprintf("Loaded %v users", len(Auth.Users)), "function", "Init", "struct", "Auth")
	Auth.HostHeaders = []string{
		"X-Forwarded-For",
		"HTTP_FORWARDED",
		"HTTP_FORWARDED_FOR",
		"HTTP_X_FORWARDED",
		"HTTP_X_FORWARDED_FOR",
		"HTTP_CLIENT_IP",
		"HTTP_VIA",
		"HTTP_X_CLUSTER_CLIENT_IP",
		"Proxy-Client-IP",
		"WL-Proxy-Client-IP",
		"REMOTE_ADDR"}
	Auth.Permissions.ListFull = &ConfigPermissions{List: true, Read: true}
	Auth.Permissions.List = &ConfigPermissions{List: true}
}

type User struct {
	PasswordEnc       [32]byte
	Permissions       map[string]ConfigPermissions
	GlobalPermissions ConfigPermissions
	Hosts             []string
}

func (User *User) HostAllowed(address string) bool {
	ip := net.ParseIP(address)
	for _, host := range User.Hosts {
		_, testSubnet, _ := net.ParseCIDR(host)
		if testSubnet != nil {
			if testSubnet.Contains(ip) {
				logger.Debug("matched CIDR for for host: "+host, "function", "HostAllowed", "struct", "User")
				return true
			}
		} else {
			testIP := net.ParseIP(host)
			if testIP != nil {
				if ip.Equal(testIP) {
					logger.Debug("matched IP for for host: "+host, "function", "HostAllowed", "struct", "User")
					return true
				}
			} else {
				testIPs, err := net.LookupIP(host)
				if err != nil {
					logger.Error("Failed to parse address for DNS: "+host, "function", "HostAllowed", "struct", "User", "error", err)
					continue
				}
				for _, testIP := range testIPs {
					logger.Debug(fmt.Sprintf("DNS Lookup for for domain %s resolved to %s ", host, testIP.String()), "function", "HostAllowed", "struct", "User")
					if ip.Equal(testIP) {
						logger.Debug(fmt.Sprintf("Matched DNS Lookup for host %s ", host), "function", "HostAllowed", "struct", "User")
						return true
					}
				}
			}
		}
	}
	logger.Debug(address+" not matched to any of users hosts", "function", "HostAllowed", "struct", "User")
	return false
}

func AuthUnpack(Data ConfigUser) User {

	logger.Debug("ReadingUser", "user", Data.Username, "function", "AuthUnpack")
	user := User{
		PasswordEnc: AuthDecode(Data.Password),
		Permissions: make(map[string]ConfigPermissions),
		Hosts:       Data.Hosts,
	}
	logger.Debug("Reading Permissionsset", "user", Data.Username, "function", "AuthUnpack", "size", len(Data.Permissionsset))
	for _, permissionset := range Data.Permissionsset {

		logger.Debug("Reading Permissionsset for Namespaces", "user", Data.Username, "function", "AuthUnpack", "size", len(permissionset.Namespaces), "namespaces", permissionset.Namespaces, "permissions", permissionset.Permissions)
		for _, namespace := range permissionset.Namespaces {
			if namespace == "*" {
				user.GlobalPermissions = permissionset.Permissions
			} else {
				user.Permissions[namespace] = permissionset.Permissions
			}
		}
	}
	return user
}

/*
	func AuthPack(Name string, Data User) ConfigUser {
		return ConfigUser{
			Username:    Name,
			Password:    AuthEncode(Data.PasswordEnc),
			Permissions: Data.Permissions,
		}
	}
*/
func AuthTestPermission(permission ConfigPermissions, expected ConfigPermissions) bool {
	return ((!expected.Read || permission.Read) &&
		(!expected.Write || permission.Write) &&
		(!expected.List || permission.List))
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func AuthGenerateRandomString(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	StringAsBytes := make([]byte, length)
	for i, b := range bytes {
		location := int(b) % len(charset)
		char := charset[location]
		StringAsBytes[i] = char
	}
	return string(StringAsBytes)
}
