package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	b64 "encoding/base64"
	"fmt"
	"log"
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

func (Auth *Auth) Authentication(r *RequestParameters) {
	user, ok := Auth.Users[r.Basic.Username]
	if ok {
		r.Authentication.User = &user
		if r.Basic.Ok {
			passwordHash := AuthHash(r.Basic.Password)
			if AuthTest(passwordHash, user.PasswordEnc) {
				r.Authentication.Verified.Password = true
			}
			r.RequestIP, _ = Auth.GetIPHeaderFromRequest(r.orgRequest)
			if user.HostAllowed(r.RequestIP) {
				r.Authentication.Verified.Host = true
			}
		}
	}
}

func (User *User) Autorization(r *RequestParameters, permissions ConfigPermissions) bool {
	if r.Api == "system" {
		return true
	}
	userPermissions, ok := User.Permissions[r.Namespace]
	if ok {
		return AuthTestPermission(userPermissions, permissions)
	} else {
		return AuthTestPermission(User.GlobalPermissions, permissions)
	}
}

func (Auth *Auth) ServeAuthFailed(w http.ResponseWriter, r *RequestParameters) {
	log.Printf("I %v %v %v %v %v", r.RequestIP, r.Basic.Username, r.Method, r.orgRequest.URL.Path, 403)
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
func (App *Application) BadRequestHandler(w http.ResponseWriter, r *RequestParameters) {
	log.Printf("I %v %v %v %v %v", r.RequestIP, r.GetUserName(), r.Method, r.orgRequest.URL.Path, 400)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 Bad Request"))
	return
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
	user := User{
		PasswordEnc: AuthDecode(Data.Password),
		Permissions: make(map[string]ConfigPermissions),
		Hosts:       Data.Hosts,
	}
	for _, permissionset := range Data.Permissionsset {
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
	return ((expected.Read == false || permission.Read) &&
		(expected.Write == false || permission.Write) &&
		(expected.List == false || permission.List))
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
