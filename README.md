# Cookie Domain Rewriter - Traefik Plugin

A Traefik middleware plugin that rewrites cookie domain attributes in `Set-Cookie` response headers based on the request's origin domain. Perfect for local development environments where you need to rewrite production domains to local equivalents.

## Use Case

When your staging servers use `.review` domains but your local development uses `.local` domains, cookies set by login servers with `Domain=my-domain.review` won't work locally. This plugin automatically rewrites them to `Domain=my-domain.local` when requests originate from `.local` domains.

## Features

- ✅ Conditionally rewrites cookie domains based on request origin
- ✅ Checks `Host`, `Origin`, and `Referer` headers to determine origin
- ✅ Supports wildcard domain matching (e.g., `*.local`)
- ✅ Handles both `Domain=` and `domain=` (case-insensitive)
- ✅ Processes multiple `Set-Cookie` headers
- ✅ Configurable domain replacement rules

## Installation

### Local Mode (Development)

For local development with Traefik, add the plugin to your `traefik.toml`:

```toml
[experimental.localPlugins.cookie-domain-rewrite]
  moduleName = "github.com/bmeyer-orm/cookie-domain-rewrite"
```

Place the plugin directory in your Traefik plugins directory (default: `./plugins-local/`).

### Plugin Catalog (Production)

1. Fork or create a GitHub repository for this plugin
2. Update the `import` path in `.traefik.yml` to match your repository
3. Add the plugin to your Traefik configuration:

```toml
[experimental.plugins.cookie-domain-rewrite]
  moduleName = "github.com/bmeyer-orm/cookie-domain-rewrite"
  version = "v1.0.0"
```

## Configuration

### Static Configuration (traefik.toml)

```toml
[experimental.plugins.cookie-domain-rewrite]
  moduleName = "github.com/bmeyer-orm/cookie-domain-rewrite"
  version = "v1.0.0"
```

### Dynamic Configuration

Define the middleware with your replacement rules:

```toml
[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite]
  matchDomains = ["*.my-domain.local", "api.my-domain.local"]
  
  [[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite.replacements]]
    from = "my-domain.review"
    to = "my-domain.local"
```

### Apply to Routes

```toml
[http.routers.my-router]
  rule = "Host(`api.my-domain.local`)"
  service = "my-service"
  middlewares = ["local-cookie-rewriter"]
```

### Docker Compose Example

```yaml
version: '3'

services:
  traefik:
    image: traefik:v2.10
    command:
      - --experimental.plugins.cookie-domain-rewrite.modulename=github.com/bmeyer-orm/cookie-domain-rewrite
      - --experimental.plugins.cookie-domain-rewrite.version=v1.0.0
    volumes:
      - ./traefik.toml:/traefik.toml
      - ./dynamic-config.toml:/dynamic-config.toml
    labels:
      - "traefik.http.middlewares.local-cookies.plugin.cookie-domain-rewrite.matchDomains=*.my-domain.local"
      - "traefik.http.middlewares.local-cookies.plugin.cookie-domain-rewrite.replacements[0].from=my-domain.review"
      - "traefik.http.middlewares.local-cookies.plugin.cookie-domain-rewrite.replacements[0].to=my-domain.local"
```

## Configuration Options

### `matchDomains` (array of strings)

List of domain patterns to match against the request origin. Supports wildcards (`*`).

**Examples:**
- `*.local` - Matches any subdomain ending in `.local`
- `api.my-domain.local` - Matches exactly this domain
- `*.my-domain.local` - Matches any subdomain of `my-domain.local`

### `replacements` (array of objects)

List of domain replacement rules to apply.

**Structure:**
```toml
[[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite.replacements]]
  from = "domain-to-replace.com"
  to = "replacement-domain.com"
```

## How It Works

1. **Origin Detection**: The plugin checks incoming request headers in this order:
   - `Host` header (includes `:authority` for HTTP/2)
   - `Origin` header
   - `Referer` header

2. **Pattern Matching**: If any header matches a configured `matchDomains` pattern, rewriting is enabled

3. **Cookie Rewriting**: All `Set-Cookie` headers in the response are processed:
   - `Domain=my-domain.review` → `Domain=my-domain.local`
   - `domain=my-domain.review` → `domain=my-domain.local`

4. **Pass-through**: If no match is found, the response passes through unmodified

## Example Scenarios

### Scenario 1: Local Development

**Request:**
```
GET /api/login HTTP/2
Host: api.my-domain.local
Origin: https://www.my-domain.local
```

**Original Response:**
```
HTTP/2 200 OK
Set-Cookie: session=abc123; Domain=my-domain.review; Secure; HttpOnly
```

**Modified Response:**
```
HTTP/2 200 OK
Set-Cookie: session=abc123; Domain=my-domain.local; Secure; HttpOnly
```

### Scenario 2: Staging (No Rewrite)

**Request:**
```
GET /api/login HTTP/2
Host: api.my-domain.review
Origin: https://www.my-domain.review
```

**Response (unchanged):**
```
HTTP/2 200 OK
Set-Cookie: session=abc123; Domain=my-domain.review; Secure; HttpOnly
```

## Testing

You can test the plugin locally:

1. Set up Traefik with local plugins enabled
2. Configure a test service that sets cookies
3. Make requests with different `Host`/`Origin` headers
4. Verify cookie domains are rewritten correctly

### Test with curl:

```bash
# Request from .local domain (should rewrite)
curl -H "Host: api.my-domain.local" \
     -H "Origin: https://www.my-domain.local" \
     -v https://your-traefik-endpoint/login

# Request from .review domain (should not rewrite)
curl -H "Host: api.my-domain.review" \
     -H "Origin: https://www.my-domain.review" \
     -v https://your-traefik-endpoint/login
```

## Troubleshooting

### Plugin Not Loading

- Ensure `experimental.plugins` or `experimental.localPlugins` is configured
- Check Traefik logs for plugin loading errors
- Verify the module name matches your repository

### Cookies Not Being Rewritten

- Confirm the middleware is applied to your router
- Check that request headers (`Host`, `Origin`, or `Referer`) match your `matchDomains` patterns
- Verify the `from` domain in `replacements` matches the actual cookie domain

### Multiple Replacements

The plugin supports multiple replacement rules:

```toml
[[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite.replacements]]
  from = "my-domain.review"
  to = "my-domain.local"

[[http.middlewares.local-cookie-rewriter.plugin.cookie-domain-rewrite.replacements]]
  from = "staging.example.com"
  to = "local.example.com"
```

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or pull request.

## Author

Brian Meyer
