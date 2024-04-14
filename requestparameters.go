package main

import (
	"net/http"
	"strings"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

type RequestParameters struct {
	Basic struct {
		Username string
		Password string
		Ok       bool
	}
	Authentication struct {
		User     *User
		Verified Verified
	}
	AttachmentPair   *rest.KVPairV2
	AttachmentUpdate *rest.KVUpdateV2
	Method           string
	Api              string
	Namespace        string
	Key              string
	orgRequest       *http.Request
	RequestIP        string
	ID               uint32
}
type Verified struct {
	Password bool
	Host     bool
	mTLS     bool
}

func (Verified *Verified) Ok() bool {
	if Verified.mTLS {
		return true
	}
	return Verified.Password && Verified.Host
}

func (RequestParameters *RequestParameters) GetUserName() string {
	if RequestParameters.Basic.Ok || RequestParameters.Authentication.Verified.mTLS {
		return RequestParameters.Basic.Username
	}
	return "anonymous"
}

func GetRequestParameters(r *http.Request, id uint32) *RequestParameters {
	slashSeperated := strings.Split(r.URL.Path[1:], "/")
	req := &RequestParameters{Method: r.Method, orgRequest: r, ID: id}
	if len(slashSeperated) > 0 {
		req.Api = slashSeperated[0]
	}
	if len(slashSeperated) > 1 {
		req.Namespace = slashSeperated[1]
	}
	if len(slashSeperated) > 2 {
		req.Key = slashSeperated[2]
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
