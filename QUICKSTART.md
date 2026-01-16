# Quick Start Guide - Cookie Domain Rewriter

## Installation Steps

### 1. Get the Plugin

Clone or download this repository to your Traefik plugins directory:

```bash
# For local development
mkdir -p ./plugins-local
cd ./plugins-local
git clone https://github.com/bmeyer-orm/cookie-domain-rewrite.git

# OR place the plugin files directly in:
# ./plugins-local/cookie-domain-rewrite/
```

### 2. Configure Traefik Static Config

Add to your `traefik.toml`:

```toml
[experimental.localPlugins.cookie-domain-rewrite]
  moduleName = "github.com/bmeyer-orm/cookie-domain-rewrite"
```

### 3. Configure Dynamic Config

Create or update your dynamic configuration:

```toml
# dynamic-config.toml
[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite]
  matchDomains = ["*.oreilly.local"]
  
  [[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite.replacements]]
    from = "oreilly.review"
    to = "oreilly.local"

[http.routers.api-local]
  rule = "Host(`api.oreilly.local`)"
  service = "api-service"
  middlewares = ["local-cookie-rewriter"]

[http.services.api-service.loadBalancer]
  [[http.services.api-service.loadBalancer.servers]]
    url = "http://localhost:8080"
```

### 4. Start Traefik

```bash
traefik --configFile=traefik.toml
```

## Testing

Test with curl:

```bash
# Should rewrite cookies (from .local domain)
curl -H "Host: api.oreilly.local" \
     -v http://localhost/login

# Should NOT rewrite cookies (from .review domain)
curl -H "Host: api.oreilly.review" \
     -v http://localhost/login
```

Check the `Set-Cookie` headers in the response.

## Troubleshooting

1. **Plugin not loading?**
   - Check Traefik logs for errors
   - Verify the plugin directory structure
   - Ensure `go.mod` and `.traefik.yml` are present

2. **Cookies not rewriting?**
   - Verify middleware is applied to your router
   - Check that request `Host` header matches `matchDomains` pattern
   - Enable debug logging in Traefik

3. **Pattern not matching?**
   - Use `*.domain.com` for wildcard subdomain matching
   - Use exact domain for specific matches: `api.domain.com`

## Common Configurations

### Multiple domains:
```toml
[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite]
  matchDomains = ["*.oreilly.local", "*.safaribooks.local", "localhost"]
```

### Multiple replacements:
```toml
[[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite.replacements]]
  from = "oreilly.review"
  to = "oreilly.local"

[[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite.replacements]]
  from = "staging.example.com"
  to = "dev.example.com"
```

## Next Steps

- Push the plugin to GitHub
- Update `.traefik.yml` with your GitHub URL
- Tag a release (e.g., `v1.0.0`)
- Switch from `localPlugins` to `plugins` in production
