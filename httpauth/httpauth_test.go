package httpauth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/O-C-R/auth/id"
)

type testInfoKey struct{}

func TestBasicAuthenticationHandler(t *testing.T) {
	const (
		realm    = "test"
		username = "username"
		password = "password"
	)

	handler := BasicAuthenticationHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), realm, NewSingleUserAuthenticator(username, password), nil)

	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := http.DefaultClient.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusUnauthorized {
		t.Error("server allowed unauthenticated request")
	}

	request, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("authenticated request failed with status %d", response.StatusCode)
	}
}

func TestBearerAuthenticationHandler(t *testing.T) {
	token, err := id.New()
	if err != nil {
		t.Fatal(err)
	}

	handler := BearerAuthenticationHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if _, ok := req.Context().Value(testInfoKey{}).(id.ID); !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
	}), NewSingleTokenAuthenticator(token), testInfoKey{})

	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := http.DefaultClient.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusUnauthorized {
		t.Error("server allowed unauthenticated request")
	}

	response, err = http.DefaultClient.Get(server.URL + "?access_token=" + url.QueryEscape(token.String()))
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("authenticated request failed with status %d", response.StatusCode)
	}

	response, err = http.PostForm(server.URL, url.Values{"access_token": {token.String()}})
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("authenticated request failed with status %d", response.StatusCode)
	}

	request, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("authorization", "Bearer "+token.String())
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("authenticated request failed with status %d", response.StatusCode)
	}
}

func TestAuthenticationFallbackHandler(t *testing.T) {
	const (
		realm    = "test"
		username = "username"
		password = "password"
	)

	handler := AuthenticationFallbackHandler(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("authentic", "1")
			w.WriteHeader(http.StatusOK)
		}),
		BasicAuthentication(realm, NewSingleUserAuthenticator(username, password), nil),
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("authentic", "0")
			w.WriteHeader(http.StatusOK)
		}))

	server := httptest.NewServer(handler)
	defer server.Close()

	request, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("authenticated request failed with status %d", response.StatusCode)
	}

	if response.Header.Get("authentic") != "1" {
		t.Error("authenticated request not served by the handler")
	}

	response, err = http.DefaultClient.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unauthenticated request failed with status %d", response.StatusCode)
	}

	if response.Header.Get("authentic") != "0" {
		t.Error("unauthenticated request not served by the fallback handler")
	}
}
