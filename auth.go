package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	b64 "encoding/base64"
	"fmt"
	"log/slog"
	"net"
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
	debugLogger := request.Logger.Ext.With("function", "Authentication")
	debugLogger.Debug(fmt.Sprintf("User in database : %v", ok), "found", ok)
	if ok {
		request.Authentication.User = &user
		if request.Basic.Ok {
			passwordHash := AuthHash(request.Basic.Password)
			if AuthTest(passwordHash, user.PasswordEnc) {
				request.Authentication.Verified.Password = true
			}
			request.RequestIP, _ = Auth.GetIPHeaderFromRequest(request)
			if user.HostAllowed(request.RequestIP, request.Logger.Ext) {
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
	debugLogger := request.Logger.Ext.With("function", "Autorization")
	if request.Api == "system" {
		debugLogger.Debug("Skipping Auth for system api")
		return true
	}
	userPermissions, ok := User.Permissions[request.Namespace]
	debugLogger.Debug("Testing User Permissions",
		"userPermissions", userPermissions, "expectedPermissions", permissions,
		"gobal", !ok)
	if ok {
		return AuthTestPermission(userPermissions, *permissions)
	} else {
		return AuthTestPermission(User.GlobalPermissions, *permissions)
	}
}

func (Auth *Auth) GetIPHeaderFromRequest(request *RequestParameters) (string, string) {
	debugLogger := request.Logger.Ext.With("function", "GetIPHeaderFromRequest")
	address, _ := remoteaddr.Parse().IP(request.orgRequest)
	foundHeaderName := "remoteaddr"

	debugLogger.Debug("Remote address: " + address)
	var trustedProxy bool
	for _, proxyAddress := range Auth.Config.TrustedProxies {
		if proxyAddress == address {
			debugLogger.Debug("Remote address is a trusted proxy")
			trustedProxy = true
			break
		}
	}
	if trustedProxy {
		for _, headerName := range Auth.HostHeaders {
			header := request.orgRequest.Header[headerName]
			if len(header) > 0 {
				debugLogger.Debug(fmt.Sprintf("Found address in headder [%v] = %v", headerName, header[0]))
				request.Logger.Ext = request.Logger.Ext.With("address", header[0], "proxy", address, "proxy-header", headerName)
				request.Logger.Log = request.Logger.Log.With("address", header[0], "proxy", address)
				return header[0], headerName
			}
		}
	} else {
		if len(Auth.Config.TrustedProxies) > 0 {
			logger.Debug("Remote address is not a trusted proxy")
		}
	}
	request.Logger.Ext = request.Logger.Ext.With("address", address)
	request.Logger.Log = request.Logger.Log.With("address", address)
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

func (User *User) HostAllowed(address string, dLogger *slog.Logger) bool {
	debugLogger := dLogger.With("function", "HostAllowed", "struct", "User")

	ip := net.ParseIP(address)
	for _, host := range User.Hosts {
		_, testSubnet, _ := net.ParseCIDR(host)
		if testSubnet != nil {
			if testSubnet.Contains(ip) {
				debugLogger.Debug("matched CIDR for for host: " + host)
				return true
			}
		} else {
			testIP := net.ParseIP(host)
			if testIP != nil {
				if ip.Equal(testIP) {
					debugLogger.Debug("matched IP for for host: " + host)
					return true
				}
			} else {
				testIPs, err := net.LookupIP(host)
				if err != nil {
					debugLogger.Error("Failed to parse address for DNS: "+host, "error", err)
					continue
				}
				for _, testIP := range testIPs {
					debugLogger.Debug(fmt.Sprintf("DNS Lookup for for domain %s resolved to %s ", host, testIP.String()))
					if ip.Equal(testIP) {
						debugLogger.Debug(fmt.Sprintf("Matched DNS Lookup for host %s ", host))
						return true
					}
				}
			}
		}
	}
	debugLogger.Debug(address + " not matched to any of users hosts")
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
