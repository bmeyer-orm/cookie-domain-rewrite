package cookie_domain_rewriter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/bmeyer-orm/cookie-domain-rewrite"
)

func TestCookieDomainRewriter(t *testing.T) {
	tests := []struct {
		name           string
		config         *plugin.Config
		requestHost    string
		requestOrigin  string
		requestReferer string
		setCookie      string
		expectedCookie string
		shouldRewrite  bool
	}{
		{
			name: "rewrite from .local domain via Host header",
			config: &plugin.Config{
				MatchDomains: []string{"*.local"},
				Replacements: []plugin.DomainReplacement{
					{From: "oreilly.review", To: "oreilly.local"},
				},
			},
			requestHost:    "api.oreilly.local",
			setCookie:      "session=abc123; Domain=oreilly.review; Secure; HttpOnly",
			expectedCookie: "session=abc123; Domain=oreilly.local; Secure; HttpOnly",
			shouldRewrite:  true,
		},
		{
			name: "rewrite from .local domain via Origin header",
			config: &plugin.Config{
				MatchDomains: []string{"*.local"},
				Replacements: []plugin.DomainReplacement{
					{From: "oreilly.review", To: "oreilly.local"},
				},
			},
			requestHost:    "api.oreilly.review",
			requestOrigin:  "https://www.oreilly.local",
			setCookie:      "session=abc123; Domain=oreilly.review; Secure; HttpOnly",
			expectedCookie: "session=abc123; Domain=oreilly.local; Secure; HttpOnly",
			shouldRewrite:  true,
		},
		{
			name: "no rewrite from .review domain",
			config: &plugin.Config{
				MatchDomains: []string{"*.local"},
				Replacements: []plugin.DomainReplacement{
					{From: "oreilly.review", To: "oreilly.local"},
				},
			},
			requestHost:    "api.oreilly.review",
			requestOrigin:  "https://www.oreilly.review",
			setCookie:      "session=abc123; Domain=oreilly.review; Secure; HttpOnly",
			expectedCookie: "session=abc123; Domain=oreilly.review; Secure; HttpOnly",
			shouldRewrite:  false,
		},
		{
			name: "rewrite lowercase domain attribute",
			config: &plugin.Config{
				MatchDomains: []string{"*.local"},
				Replacements: []plugin.DomainReplacement{
					{From: "oreilly.review", To: "oreilly.local"},
				},
			},
			requestHost:    "api.oreilly.local",
			setCookie:      "session=abc123; domain=oreilly.review; Secure; HttpOnly",
			expectedCookie: "session=abc123; domain=oreilly.local; Secure; HttpOnly",
			shouldRewrite:  true,
		},
		{
			name: "multiple replacements",
			config: &plugin.Config{
				MatchDomains: []string{"*.local"},
				Replacements: []plugin.DomainReplacement{
					{From: "oreilly.review", To: "oreilly.local"},
					{From: "example.review", To: "example.local"},
				},
			},
			requestHost:    "api.oreilly.local",
			setCookie:      "session=abc123; Domain=example.review; Secure",
			expectedCookie: "session=abc123; Domain=example.local; Secure",
			shouldRewrite:  true,
		},
		{
			name: "exact domain match",
			config: &plugin.Config{
				MatchDomains: []string{"api.oreilly.local"},
				Replacements: []plugin.DomainReplacement{
					{From: "oreilly.review", To: "oreilly.local"},
				},
			},
			requestHost:    "api.oreilly.local",
			setCookie:      "session=abc123; Domain=oreilly.review; Secure",
			expectedCookie: "session=abc123; Domain=oreilly.local; Secure",
			shouldRewrite:  true,
		},
		{
			name: "no match for exact domain",
			config: &plugin.Config{
				MatchDomains: []string{"api.oreilly.local"},
				Replacements: []plugin.DomainReplacement{
					{From: "oreilly.review", To: "oreilly.local"},
				},
			},
			requestHost:    "www.oreilly.local",
			setCookie:      "session=abc123; Domain=oreilly.review; Secure",
			expectedCookie: "session=abc123; Domain=oreilly.review; Secure",
			shouldRewrite:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock next handler that sets a cookie
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Set-Cookie", tt.setCookie)
				w.WriteHeader(http.StatusOK)
			})

			// Create the plugin
			handler, err := plugin.New(context.Background(), next, tt.config, "cookie-domain-rewriter")
			if err != nil {
				t.Fatalf("failed to create plugin: %v", err)
			}

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
			req.Host = tt.requestHost
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			if tt.requestReferer != "" {
				req.Header.Set("Referer", tt.requestReferer)
			}

			// Create a response recorder
			recorder := httptest.NewRecorder()

			// Execute the handler
			handler.ServeHTTP(recorder, req)

			// Check the Set-Cookie header
			gotCookie := recorder.Header().Get("Set-Cookie")
			if gotCookie != tt.expectedCookie {
				t.Errorf("Set-Cookie header mismatch:\ngot:  %s\nwant: %s", gotCookie, tt.expectedCookie)
			}
		})
	}
}

func TestMultipleSetCookieHeaders(t *testing.T) {
	config := &plugin.Config{
		MatchDomains: []string{"*.local"},
		Replacements: []plugin.DomainReplacement{
			{From: "oreilly.review", To: "oreilly.local"},
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "session=abc123; Domain=oreilly.review; Secure")
		w.Header().Add("Set-Cookie", "user=john; Domain=oreilly.review; HttpOnly")
		w.Header().Add("Set-Cookie", "tracking=xyz; Domain=other.com; Secure")
		w.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(context.Background(), next, config, "cookie-domain-rewriter")
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.Host = "api.oreilly.local"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	cookies := recorder.Header().Values("Set-Cookie")
	if len(cookies) != 3 {
		t.Fatalf("expected 3 Set-Cookie headers, got %d", len(cookies))
	}

	expected := []string{
		"session=abc123; Domain=oreilly.local; Secure",
		"user=john; Domain=oreilly.local; HttpOnly",
		"tracking=xyz; Domain=other.com; Secure",
	}

	for i, cookie := range cookies {
		if cookie != expected[i] {
			t.Errorf("Cookie %d mismatch:\ngot:  %s\nwant: %s", i, cookie, expected[i])
		}
	}
}
