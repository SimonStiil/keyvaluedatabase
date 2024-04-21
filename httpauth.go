package main

/*
import (
	"log"
	"net/http"
)

type UserLoginError struct{}

func (e *UserLoginError) Error() string {
	return "httpauth: User unable to be authenticated"
}

type HostBlockedError struct{}

func (e *HostBlockedError) Error() string {
	return "httpauth: Host not allowed to login"
}

func (App *Application) BasicAuth(next http.HandlerFunc, permission *ConfigPermissions) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(App.Auth.Users) == 0 {
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
		logger.Debug("D BasicAuth for: "+GetFunctionName(next), "function", "BasicAuth")
		username, password, ok := r.BasicAuth()
		if ok {
			user, ok := App.Auth.Users[username]
			if ok {
				passwordHash := AuthHash(password)
				if AuthTest(passwordHash, user.PasswordEnc) {
					if AuthTestPermission(user.Permissions, *permission) {
						r.Header.Set("secret_remote_username", username)
						next.ServeHTTP(w, r)
						return
					}
				}
			}
		} else {
			username = "-"
		}
		log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), username, r.Method, r.URL.Path, 401, "BasicAuthCheckFailed")
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
*/
