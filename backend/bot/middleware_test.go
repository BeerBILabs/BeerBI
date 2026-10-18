package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestAuthMiddleware(t *testing.T) {
	const token = "s3cret-test-token"
	cases := []struct {
		name   string
		header string
		set    bool
		want   int
	}{
		{name: "missing header", want: http.StatusUnauthorized},
		{name: "empty header", set: true, header: "", want: http.StatusUnauthorized},
		{name: "wrong scheme", set: true, header: "Basic " + token, want: http.StatusUnauthorized},
		{name: "lowercase scheme", set: true, header: "bearer " + token, want: http.StatusUnauthorized},
		{name: "no token after scheme", set: true, header: "Bearer", want: http.StatusUnauthorized},
		{name: "token without scheme", set: true, header: token, want: http.StatusUnauthorized},
		{name: "extra part", set: true, header: "Bearer " + token + " extra", want: http.StatusUnauthorized},
		{name: "double space", set: true, header: "Bearer  " + token, want: http.StatusUnauthorized},
		{name: "wrong token", set: true, header: "Bearer wrong-token", want: http.StatusUnauthorized},
		{name: "prefix of token", set: true, header: "Bearer " + token[:5], want: http.StatusUnauthorized},
		{name: "token with suffix", set: true, header: "Bearer " + token + "x", want: http.StatusUnauthorized},
		{name: "correct token", set: true, header: "Bearer " + token, want: http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			h := authMiddleware(token, zerolog.Nop(), next)

			req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
			if tc.set {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d", rec.Code, tc.want)
			}
			if called != (tc.want == http.StatusNoContent) {
				t.Errorf("next handler called = %v for status %d", called, rec.Code)
			}
		})
	}
}

// An empty configured token must lock the API: "Bearer " (empty token) must not log in.
func TestAuthMiddlewareEmptyConfiguredToken(t *testing.T) {
	h := authMiddleware("", zerolog.Nop(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler reached although no API token is configured")
	}))

	for _, header := range []string{"Bearer ", "Bearer x", ""} {
		req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status = %d, want 401", header, rec.Code)
		}
	}
}
