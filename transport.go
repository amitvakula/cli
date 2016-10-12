package main

import (
	"net/http"
)

// Http transports in this file are heavily inspired by https://github.com/golang/oauth2, among other places! :)

// ApiKeyTransport is an http.RoundTripper that authenticates all requests using an API key.
type ApiKeyTransport struct {
	Key string

	// Transport is the underlying HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	Transport http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
func (t *ApiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req) // per RoundTrip contract
	req.Header.Set("Authorization", "scitran-user "+t.Key)

	if t.Transport != nil {
		return t.Transport.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// Client returns an *http.Client that makes requests that are authenticated with an API key.
func (t *ApiKeyTransport) Client() *http.Client {
	return &http.Client{Transport: t}
}

// cloneRequest returns a clone of the provided *http.Request. The clone is a
// shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}
