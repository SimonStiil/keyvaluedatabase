package main

import (
	"log"
	"net"
	"net/http"
)

func (App *Application) HostBlocker(next http.HandlerFunc, permission *ConfigPermissions) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(App.Config.Hosts) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if permission == nil {
			switch r.Method {
			case "GET":
				permission = &ConfigPermissions{Read: true}
				break
			case "POST", "PUT", "UPDATE", "PATCH", "DELETE":
				permission = &ConfigPermissions{Write: true}
				break
			default:
				permission = &ConfigPermissions{Write: true, Read: true, List: true}
			}
		}
		address := r.Header.Get("secret_remote_address")
		foundHeadderName := r.Header.Get("secret_remote_header")
		allowed, found := App.testhost(address, permission)
		if allowed {
			next.ServeHTTP(w, r)
			return
		} else {
			if !found && App.Config.Debug {
				log.Println("D HostBlocker - ", foundHeadderName, " - Lookup Host failed for ", address)
			}
			log.Printf("I %v %v %v %v %v %v", address, "-", r.Method, r.URL.Path, 401, "HostCheckFailed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})
}

func (App *Application) testhost(address string, permission *ConfigPermissions) (bool, bool) {
	found := false
	ip := net.ParseIP(address)
	for _, host := range App.Config.Hosts {
		_, testSubnet, _ := net.ParseCIDR(host.Address)
		if testSubnet != nil {
			if testSubnet.Contains(ip) {
				found = true
				if AuthTestPermission(host.Permissions, *permission) {
					return true, true
				} else {
					if App.Config.Debug {
						log.Println("D HostBlocker(CIDR) - AuthTestPermission failed for ", host.Address)
					}
				}
			}
		} else {
			testIP := net.ParseIP(host.Address)
			if testIP != nil {
				if ip.Equal(testIP) {
					found = true
					if AuthTestPermission(host.Permissions, *permission) {
						return true, true
					} else {
						if App.Config.Debug {
							log.Println("D HostBlocker(IP) - AuthTestPermission failed for ", host.Address)
						}
					}
				}
			} else {
				testIPs, err := net.LookupIP(host.Address)
				if err != nil {
					log.Println("E HostBlocker(...) - unable to parse address as CIRD, IP or DNS:", host.Address)
					continue
				}
				for _, testIP := range testIPs {
					if App.Config.Debug {
						log.Printf("D HostBlocker(LookupIP) - domain %s resolved to %s ", host.Address, testIP.String())
					}
					if ip.Equal(testIP) {
						found = true
						if AuthTestPermission(host.Permissions, *permission) {
							return true, true
						} else {
							if App.Config.Debug {
								log.Println("D HostBlocker(LookupIP) - AuthTestPermission failed for ", host.Address)
							}
						}
					}
				}
			}
		}
	}
	if found && App.Config.Debug {
		log.Println("D HostBlocker(...) - Found host but did not have persmissions ")
	}
	return false, found
}
