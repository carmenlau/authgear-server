package config

var _ = Schema.Add("HTTPConfig", `
{
	"type": "object",
	"additionalProperties": false,
	"properties": {
		"public_origin": { "type": "string", "format": "http_origin" },
		"allowed_origins": { "type": "array", "items": { "type": "string", "minLength": 1 } },
		"cookie_prefix": { "type": "string" },
		"cookie_domain": { "type": "string" },
		"csp_directives": { "type": "array", "items": { "type": "string", "minLength": 1 } }
	},
	"required": [ "public_origin" ]
}
`)

type HTTPCSPDirectives []string

type HTTPConfig struct {
	PublicOrigin   string            `json:"public_origin"`
	AllowedOrigins []string          `json:"allowed_origins,omitempty"`
	CookiePrefix   string            `json:"cookie_prefix,omitempty"`
	CookieDomain   *string           `json:"cookie_domain,omitempty"`
	CSPDirectives  HTTPCSPDirectives `json:"csp_directives,omitempty"`
}

func (c *HTTPConfig) SetDefaults() {
	if len(c.CSPDirectives) == 0 {
		c.CSPDirectives = HTTPCSPDirectives{
			"default-src 'self'",
			"font-src 'self' cdnjs.cloudflare.com static2.sharepointonline.com",
			"style-src 'self' 'unsafe-inline' cdnjs.cloudflare.com",
			"object-src 'none'",
			"base-uri 'none'",
			"block-all-mixed-content",
		}
	}
}
