package cookie_domain_rewrite

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Config holds the plugin configuration
type Config struct {
	MatchDomains []string            `json:"matchDomains,omitempty"`
	Replacements []DomainReplacement `json:"replacements,omitempty"`
}

// DomainReplacement defines a domain substitution rule
type DomainReplacement struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// CreateConfig creates the default plugin configuration
func CreateConfig() *Config {
	return &Config{
		MatchDomains: []string{"*.local"},
		Replacements: []DomainReplacement{
			{
				From: "oreilly.review",
				To:   "oreilly.local",
			},
		},
	}
}

// CookieDomainRewriter is the middleware struct
type CookieDomainRewriter struct {
	next         http.Handler
	name         string
	matchDomains []*regexp.Regexp
	replacements []DomainReplacement
}

// New creates a new CookieDomainRewriter plugin
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Replacements) == 0 {
		return nil, fmt.Errorf("no replacements configured")
	}

	// Compile match domain patterns
	var matchers []*regexp.Regexp
	for _, pattern := range config.MatchDomains {
		// Convert wildcard pattern to regex
		regexPattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, `.*`)
		regexPattern = "^" + regexPattern + "$"
		
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid match domain pattern '%s': %w", pattern, err)
		}
		matchers = append(matchers, re)
	}

	return &CookieDomainRewriter{
		next:         next,
		name:         name,
		matchDomains: matchers,
		replacements: config.Replacements,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (c *CookieDomainRewriter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Check if request is from a matching domain
	shouldRewrite := c.shouldRewriteForRequest(req)

	if !shouldRewrite {
		// Pass through without modification
		c.next.ServeHTTP(rw, req)
		return
	}

	// Wrap the response writer to intercept Set-Cookie headers
	wrappedWriter := &responseWriter{
		ResponseWriter: rw,
		replacements:   c.replacements,
	}

	c.next.ServeHTTP(wrappedWriter, req)
}

// shouldRewriteForRequest checks if the request originates from a domain we should rewrite
func (c *CookieDomainRewriter) shouldRewriteForRequest(req *http.Request) bool {
	// Check Host header (includes :authority for HTTP/2)
	host := req.Host
	if host != "" {
		// Strip port if present
		if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
			host = host[:colonIdx]
		}
		
		if c.matchesDomain(host) {
			return true
		}
	}

	// Check Origin header as fallback
	origin := req.Header.Get("Origin")
	if origin != "" {
		// Extract hostname from Origin URL
		if strings.HasPrefix(origin, "http://") {
			origin = origin[7:]
		} else if strings.HasPrefix(origin, "https://") {
			origin = origin[8:]
		}
		
		if colonIdx := strings.Index(origin, ":"); colonIdx != -1 {
			origin = origin[:colonIdx]
		}
		if slashIdx := strings.Index(origin, "/"); slashIdx != -1 {
			origin = origin[:slashIdx]
		}
		
		if c.matchesDomain(origin) {
			return true
		}
	}

	// Check Referer header as final fallback
	referer := req.Header.Get("Referer")
	if referer != "" {
		// Extract hostname from Referer URL
		if strings.HasPrefix(referer, "http://") {
			referer = referer[7:]
		} else if strings.HasPrefix(referer, "https://") {
			referer = referer[8:]
		}
		
		if colonIdx := strings.Index(referer, ":"); colonIdx != -1 {
			referer = referer[:colonIdx]
		}
		if slashIdx := strings.Index(referer, "/"); slashIdx != -1 {
			referer = referer[:slashIdx]
		}
		
		if c.matchesDomain(referer) {
			return true
		}
	}

	return false
}

// matchesDomain checks if a hostname matches any of our configured patterns
func (c *CookieDomainRewriter) matchesDomain(hostname string) bool {
	for _, matcher := range c.matchDomains {
		if matcher.MatchString(hostname) {
			return true
		}
	}
	return false
}

// responseWriter wraps http.ResponseWriter to intercept Set-Cookie headers
type responseWriter struct {
	http.ResponseWriter
	replacements []DomainReplacement
	wroteHeader  bool
}

// WriteHeader intercepts the header write to modify Set-Cookie headers
func (r *responseWriter) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.rewriteCookieDomains()
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write ensures headers are written before body
func (r *responseWriter) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.rewriteCookieDomains()
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(b)
}

// rewriteCookieDomains modifies all Set-Cookie headers according to replacement rules
func (r *responseWriter) rewriteCookieDomains() {
	cookies := r.Header().Values("Set-Cookie")
	if len(cookies) == 0 {
		return
	}

	// Remove existing Set-Cookie headers
	r.Header().Del("Set-Cookie")

	// Process and re-add each cookie with domain replacement
	for _, cookie := range cookies {
		modified := cookie
		
		for _, replacement := range r.replacements {
			// Case-insensitive domain replacement
			// Handle "Domain=example.com" (capital D)
			modified = strings.ReplaceAll(modified,
				fmt.Sprintf("Domain=%s", replacement.From),
				fmt.Sprintf("Domain=%s", replacement.To))
			
			// Handle "domain=example.com" (lowercase d)
			modified = strings.ReplaceAll(modified,
				fmt.Sprintf("domain=%s", replacement.From),
				fmt.Sprintf("domain=%s", replacement.To))
		}
		
		r.Header().Add("Set-Cookie", modified)
	}
}
