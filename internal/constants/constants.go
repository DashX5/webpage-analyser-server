package constants

import "time"

// Environment constants
const (
	EnvAppEnv           = "APP_ENV"
	EnvDevelopment      = "dev"
)

// Server constants
const (
	DefaultServerPort    = 8080
	DefaultServerMode    = "debug"
	DefaultServerTimeout = 30 * time.Second
)

// Cache constants
const (
	DefaultCacheTTL        = 1 * time.Hour
	DefaultRedisPort       = 6379
	DefaultRedisDB        = 0
	DefaultRedisHost      = "redis"
	CacheConnectionTimeout = 5 * time.Second
)

// Analyzer constants
const (
	DefaultMaxLinks     = 100
	DefaultLinkTimeout  = 10 * time.Second
	DefaultInternalLinkTimeout = 3 * time.Second // Shorter timeout for internal links
	DefaultMaxWorkers   = 20
	DefaultMaxRedirects = 0
	DefaultLoginFormThreshold = 10 // Minimum score required to consider a form as login form
)

// RateLimit constants
const (
	DefaultRateLimitEnabled        = true
	DefaultRequestsPerMinute       = 60.0
	DefaultRateLimitBurstFactor    = 0.1 // 10% of rate
	DefaultRateLimitCleanupTimeout = 1 * time.Hour
)

// HTTP Status codes
const (
	StatusOK                  = 200
	StatusBadRequest         = 400
	StatusTooManyRequests    = 429
	StatusInternalServerError = 500
)

// Validation constants
const (
	MaxURLLength = 2048
)

// Metrics constants
const (
	MetricRequestDurationName     = "webpage_analyzer_request_duration_seconds"
	MetricRequestDurationHelp     = "Time (in seconds) spent processing webpage analysis requests"
	MetricCacheHitsName          = "webpage_analyzer_cache_hits_total"
	MetricCacheHitsHelp          = "Total number of cache hits"
	MetricCacheMissesName        = "webpage_analyzer_cache_misses_total"
	MetricCacheMissesHelp        = "Total number of cache misses"
	MetricLinkCheckDurationName  = "webpage_analyzer_link_check_duration_seconds"
	MetricLinkCheckDurationHelp  = "Time (in seconds) spent checking link accessibility"
)

// Response messages
const (
	ErrInvalidURL           = "invalid URL provided"
	ErrURLTooLong          = "URL exceeds maximum length"
	ErrRateLimitExceeded   = "rate limit exceeded, please try again later"
	ErrInternalServer      = "internal server error occurred"
	ErrAnalysisFailed      = "webpage analysis failed"
	ErrCacheUnavailable    = "cache service unavailable"
	MsgAnalysisInProgress  = "analysis in progress"
	MsgAnalysisComplete    = "analysis completed successfully"
)

// Template paths
const (
	IndexTemplatePath    = "web/templates/index.html"
	ErrorTemplatePath    = "web/templates/error.html"
	ResultTemplatePath   = "web/templates/result.html"
)

// Configuration paths
const (
	DefaultConfigDir     = "config-files"  
	DefaultConfigEnv     = EnvDevelopment  
	ConfigFileExtension  = ".yaml"
	ConfigFileType       = "yaml"     // Configuration file type for viper
)

// HTTP headers
const (
	HeaderContentType     = "Content-Type"
	HeaderAccept         = "Accept"
	HeaderAuthorization  = "Authorization"
	HeaderRateLimit      = "X-RateLimit-Limit"
	HeaderRateRemaining  = "X-RateLimit-Remaining"
	HeaderRateReset      = "X-RateLimit-Reset"
	HeaderCacheControl   = "Cache-Control"
	HeaderRequestID      = "X-Request-ID"
)

// HTML Version Detection constants
const (
	// HTML Version strings
	HTMLVersion5            = "HTML5"
	HTMLVersionXHTML11      = "XHTML 1.1"
	HTMLVersionXHTML10      = "XHTML 1.0"
	HTMLVersionXHTML10Strict     = "XHTML 1.0 Strict"
	HTMLVersionXHTML10Transitional = "XHTML 1.0 Transitional"
	HTMLVersionXHTML10Frameset   = "XHTML 1.0 Frameset"
	HTMLVersionHTML401      = "HTML 4.01"
	HTMLVersionHTML401Strict     = "HTML 4.01 Strict"
	HTMLVersionHTML401Transitional = "HTML 4.01 Transitional"
	HTMLVersionHTML401Frameset   = "HTML 4.01 Frameset"
	HTMLVersionHTML40       = "HTML 4.0"
	HTMLVersionHTML32       = "HTML 3.2"
	HTMLVersionHTML20       = "HTML 2.0"
	HTMLVersionXHTMLGeneric = "XHTML"
	HTMLVersionHTMLGeneric  = "HTML"
	HTMLVersionUnknown      = "Unknown DOCTYPE"
	
	// DOCTYPE keywords
	DOCTYPEKeywordHTML      = "HTML"
	DOCTYPEKeywordXHTML     = "XHTML"
	DOCTYPEKeywordStrict    = "STRICT"
	DOCTYPEKeywordTransitional = "TRANSITIONAL"
	DOCTYPEKeywordFrameset  = "FRAMESET"
	DOCTYPEKeywordHTML401   = "HTML 4.01"
	DOCTYPEKeywordHTML40    = "HTML 4.0"
	DOCTYPEKeywordHTML32    = "HTML 3.2"
	DOCTYPEKeywordHTML20    = "HTML 2.0"
	DOCTYPEKeywordXHTML11   = "XHTML 1.1"
	DOCTYPEKeywordXHTML10   = "XHTML 1.0"
)

// HTML Version Detection regex patterns
const (
	RegexXMLDeclaration     = `(?i)^\s*<\?xml[^>]*\?>\s*`
	RegexHTMLComment       = `(?i)^\s*<!--.*?-->\s*`
	RegexDOCTYPEExtraction = `(?i)^\s*<!DOCTYPE\s+[^>]*>`
	RegexHTML5DOCTYPE      = `^\s*<!DOCTYPE\s+HTML\s*>\s*$`
) 