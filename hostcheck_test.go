package main

import (
	"fmt"
	"testing"
)

type HostCheckTest struct {
	App Application
}

func Test_testHosts(t *testing.T) {
	HCT := new(HostCheckTest)
	HCT.App.Config = ConfigType{Hosts: []ConfigHosts{
		{Address: "192.168.0.1", Permissions: ConfigPermissions{Read: true, Write: true, List: true}},
		{Address: "192.168.0.1/24", Permissions: ConfigPermissions{Read: true, Write: false, List: false}},
		{Address: "example.com", Permissions: ConfigPermissions{Read: true, Write: true, List: true}},
		{Address: "hello.world", Permissions: ConfigPermissions{Read: true, Write: true, List: true}},
	}}
	CPAll := ConfigPermissions{Read: true, Write: true, List: true}
	CPRead := ConfigPermissions{Read: true, Write: false, List: false}
	ip := "192.168.0.1"
	t.Run(fmt.Sprintf("normal ip %s in range", ip), func(t *testing.T) {
		allowed, _ := HCT.App.testhost(ip, &CPAll)
		if !allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	ip = "192.168.0.12"
	t.Run(fmt.Sprintf("CIDR ip %s in range \"pass\"", ip), func(t *testing.T) {
		allowed, _ := HCT.App.testhost(ip, &CPRead)
		if !allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	t.Run(fmt.Sprintf("CIDR ip %s with wrong permissions\"fail\"", ip), func(t *testing.T) {
		allowed, _ := HCT.App.testhost(ip, &CPAll)
		if allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	// example.com IP's based on nslookup example.com
	ip = "93.184.216.34"
	t.Run(fmt.Sprintf("DNS for ip %s", ip), func(t *testing.T) {
		allowed, _ := HCT.App.testhost(ip, &CPAll)
		if !allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	// Fail tests
	ip = "127.0.0.1"
	t.Run(fmt.Sprintf("normal ip not in list %s \"fail\"", ip), func(t *testing.T) {
		allowed, _ := HCT.App.testhost(ip, &CPAll)
		if allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	ip = "172.16.0.1"
	t.Run(fmt.Sprintf("normal ip not in list %s \"fail\"", ip), func(t *testing.T) {
		allowed, _ := HCT.App.testhost(ip, &CPAll)
		if allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
	ip = "hello.world"
	t.Run(fmt.Sprintf("normal ip %s \"fail\"", ip), func(t *testing.T) {
		allowed, _ := HCT.App.testhost(ip, &CPAll)
		if allowed {
			t.Errorf("failed for host %v", ip)
		}
	})
}
