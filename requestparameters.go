package main

import (
	"net/http"
	"strings"
)

type RequestParameters struct {
	Basic struct {
		Username string
		Password string
		Ok       bool
	}
	Authentication struct {
		User     *User
		Verified struct {
			Password bool
			Host     bool
			mTLS     bool
		}
	}
	Method     string
	Api        string
	Namespace  string
	Secret     string
	orgRequest *http.Request
	RequestIP  string
}

func (RequestParameters *RequestParameters) GetUserName() string {
	if RequestParameters.Basic.Ok || RequestParameters.Authentication.Verified.mTLS {
		return RequestParameters.Basic.Username
	}
	return "anonymous"
}

func GetRequestParameters(r *http.Request) *RequestParameters {
	slashSeperated := strings.Split(r.URL.Path[1:], "/")
	req := &RequestParameters{Method: r.Method, orgRequest: r}
	if len(slashSeperated) > 0 {
		req.Api = slashSeperated[0]
	}
	if len(slashSeperated) > 1 {
		req.Secret = slashSeperated[1]
	}
	if len(slashSeperated) > 2 {
		req.Namespace = slashSeperated[2]
	}
	if r.TLS != nil && len(r.TLS.VerifiedChains) > 0 && len(r.TLS.VerifiedChains[0]) > 0 {
		req.Basic.Username = r.TLS.VerifiedChains[0][0].Subject.CommonName
		req.Basic.Password = ""
		req.Basic.Ok = true
		req.Authentication.Verified.mTLS = true
	} else {
		req.Basic.Username, req.Basic.Password, req.Basic.Ok = r.BasicAuth()
	}
	return req
}
