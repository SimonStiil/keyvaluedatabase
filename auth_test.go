package main

import (
	"fmt"
	"testing"
)

type HostCheckTest struct {
	User User
}

func Test_Auth(t *testing.T) {
	setupTestlogging()
	t.Run("Auth Hash And Encoding", func(t *testing.T) {
		data := "hello"
		expected := "LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ="
		hash := AuthHash(data)
		encodedHash := AuthEncode(hash)
		if encodedHash != expected {
			t.Errorf("Encoding failed got %v, expected %v", encodedHash, expected)
		}
	})
	t.Run("Auth Decoding And Test", func(t *testing.T) {
		expectedData := "hello"
		base := "LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ="
		expected := AuthHash(expectedData)
		hash := AuthDecode(base)
		result := AuthTest(hash, expected)
		if !result {
			t.Errorf("Decoding Test failed got %v, expected %v", expected, hash)
		}
	})
	t.Run("Load / Test Example User", func(t *testing.T) {
		Auth := new(Auth)
		config := ConfigType{}
		ConfigRead("example-config", &config)

		Auth.Init(config)
		ExampleUsername := "user"
		ExamplePassword := "password"
		user, ok := Auth.Users[ExampleUsername]
		if !ok {
			t.Errorf("unable to read user %v - %v", ExampleUsername, user)
		}
		ok = AuthTest(user.PasswordEnc, AuthHash(ExamplePassword))
		if !ok {
			t.Errorf("Password did not match for %v / %v", ExampleUsername, ExamplePassword)
		}
	})

	t.Run("Test Permissions", func(t *testing.T) {
		none := ConfigPermissions{}
		write := ConfigPermissions{Write: true}
		read := ConfigPermissions{Read: true}
		list := ConfigPermissions{List: true}
		rw := ConfigPermissions{Write: true, Read: true}
		rl := ConfigPermissions{Read: true, List: true}
		all := ConfigPermissions{Read: true, List: true, Write: true}

		permissions := map[string]ConfigPermissions{"none": none, "write": write, "read": read, "list": list, "rw": rw, "rl": rl, "all": all}
		truthTable := map[string]map[string]bool{
			"none":  {"none": true, "write": true, "read": true, "list": true, "rw": true, "rl": true, "all": true},
			"write": {"none": false, "write": true, "read": false, "list": false, "rw": true, "rl": false, "all": true},
			"read":  {"none": false, "write": false, "read": true, "list": false, "rw": true, "rl": true, "all": true},
			"list":  {"none": false, "write": false, "read": false, "list": true, "rw": false, "rl": true, "all": true},
			"rw":    {"none": false, "write": false, "read": false, "list": false, "rw": true, "rl": false, "all": true},
			"rl":    {"none": false, "write": false, "read": false, "list": false, "rw": false, "rl": true, "all": true},
			"all":   {"none": false, "write": false, "read": false, "list": false, "rw": false, "rl": false, "all": true},
		}
		for expectedKey, test := range truthTable {
			for testKey, testResult := range test {
				if AuthTestPermission(permissions[testKey], permissions[expectedKey]) != testResult {
					t.Errorf("Permission %v expecting %v supposed to %v", testKey, expectedKey, testResult)
				}
			}
		}
	})

	HCT := new(HostCheckTest)
	HCT.User.Hosts = []string{
		"192.168.0.1",
		"192.168.0.1/24",
		"example.com",
		"hello.world",
	}

	ip := "192.168.0.1"
	t.Run(fmt.Sprintf("normal ip %s in range", ip), func(t *testing.T) {
		allowed := HCT.User.HostAllowed(ip, logger)
		if !allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	ip = "192.168.0.12"
	t.Run(fmt.Sprintf("CIDR ip %s in range \"pass\"", ip), func(t *testing.T) {
		allowed := HCT.User.HostAllowed(ip, logger)
		if !allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	// example.com IP's based on nslookup example.com
	ip = "23.215.0.138"
	t.Run(fmt.Sprintf("DNS for ip %s", ip), func(t *testing.T) {
		allowed := HCT.User.HostAllowed(ip, logger)
		if !allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	// Fail tests
	ip = "127.0.0.1"
	t.Run(fmt.Sprintf("normal ip not in list %s \"fail\"", ip), func(t *testing.T) {
		allowed := HCT.User.HostAllowed(ip, logger)
		if allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	ip = "172.16.0.1"
	t.Run(fmt.Sprintf("normal ip not in list %s \"fail\"", ip), func(t *testing.T) {
		allowed := HCT.User.HostAllowed(ip, logger)
		if allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	ip = "hello.world"
	t.Run(fmt.Sprintf("normal ip %s \"fail\"", ip), func(t *testing.T) {
		allowed := HCT.User.HostAllowed(ip, logger)
		if allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
}
