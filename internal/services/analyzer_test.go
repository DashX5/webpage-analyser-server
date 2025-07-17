package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/webpage-analyser-server/internal/config"
	"github.com/webpage-analyser-server/internal/constants"
	"github.com/webpage-analyser-server/internal/metrics"
	"github.com/webpage-analyser-server/internal/models"
)

// MockCache is a mock implementation of the CacheInterface
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, url string) (*models.AnalyzeResponse, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AnalyzeResponse), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, url string, result *models.AnalyzeResponse) error {
	args := m.Called(ctx, url, result)
	return args.Error(0)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockMetrics is a mock implementation of metrics to avoid Prometheus registration issues
type MockMetrics struct {
	RequestDuration   *prometheus.HistogramVec
	CacheHits        prometheus.Counter
	CacheMisses      prometheus.Counter
	LinkCheckDuration prometheus.Histogram
}

func NewMockMetrics() *metrics.Metrics {
	return &metrics.Metrics{
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "test_request_duration_seconds",
				Help: "Test metric",
			},
			[]string{"status"},
		),
		CacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_cache_hits_total",
				Help: "Test metric",
			},
		),
		CacheMisses: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_cache_misses_total",
				Help: "Test metric",
			},
		),
		LinkCheckDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name: "test_link_check_duration_seconds",
				Help: "Test metric",
			},
		),
	}
}


func createTestConfig() *config.Config {
	return &config.Config{
		Analyzer: config.AnalyzerConfig{
			MaxLinks:     constants.DefaultMaxLinks,
			LinkTimeout:  constants.DefaultLinkTimeout,
			MaxWorkers:   constants.DefaultMaxWorkers,
			MaxRedirects: constants.DefaultMaxRedirects,
		},
	}
}


func TestNewAnalyzer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()

	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	assert.NotNil(t, analyzer)
	assert.Equal(t, logger, analyzer.logger)
	assert.Equal(t, metrics, analyzer.metrics)
	assert.Equal(t, cache, analyzer.cache)
	assert.Equal(t, cfg, analyzer.config)
	assert.NotNil(t, analyzer.httpClient)
}


func TestNewAnalyzer_WithZeroConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := &config.Config{
		Analyzer: config.AnalyzerConfig{
			MaxLinks:     0,
			LinkTimeout:  0,
			MaxWorkers:   0,
			MaxRedirects: 0,
		},
	}

	NewAnalyzer(cfg, logger, metrics, cache)

	assert.Equal(t, constants.DefaultMaxLinks, cfg.Analyzer.MaxLinks)
	assert.Equal(t, constants.DefaultLinkTimeout, cfg.Analyzer.LinkTimeout)
	assert.Equal(t, constants.DefaultMaxWorkers, cfg.Analyzer.MaxWorkers)
	assert.Equal(t, constants.DefaultMaxRedirects, cfg.Analyzer.MaxRedirects)
}


func TestAnalyzer_DetectHTMLVersion(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello</h1></body>
</html>`

	version := analyzer.detectHTMLVersion(html)
	assert.Equal(t, "HTML5", version)
}


func TestAnalyzer_DetectLoginForm(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name: "Complete login form",
			html: `
			<form action="/login">
				<input type="text" name="username" />
				<input type="password" name="password" />
				<button type="submit">Login</button>
			</form>`,
			expected: true,
		},
		{
			name: "Login form with email",
			html: `
			<form action="/signin">
				<input type="email" name="email" />
				<input type="password" name="password" />
				<input type="submit" value="Sign In" />
			</form>`,
			expected: true,
		},
		{
			name: "Login form with remember me",
			html: `
			<form action="/auth">
				<input type="text" name="username" />
				<input type="password" name="password" />
				<input type="checkbox" name="remember" />
				<label>Remember me</label>
				<button type="submit">Log in</button>
			</form>`,
			expected: true,
		},
		{
			name: "Login form with forgot password",
			html: `
			<form action="/login">
				<input type="text" name="username" />
				<input type="password" name="password" />
				<button type="submit">Login</button>
				<a href="/forgot">Forgot password?</a>
			</form>`,
			expected: true,
		},
		{
			name: "Login form with OAuth",
			html: `
			<form action="/login">
				<input type="text" name="username" />
				<input type="password" name="password" />
				<button type="submit">Login</button>
				<button class="google-auth">Sign in with Google</button>
			</form>`,
			expected: true,
		},
		{
			name: "Regular form (not login)",
			html: `
			<form action="/contact">
				<input type="text" name="name" />
				<input type="email" name="email" />
				<textarea name="message"></textarea>
				<button type="submit">Send</button>
			</form>`,
			expected: false,
		},
		{
			name: "Form with only username (no password)",
			html: `
			<form action="/search">
				<input type="text" name="username" />
				<button type="submit">Search</button>
			</form>`,
			expected: false,
		},
		{
			name: "No form",
			html: `
			<div>
				<p>Welcome to our site</p>
			</div>`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			require.NoError(t, err)

			result := analyzer.detectLoginForm(doc)
			assert.Equal(t, tt.expected, result)
		})
	}
}


func TestAnalyzer_CheckLink(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	tests := []struct {
		name           string
		statusCode     int
		expectedResult bool
	}{
		{
			name:           "Accessible link (200)",
			statusCode:     http.StatusOK,
			expectedResult: true,
		},
		{
			name:           "Accessible link (301)",
			statusCode:     http.StatusMovedPermanently,
			expectedResult: true,
		},
		{
			name:           "Inaccessible link (404)",
			statusCode:     http.StatusNotFound,
			expectedResult: false,
		},
		{
			name:           "Inaccessible link (500)",
			statusCode:     http.StatusInternalServerError,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			ctx := context.Background()
			result := analyzer.checkLink(ctx, server.URL)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}


func TestAnalyzer_CheckLink_InvalidURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	ctx := context.Background()
	result := analyzer.checkLink(ctx, "invalid-url")
	assert.False(t, result)
}


func TestAnalyzer_AnalyzeLinks(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	cfg.Analyzer.MaxWorkers = 2 // Reduce for testing
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	// Create test servers
	accessibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer accessibleServer.Close()

	inaccessibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer inaccessibleServer.Close()

	html := `
	<html>
		<body>
			<a href="/internal1">Internal Link 1</a>
			<a href="/internal2">Internal Link 2</a>
			<a href="` + accessibleServer.URL + `">External Accessible</a>
			<a href="` + inaccessibleServer.URL + `">External Inaccessible</a>
		</body>
	</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	// Create base URL for testing
	baseURL, err := url.Parse("http://example.com")
	require.NoError(t, err)

	ctx := context.Background()
	result := analyzer.analyzeLinks(ctx, doc, baseURL)

	// Should have 2 internal links
	assert.Equal(t, 2, result.Internal)
	// Should have 2 external links
	assert.Equal(t, 2, result.External)
	// Should have 3 inaccessible links (2 internal links to non-existent http://example.com + 1 external returning 404)
	assert.Equal(t, 3, result.Inaccessible)
}


func TestAnalyzer_Analyze_CacheHit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	expectedResult := &models.AnalyzeResponse{
		URL:         "http://example.com",
		HTMLVersion: "HTML5",
		Title:       "Test Page",
		Headings:    map[string]int{"h1": 1},
		Links: models.LinkAnalysis{
			Internal:     2,
			External:     1,
			Inaccessible: 0,
		},
		HasLoginForm: false,
		AnalyzedAt:   time.Now(),
	}

	cache.On("Get", mock.Anything, "http://example.com").Return(expectedResult, nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "http://example.com")

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	cache.AssertExpectations(t)
}


func TestAnalyzer_Analyze_CacheMiss(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
	<h1>Welcome</h1>
	<h2>About</h2>
	<h2>Contact</h2>
	<a href="/page1">Internal Link</a>
	<a href="http://external.com">External Link</a>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	cache.On("Get", mock.Anything, server.URL).Return(nil, nil)
	cache.On("Set", mock.Anything, server.URL, mock.AnythingOfType("*models.AnalyzeResponse")).Return(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, server.URL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, server.URL, result.URL)
	assert.Equal(t, "HTML5", result.HTMLVersion)
	assert.Equal(t, "Test Page", result.Title)
	assert.Equal(t, 1, result.Headings["h1"])
	assert.Equal(t, 2, result.Headings["h2"])
	assert.Equal(t, 1, result.Links.Internal)
	assert.Equal(t, 1, result.Links.External)
	assert.False(t, result.HasLoginForm)
	cache.AssertExpectations(t)
}


func TestAnalyzer_Analyze_InvalidURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	cache.On("Get", mock.Anything, "invalid-url").Return(nil, nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "invalid-url")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid URL")
}


func TestAnalyzer_Analyze_HTTPError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	// Create server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cache.On("Get", mock.Anything, server.URL).Return(nil, nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, server.URL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status code 404")
}


func TestAnalyzer_Analyze_WithLoginForm(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	// Create test server with login form
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head><title>Login Page</title></head>
<body>
	<form action="/login">
		<input type="text" name="username" />
		<input type="password" name="password" />
		<button type="submit">Login</button>
	</form>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	cache.On("Get", mock.Anything, server.URL).Return(nil, nil)
	cache.On("Set", mock.Anything, server.URL, mock.AnythingOfType("*models.AnalyzeResponse")).Return(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, server.URL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasLoginForm)
	cache.AssertExpectations(t)
}


func TestAnalyzer_Analyze_MalformedHTML(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	// Create server with malformed HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid html content"))
	}))
	defer server.Close()

	cache.On("Get", mock.Anything, server.URL).Return(nil, nil)
	cache.On("Set", mock.Anything, server.URL, mock.AnythingOfType("*models.AnalyzeResponse")).Return(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, server.URL)

	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	cache.AssertExpectations(t)
}


func BenchmarkAnalyzer_Analyze(b *testing.B) {
	logger := zaptest.NewLogger(b)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
	<h1>Welcome</h1>
	<h2>About</h2>
	<a href="/page1">Internal Link</a>
	<a href="http://external.com">External Link</a>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	cache.On("Get", mock.Anything, server.URL).Return(nil, nil)
	cache.On("Set", mock.Anything, server.URL, mock.AnythingOfType("*models.AnalyzeResponse")).Return(nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.Analyze(ctx, server.URL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestAnalyzer_ParseAndValidateURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	tests := []struct {
		name      string
		url       string
		shouldErr bool
	}{
		{
			name:      "Valid HTTP URL",
			url:       "http://example.com",
			shouldErr: false,
		},
		{
			name:      "Valid HTTPS URL",
			url:       "https://example.com/path?param=value",
			shouldErr: false,
		},
		{
			name:      "Invalid URL - malformed",
			url:       "not-a-url",
			shouldErr: true,
		},
		{
			name:      "Invalid URL - with spaces",
			url:       "http://example .com",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL, err := analyzer.parseAndValidateURL(tt.url)
			
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Nil(t, parsedURL)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsedURL)
				assert.Equal(t, tt.url, parsedURL.String())
			}
		})
	}
}

func TestAnalyzer_FetchWebpage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	t.Run("Successful fetch", func(t *testing.T) {
		expectedHTML := "<html><body>Test content</body></html>"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedHTML))
		}))
		defer server.Close()

		content, err := analyzer.fetchWebpage(server.URL)
		assert.NoError(t, err)
		assert.Equal(t, expectedHTML, content)
	})

	t.Run("Server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		content, err := analyzer.fetchWebpage(server.URL)
		assert.Error(t, err)
		assert.Empty(t, content)
		assert.Contains(t, err.Error(), "status code 500")
	})

	t.Run("Invalid URL", func(t *testing.T) {
		content, err := analyzer.fetchWebpage("invalid-url")
		assert.Error(t, err)
		assert.Empty(t, content)
		assert.Contains(t, err.Error(), "failed to fetch webpage")
	})
}

func TestAnalyzer_ParseHTML(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	t.Run("Valid HTML", func(t *testing.T) {
		html := "<html><head><title>Test</title></head><body><h1>Hello</h1></body></html>"
		doc, err := analyzer.parseHTML(html)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "Test", doc.Find("title").Text())
	})

	t.Run("Malformed HTML", func(t *testing.T) {
		html := "<html><head><title>Test</title><body><h1>Hello</h1>"
		doc, err := analyzer.parseHTML(html)
		assert.NoError(t, err) // goquery is forgiving with malformed HTML
		assert.NotNil(t, doc)
	})

	t.Run("Empty HTML", func(t *testing.T) {
		html := ""
		doc, err := analyzer.parseHTML(html)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
	})
}

func TestAnalyzer_ExtractPageTitle(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	tests := []struct {
		name          string
		html          string
		expectedTitle string
	}{
		{
			name:          "Normal title",
			html:          "<html><head><title>Test Page</title></head></html>",
			expectedTitle: "Test Page",
		},
		{
			name:          "Title with extra whitespace",
			html:          "<html><head><title>  Test Page  </title></head></html>",
			expectedTitle: "Test Page",
		},
		{
			name:          "Empty title",
			html:          "<html><head><title></title></head></html>",
			expectedTitle: "",
		},
		{
			name:          "No title tag",
			html:          "<html><head></head></html>",
			expectedTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			require.NoError(t, err)

			title := analyzer.extractPageTitle(doc)
			assert.Equal(t, tt.expectedTitle, title)
		})
	}
}

func TestAnalyzer_CountHeadings(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	html := `
	<html>
		<body>
			<h1>Heading 1</h1>
			<h1>Another H1</h1>
			<h2>Heading 2</h2>
			<h3>Heading 3</h3>
			<h3>Another H3</h3>
			<h3>Yet Another H3</h3>
			<h6>Heading 6</h6>
		</body>
	</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	headings := analyzer.countHeadings(doc)

	expected := map[string]int{
		"h1": 2,
		"h2": 1,
		"h3": 3,
		"h4": 0,
		"h5": 0,
		"h6": 1,
	}

	assert.Equal(t, expected, headings)
}

func TestAnalyzer_ExtractDOCTYPE(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "HTML5 DOCTYPE",
			html:     "<!DOCTYPE html><html></html>",
			expected: "<!DOCTYPE HTML>",
		},
		{
			name:     "HTML 4.01 Strict DOCTYPE",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd"><html></html>`,
			expected: `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "HTTP://WWW.W3.ORG/TR/HTML4/STRICT.DTD">`,
		},
		{
			name:     "XHTML 1.0 DOCTYPE",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><html></html>`,
			expected: `<!DOCTYPE HTML PUBLIC "-//W3C//DTD XHTML 1.0 TRANSITIONAL//EN" "HTTP://WWW.W3.ORG/TR/XHTML1/DTD/XHTML1-TRANSITIONAL.DTD">`,
		},
		{
			name:     "With XML declaration",
			html:     `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE html><html></html>`,
			expected: "<!DOCTYPE HTML>",
		},
		{
			name:     "With HTML comments",
			html:     `<!-- This is a comment --><!DOCTYPE html><html></html>`,
			expected: "<!DOCTYPE HTML>",
		},
		{
			name:     "No DOCTYPE",
			html:     "<html></html>",
			expected: "",
		},
		{
			name:     "Whitespace before DOCTYPE",
			html:     "   \n\t<!DOCTYPE html><html></html>",
			expected: "<!DOCTYPE HTML>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractDOCTYPE(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzer_CheckHTMLVersionWithVariants(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	tests := []struct {
		name                  string
		doctype               string
		keyword               string
		baseVersion           string
		strictVersion         string
		transitionalVersion   string
		framesetVersion       string
		expected              string
	}{
		{
			name:                "HTML 4.01 Strict",
			doctype:             `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "HTTP://WWW.W3.ORG/TR/HTML4/STRICT.DTD">`,
			keyword:             "HTML 4.01",
			baseVersion:         "HTML 4.01",
			strictVersion:       "HTML 4.01 Strict",
			transitionalVersion: "HTML 4.01 Transitional",
			framesetVersion:     "HTML 4.01 Frameset",
			expected:            "HTML 4.01 Strict",
		},
		{
			name:                "HTML 4.01 Transitional",
			doctype:             `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 TRANSITIONAL//EN" "HTTP://WWW.W3.ORG/TR/HTML4/LOOSE.DTD">`,
			keyword:             "HTML 4.01",
			baseVersion:         "HTML 4.01",
			strictVersion:       "HTML 4.01 Strict",
			transitionalVersion: "HTML 4.01 Transitional",
			framesetVersion:     "HTML 4.01 Frameset",
			expected:            "HTML 4.01 Transitional",
		},
		{
			name:                "HTML 4.01 Frameset",
			doctype:             `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 FRAMESET//EN" "HTTP://WWW.W3.ORG/TR/HTML4/FRAMESET.DTD">`,
			keyword:             "HTML 4.01",
			baseVersion:         "HTML 4.01",
			strictVersion:       "HTML 4.01 Strict",
			transitionalVersion: "HTML 4.01 Transitional",
			framesetVersion:     "HTML 4.01 Frameset",
			expected:            "HTML 4.01 Frameset",
		},
		{
			name:        "No keyword match",
			doctype:     `<!DOCTYPE HTML>`,
			keyword:     "HTML 4.01",
			baseVersion: "HTML 4.01",
			expected:    "",
		},
		{
			name:        "Base version without variants",
			doctype:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN">`,
			keyword:     "HTML 4.01",
			baseVersion: "HTML 4.01",
			expected:    "HTML 4.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.checkHTMLVersionWithVariants(
				tt.doctype,
				tt.keyword,
				tt.baseVersion,
				tt.strictVersion,
				tt.transitionalVersion,
				tt.framesetVersion,
			)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzer_CheckLinkWithTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	// Create test servers
	accessibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer accessibleServer.Close()

	inaccessibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer inaccessibleServer.Close()

	tests := []struct {
		name       string
		url        string
		isInternal bool
		expected   bool
	}{
		{
			name:       "External accessible link",
			url:        accessibleServer.URL,
			isInternal: false,
			expected:   true,
		},
		{
			name:       "Internal accessible link",
			url:        accessibleServer.URL,
			isInternal: true,
			expected:   true,
		},
		{
			name:       "External inaccessible link",
			url:        inaccessibleServer.URL,
			isInternal: false,
			expected:   false,
		},
		{
			name:       "Internal inaccessible link",
			url:        inaccessibleServer.URL,
			isInternal: true,
			expected:   false,
		},
		{
			name:       "Invalid URL",
			url:        "invalid-url",
			isInternal: false,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := analyzer.checkLinkWithTimeout(ctx, tt.url, tt.isInternal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzer_DetectHTMLVersion_AdditionalCases(t *testing.T) {
	t.Skip("Ignoring this test")
	
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "HTML 4.01 Strict",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">`,
			expected: "HTML 4.01 Strict",
		},
		{
			name:     "HTML 4.01 Transitional",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">`,
			expected: "HTML 4.01 Transitional",
		},
		{
			name:     "HTML 4.01 Frameset",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Frameset//EN" "http://www.w3.org/TR/html4/frameset.dtd">`,
			expected: "HTML 4.01 Frameset",
		},
		{
			name:     "XHTML 1.0 Strict",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">`,
			expected: "XHTML 1.0 Strict",
		},
		{
			name:     "XHTML 1.0 Transitional",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`,
			expected: "XHTML 1.0 Transitional",
		},
		{
			name:     "XHTML 1.0 Frameset",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Frameset//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd">`,
			expected: "XHTML 1.0 Frameset",
		},
		{
			name:     "XHTML 1.1",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`,
			expected: "XHTML 1.1",
		},
		{
			name:     "HTML 4.0",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.0//EN">`,
			expected: "HTML 4.0",
		},
		{
			name:     "HTML 3.2",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN">`,
			expected: "HTML 3.2",
		},
		{
			name:     "HTML 2.0",
			html:     `<!DOCTYPE html PUBLIC "-//IETF//DTD HTML 2.0//EN">`,
			expected: "HTML 2.0",
		},
		{
			name:     "Generic XHTML",
			html:     `<!DOCTYPE html PUBLIC "-//SOMETHING//DTD XHTML Custom//EN">`,
			expected: "XHTML (Generic)",
		},
		{
			name:     "Generic HTML",
			html:     `<!DOCTYPE HTML PUBLIC "-//SOMETHING//DTD HTML Custom//EN">`,
			expected: "HTML (Generic)",
		},
		{
			name:     "Unknown DOCTYPE",
			html:     `<!DOCTYPE something-else>`,
			expected: "Unknown",
		},
		{
			name:     "No DOCTYPE",
			html:     `<html><head><title>Test</title></head></html>`,
			expected: "HTML5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.detectHTMLVersion(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzer_PerformWebpageAnalysis(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMockMetrics()
	cache := &MockCache{}
	cfg := createTestConfig()
	analyzer := NewAnalyzer(cfg, logger, metrics, cache)

	html := `<!DOCTYPE html>
<html>
<head><title>Test Analysis Page</title></head>
<body>
	<h1>Main Heading</h1>
	<h2>Sub Heading 1</h2>
	<h2>Sub Heading 2</h2>
	<h3>Sub Sub Heading</h3>
	<a href="/internal">Internal Link</a>
	<a href="http://external.com">External Link</a>
	<form action="/login">
		<input type="text" name="username" />
		<input type="password" name="password" />
		<button type="submit">Login</button>
	</form>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	baseURL, err := url.Parse("http://example.com")
	require.NoError(t, err)

	ctx := context.Background()
	result := analyzer.performWebpageAnalysis(ctx, "http://example.com", html, doc, baseURL)

	assert.Equal(t, "http://example.com", result.URL)
	assert.Equal(t, "HTML5", result.HTMLVersion)
	assert.Equal(t, "Test Analysis Page", result.Title)
	assert.Equal(t, 1, result.Headings["h1"])
	assert.Equal(t, 2, result.Headings["h2"])
	assert.Equal(t, 1, result.Headings["h3"])
	assert.Equal(t, 0, result.Headings["h4"])
	assert.Equal(t, 0, result.Headings["h5"])
	assert.Equal(t, 0, result.Headings["h6"])
	assert.True(t, result.HasLoginForm)
	assert.NotZero(t, result.AnalyzedAt)
} 