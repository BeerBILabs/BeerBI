package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNewHTTPServerHasLimits(t *testing.T) {
	handler := http.NewServeMux()
	srv := newHTTPServer(":9999", handler)

	if srv.Addr != ":9999" || srv.Handler != handler {
		t.Errorf("addr/handler not set: %q %v", srv.Addr, srv.Handler)
	}
	checks := map[string]struct{ got, want time.Duration }{
		"ReadHeaderTimeout": {srv.ReadHeaderTimeout, 5 * time.Second},
		"ReadTimeout":       {srv.ReadTimeout, 15 * time.Second},
		"WriteTimeout":      {srv.WriteTimeout, 30 * time.Second},
		"IdleTimeout":       {srv.IdleTimeout, 120 * time.Second},
	}
	for name, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", name, c.got, c.want)
		}
	}
	if srv.MaxHeaderBytes <= 0 || srv.MaxHeaderBytes > 1<<20 {
		t.Errorf("MaxHeaderBytes = %d", srv.MaxHeaderBytes)
	}
	// The timeouts must be ordered so that a request header always arrives within the read timeout.
	if srv.ReadHeaderTimeout > srv.ReadTimeout || srv.ReadTimeout > srv.WriteTimeout {
		t.Errorf("timeouts out of order: %v %v %v", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout)
	}
}

func TestRouter(t *testing.T) {
	const token = "router-test-token"
	h, _ := newStatsTestHandlers(t)
	h.slackManager = &SlackConnectionManager{logger: zerolog.Nop()}
	router := newRouter(h, token, zerolog.Nop())

	do := func(method, target string, authed bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		if authed {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// Public endpoints.
	if rec := do(http.MethodGet, "/healthz", false); rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Errorf("/healthz = %d %q", rec.Code, rec.Body.String())
	}
	if rec := do(http.MethodGet, "/api/health", false); rec.Code != http.StatusOK {
		t.Errorf("/api/health = %d", rec.Code)
	}
	if rec := do(http.MethodGet, "/metrics", false); rec.Code != http.StatusOK {
		t.Errorf("/metrics = %d", rec.Code)
	}

	// Protected endpoints: 401 without a token, the handler's answer with one.
	protected := []struct {
		target     string
		wantAuthed int
	}{
		{"/api/given", http.StatusBadRequest},
		{"/api/received", 400},
		{"/api/user", 400},
		{"/api/users", 400},
		{"/api/stats/combined", 400},
		{"/api/stats/timeline", 400},
		{"/api/stats/quarterly", 200},
		{"/api/stats/top", 400},
		{"/api/stats/heatmap", 400},
		{"/api/stats/pairs", 400},
	}
	for _, p := range protected {
		if rec := do(http.MethodGet, p.target, false); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without token = %d, want 401", p.target, rec.Code)
		}
		if rec := do(http.MethodGet, p.target, true); rec.Code != p.wantAuthed {
			t.Errorf("%s with token = %d, want %d", p.target, rec.Code, p.wantAuthed)
		}
	}

	// B12: the unauthenticated, unbounded list endpoints are gone, with and without a token.
	for _, target := range []string{"/api/givers", "/api/recipients"} {
		for _, authed := range []bool{false, true} {
			if rec := do(http.MethodGet, target, authed); rec.Code != http.StatusNotFound {
				t.Errorf("%s (token %v) = %d, want 404", target, authed, rec.Code)
			}
		}
	}
	if rec := do(http.MethodGet, "/api/nope", true); rec.Code != http.StatusNotFound {
		t.Errorf("unknown route = %d, want 404", rec.Code)
	}
}
