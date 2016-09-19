package httpauth

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/O-C-R/auth/id"
)

var (
	basicAuthenticationSep = []byte{':'}
)

type AuthenticationFunc func(w http.ResponseWriter, req *http.Request) (*http.Request, bool, error)

func AuthenticationHandler(handler http.Handler, authenticationFunc AuthenticationFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authenticationReq, authentic, err := authenticationFunc(w, req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !authentic {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(w, authenticationReq)
	})
}

func AuthenticationFallbackHandler(handler http.Handler, authenticationFunc AuthenticationFunc, fallbackHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authenticationReq, authentic, err := authenticationFunc(w, req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !authentic {
			fallbackHandler.ServeHTTP(w, authenticationReq)
			return
		}

		handler.ServeHTTP(w, authenticationReq)
	})
}

type UserAuthenticator interface {
	AuthenticateUser(username, password string) (info interface{}, authentic bool, err error)
}

type SingleUserAuthenticator struct {
	username, password string
}

func NewSingleUserAuthenticator(username, password string) *SingleUserAuthenticator {
	return &SingleUserAuthenticator{
		username: username,
		password: password,
	}
}

func (s *SingleUserAuthenticator) AuthenticateUser(username, password string) (info interface{}, authentic bool, err error) {
	if username != s.username || password != s.password {
		return nil, false, nil
	}

	return username, true, nil
}

func BasicAuthentication(realm string, userAuthenticator UserAuthenticator, contextKey interface{}) AuthenticationFunc {
	authenticateHeader := "Basic realm=\"" + realm + "\""
	return func(w http.ResponseWriter, req *http.Request) (*http.Request, bool, error) {
		encodedUsernamePassword := ""
		if _, err := fmt.Sscanf(req.Header.Get("authorization"), "Basic %s", &encodedUsernamePassword); err != nil {
			w.Header().Set("www-authenticate", authenticateHeader)
			return req, false, nil
		}

		decodedUsernamePassword, err := base64.StdEncoding.DecodeString(encodedUsernamePassword)
		if err != nil {
			w.Header().Set("www-authenticate", authenticateHeader)
			return req, false, nil
		}

		usernamePassword := bytes.SplitN(decodedUsernamePassword, basicAuthenticationSep, 2)
		if len(usernamePassword) != 2 {
			return req, false, nil
		}

		username, password := string(usernamePassword[0]), string(usernamePassword[1])
		info, authentic, err := userAuthenticator.AuthenticateUser(username, password)
		if err != nil {
			w.Header().Set("www-authenticate", authenticateHeader)
			return req, false, err
		}

		if !authentic {
			return req, false, nil
		}

		if contextKey != nil {
			ctx := req.Context()
			ctx = context.WithValue(ctx, contextKey, info)
			req = req.WithContext(ctx)
		}

		return req, authentic, nil
	}
}

func BasicAuthenticationHandler(handler http.Handler, realm string, userAuthenticator UserAuthenticator, contextKey interface{}) http.Handler {
	return AuthenticationHandler(handler, BasicAuthentication(realm, userAuthenticator, contextKey))
}

type TokenAuthenticator interface {
	AuthenticateToken(id.ID) (info interface{}, authentic bool, err error)
}

type SingleTokenAuthenticator struct {
	id id.ID
}

func NewSingleTokenAuthenticator(id id.ID) *SingleTokenAuthenticator {
	return &SingleTokenAuthenticator{id}
}

func (s *SingleTokenAuthenticator) AuthenticateToken(id id.ID) (interface{}, bool, error) {
	if id != s.id {
		return nil, false, nil
	}

	return id, true, nil
}

func BearerAuthentication(tokenAuthenticator TokenAuthenticator, contextKey interface{}) AuthenticationFunc {
	return func(w http.ResponseWriter, req *http.Request) (*http.Request, bool, error) {
		tokenString := req.FormValue("access_token")
		if tokenString == "" {
			if _, err := fmt.Sscanf(req.Header.Get("authorization"), "Bearer %s", &tokenString); err != nil {
				return req, false, nil
			}
		}

		var token id.ID
		if err := token.UnmarshalText([]byte(tokenString)); err != nil {
			return req, false, nil
		}

		info, authentic, err := tokenAuthenticator.AuthenticateToken(token)
		if err != nil {
			return req, false, err
		}

		if !authentic {
			return req, false, nil
		}

		if contextKey != nil {
			ctx := req.Context()
			ctx = context.WithValue(ctx, contextKey, info)
			req = req.WithContext(ctx)
		}

		return req, true, nil
	}
}

func BearerAuthenticationHandler(handler http.Handler, tokenAuthenticator TokenAuthenticator, contextKey interface{}) http.Handler {
	return AuthenticationHandler(handler, BearerAuthentication(tokenAuthenticator, contextKey))
}

func TokenHeaderAuthentication(tokenAuthenticator TokenAuthenticator, contextKey interface{}, header string) AuthenticationFunc {
	return func(w http.ResponseWriter, req *http.Request) (*http.Request, bool, error) {
		var token id.ID
		if err := token.UnmarshalText([]byte(req.Header.Get(header))); err != nil {
			return req, false, nil
		}

		info, authentic, err := tokenAuthenticator.AuthenticateToken(token)
		if err != nil {
			return req, false, err
		}

		if !authentic {
			return req, false, nil
		}

		if contextKey != nil {
			ctx := req.Context()
			ctx = context.WithValue(ctx, contextKey, info)
			req = req.WithContext(ctx)
		}

		return req, true, nil
	}
}

func TokenHeaderAuthenticationHandler(handler http.Handler, tokenAuthenticator TokenAuthenticator, contextKey interface{}, header string) http.Handler {
	return AuthenticationHandler(handler, TokenHeaderAuthentication(tokenAuthenticator, contextKey, header))
}
