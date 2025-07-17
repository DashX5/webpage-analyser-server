package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"

	"github.com/webpage-analyser-server/internal/config"
	"github.com/webpage-analyser-server/internal/constants"
	"github.com/webpage-analyser-server/internal/metrics"
	"github.com/webpage-analyser-server/internal/models"
)

// CacheInterface defines the interface for cache operations
type CacheInterface interface {
	Get(ctx context.Context, url string) (*models.AnalyzeResponse, error)
	Set(ctx context.Context, url string, result *models.AnalyzeResponse) error
	Close() error
}

// linkCheckRequest represents a request to check a link's accessibility
type linkCheckRequest struct {
	url        string
	isInternal bool
}

// Analyzer handles webpage analysis
type Analyzer struct {
	logger     *zap.Logger
	metrics    *metrics.Metrics
	httpClient *http.Client
	cache      CacheInterface
	config     *config.Config
}


func NewAnalyzer(cfg *config.Config, logger *zap.Logger, metrics *metrics.Metrics, cache CacheInterface) *Analyzer {
	if cfg.Analyzer.MaxLinks == 0 {
		cfg.Analyzer.MaxLinks = constants.DefaultMaxLinks
	}
	if cfg.Analyzer.LinkTimeout == 0 {
		cfg.Analyzer.LinkTimeout = constants.DefaultLinkTimeout
	}
	if cfg.Analyzer.MaxWorkers == 0 {
		cfg.Analyzer.MaxWorkers = constants.DefaultMaxWorkers
	}
	if cfg.Analyzer.MaxRedirects == 0 {
		cfg.Analyzer.MaxRedirects = constants.DefaultMaxRedirects
	}

	return &Analyzer{
		logger:  logger,
		metrics: metrics,
		httpClient: &http.Client{
			Timeout: cfg.Analyzer.LinkTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= cfg.Analyzer.MaxRedirects {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		cache:  cache,
		config: cfg,
	}
}

// Analyze performs the webpage analysis
func (a *Analyzer) Analyze(ctx context.Context, targetURL string) (*models.AnalyzeResponse, error) {
	// Check cache first
	if result, err := a.cache.Get(ctx, targetURL); err != nil {
		a.logger.Error("Failed to get from cache", zap.Error(err))
	} else if result != nil {
		return result, nil
	}

	// Parse and validate URL
	parsedURL, err := a.parseAndValidateURL(targetURL)
	if err != nil {
		return nil, err
	}

	// Fetch webpage content
	htmlContent, err := a.fetchWebpage(targetURL)
	if err != nil {
		return nil, err
	}

	// Parse HTML document
	doc, err := a.parseHTML(htmlContent)
	if err != nil {
		return nil, err
	}

	// Perform comprehensive analysis
	result := a.performWebpageAnalysis(ctx, targetURL, htmlContent, doc, parsedURL)

	// Cache the result
	if err := a.cache.Set(ctx, targetURL, result); err != nil {
		a.logger.Error("Failed to cache result", zap.Error(err))
	}

	return result, nil
}

// parseAndValidateURL parses and validates the target URL
func (a *Analyzer) parseAndValidateURL(targetURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	
	// Validate that the URL has a scheme and host
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid URL: missing scheme or host")
	}
	
	// Validate that the scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL: unsupported scheme %s", parsedURL.Scheme)
	}
	
	return parsedURL, nil
}

// fetchWebpage fetches the webpage content via HTTP
func (a *Analyzer) fetchWebpage(targetURL string) (string, error) {
	resp, err := a.httpClient.Get(targetURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch webpage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != constants.StatusOK {
		return "", fmt.Errorf("webpage returned status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}

// parseHTML parses the HTML content into a goquery document
func (a *Analyzer) parseHTML(htmlContent string) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return doc, nil
}

// performWebpageAnalysis performs comprehensive analysis of the webpage
func (a *Analyzer) performWebpageAnalysis(ctx context.Context, targetURL, htmlContent string, doc *goquery.Document, parsedURL *url.URL) *models.AnalyzeResponse {
	result := &models.AnalyzeResponse{
		URL:        targetURL,
		AnalyzedAt: time.Now(),
		Headings:   make(map[string]int),
	}

	// Detect HTML version
	result.HTMLVersion = a.detectHTMLVersion(htmlContent)

	// Extract page title
	result.Title = a.extractPageTitle(doc)

	// Count headings
	result.Headings = a.countHeadings(doc)

	// Analyze links
	result.Links = a.analyzeLinks(ctx, doc, parsedURL)

	// Check for login form
	result.HasLoginForm = a.detectLoginForm(doc)

	return result
}

// extractPageTitle extracts the page title from the document
func (a *Analyzer) extractPageTitle(doc *goquery.Document) string {
	return strings.TrimSpace(doc.Find("title").Text())
}

// countHeadings counts all heading elements (h1-h6) in the document
func (a *Analyzer) countHeadings(doc *goquery.Document) map[string]int {
	headings := make(map[string]int)
	
	for i := 1; i <= 6; i++ {
		selector := fmt.Sprintf("h%d", i)
		headings[selector] = doc.Find(selector).Length()
	}
	
	return headings
}

func (a *Analyzer) detectHTMLVersion(htmlContent string) string {
	// Extract and clean DOCTYPE
	doctype := a.extractDOCTYPE(htmlContent)
	if doctype == "" {
		return constants.HTMLVersion5 // No DOCTYPE found - assume HTML5
	}
	
	// HTML5 DOCTYPE (simple case)
	if regexp.MustCompile(constants.RegexHTML5DOCTYPE).MatchString(doctype) {
		return constants.HTMLVersion5
	}
	
	// Check for specific HTML versions with variants
	if version := a.checkHTMLVersionWithVariants(doctype, constants.DOCTYPEKeywordXHTML11, constants.HTMLVersionXHTML11, "", "", ""); version != "" {
		return version
	}
	
	if version := a.checkHTMLVersionWithVariants(doctype, constants.DOCTYPEKeywordXHTML10, constants.HTMLVersionXHTML10, 
		constants.HTMLVersionXHTML10Strict, constants.HTMLVersionXHTML10Transitional, constants.HTMLVersionXHTML10Frameset); version != "" {
		return version
	}
	
	if version := a.checkHTMLVersionWithVariants(doctype, constants.DOCTYPEKeywordHTML401, constants.HTMLVersionHTML401,
		constants.HTMLVersionHTML401Strict, constants.HTMLVersionHTML401Transitional, constants.HTMLVersionHTML401Frameset); version != "" {
		return version
	}
	
	// Check for simple HTML versions (no variants)
	simpleVersions := map[string]string{
		constants.DOCTYPEKeywordHTML40: constants.HTMLVersionHTML40,
		constants.DOCTYPEKeywordHTML32: constants.HTMLVersionHTML32,
		constants.DOCTYPEKeywordHTML20: constants.HTMLVersionHTML20,
	}
	
	for keyword, version := range simpleVersions {
		if strings.Contains(doctype, keyword) {
			return version
		}
	}
	
	// Generic fallback detection
	if strings.Contains(doctype, constants.DOCTYPEKeywordXHTML) {
		return constants.HTMLVersionXHTMLGeneric
	}
	
	if strings.Contains(doctype, constants.DOCTYPEKeywordHTML) {
		return constants.HTMLVersionHTMLGeneric
	}
	
	return constants.HTMLVersionUnknown
}

// extractDOCTYPE extracts and cleans the DOCTYPE declaration from HTML content
func (a *Analyzer) extractDOCTYPE(htmlContent string) string {
	// Remove leading whitespace
	cleanedHTML := strings.TrimSpace(htmlContent)
	
	// Remove XML declaration if present (for XHTML)
	xmlDeclRegex := regexp.MustCompile(constants.RegexXMLDeclaration)
	cleanedHTML = xmlDeclRegex.ReplaceAllString(cleanedHTML, "")
	
	// Remove any leading comments
	commentRegex := regexp.MustCompile(constants.RegexHTMLComment)
	cleanedHTML = commentRegex.ReplaceAllString(cleanedHTML, "")
	
	// Extract DOCTYPE declaration
	doctypeRegex := regexp.MustCompile(constants.RegexDOCTYPEExtraction)
	matches := doctypeRegex.FindString(cleanedHTML)
	
	// Convert to uppercase for easier matching
	return strings.ToUpper(matches)
}

// checkHTMLVersionWithVariants checks for HTML versions that have Strict/Transitional/Frameset variants
func (a *Analyzer) checkHTMLVersionWithVariants(doctype, keyword, baseVersion, strictVersion, transitionalVersion, framesetVersion string) string {
	if !strings.Contains(doctype, keyword) {
		return ""
	}
	
	// Check for variants if they are provided
	if strictVersion != "" && strings.Contains(doctype, constants.DOCTYPEKeywordStrict) {
		return strictVersion
	}
	
	if transitionalVersion != "" && strings.Contains(doctype, constants.DOCTYPEKeywordTransitional) {
		return transitionalVersion
	}
	
	if framesetVersion != "" && strings.Contains(doctype, constants.DOCTYPEKeywordFrameset) {
		return framesetVersion
	}
	
	// Return base version if no variants found
	return baseVersion
}

// analyzeLinks analyzes all links in the document
func (a *Analyzer) analyzeLinks(ctx context.Context, doc *goquery.Document, baseURL *url.URL) models.LinkAnalysis {
	var analysis models.LinkAnalysis
	var wg sync.WaitGroup
	linkChan := make(chan linkCheckRequest, a.config.Analyzer.MaxLinks)
	resultChan := make(chan bool, a.config.Analyzer.MaxLinks)

	// Start worker pool
	for i := 0; i < a.config.Analyzer.MaxWorkers; i++ {
		go a.linkWorker(ctx, &wg, linkChan, resultChan)
	}

	// Collect all links first
	var externalLinks []string
	var internalLinks []string

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			linkURL, err := baseURL.Parse(href)
			if err != nil {
				return
			}

			if linkURL.Host == baseURL.Host {
				analysis.Internal++
				internalLinks = append(internalLinks, linkURL.String())
			} else {
				analysis.External++
				externalLinks = append(externalLinks, linkURL.String())
			}
		}
	})

	// Check links with priority (external first, then internal up to limit)
	linksToCheck := 0
	maxLinksToCheck := a.config.Analyzer.MaxLinks

	// Add external links first (higher priority)
	for _, link := range externalLinks {
		if linksToCheck >= maxLinksToCheck {
			break
		}
		wg.Add(1)
		linkChan <- linkCheckRequest{url: link, isInternal: false}
		linksToCheck++
	}

	// Add internal links if we have capacity (limit to prevent performance issues)
	remainingCapacity := maxLinksToCheck - linksToCheck
	internalLinksToCheck := len(internalLinks)
	if internalLinksToCheck > remainingCapacity {
		internalLinksToCheck = remainingCapacity
	}

	for i := 0; i < internalLinksToCheck; i++ {
		wg.Add(1)
		linkChan <- linkCheckRequest{url: internalLinks[i], isInternal: true}
		linksToCheck++
	}

	// Close link channel and wait for workers
	close(linkChan)
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Count inaccessible links
	for accessible := range resultChan {
		if !accessible {
			analysis.Inaccessible++
		}
	}

	return analysis
}

// linkWorker checks if links are accessible
func (a *Analyzer) linkWorker(ctx context.Context, wg *sync.WaitGroup, links <-chan linkCheckRequest, results chan<- bool) {
	for linkReq := range links {
		start := time.Now()
		accessible := a.checkLinkWithTimeout(ctx, linkReq.url, linkReq.isInternal)
		a.metrics.LinkCheckDuration.Observe(time.Since(start).Seconds())
		results <- accessible
		wg.Done()
	}
}

// checkLinkWithTimeout checks if a link is accessible with different timeouts for internal vs external links
func (a *Analyzer) checkLinkWithTimeout(ctx context.Context, link string, isInternal bool) bool {
	// Create a client with appropriate timeout
	var client *http.Client
	if isInternal {
		// Use shorter timeout for internal links
		client = &http.Client{
			Timeout: constants.DefaultInternalLinkTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= a.config.Analyzer.MaxRedirects {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}
	} else {
		// Use the configured timeout for external links
		client = a.httpClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < constants.StatusBadRequest
}

// checkLink checks if a link is accessible (kept for backward compatibility)
func (a *Analyzer) checkLink(ctx context.Context, link string) bool {
	return a.checkLinkWithTimeout(ctx, link, false)
}

// detectLoginForm checks for the presence of a login form using a scoring system
func (a *Analyzer) detectLoginForm(doc *goquery.Document) bool {
	score := 0
	requiredScore := constants.DefaultLoginFormThreshold // Using constant instead of hardcoded value

	// Check for forms with both username/email and password fields
	doc.Find("form").Each(func(_ int, form *goquery.Selection) {
		formScore := 0

		// Check form attributes
		if action, exists := form.Attr("action"); exists {
			actionLower := strings.ToLower(action)
			if strings.Contains(actionLower, "login") || strings.Contains(actionLower, "signin") || strings.Contains(actionLower, "auth") {
				formScore += 3
			}
		}

		// Check for password field
		passwordFields := form.Find("input[type='password']")
		if passwordFields.Length() > 0 {
			formScore += 4
		}

		// Check for username/email field combinations
		userFields := form.Find("input[type='text'], input[type='email'], input[name*='username' i], input[name*='email' i], input[id*='username' i], input[id*='email' i]")
		if userFields.Length() > 0 {
			formScore += 3
		}

		// Check for submit button with login-related text
		form.Find("button[type='submit'], input[type='submit']").Each(func(_ int, btn *goquery.Selection) {
			btnText := strings.ToLower(btn.Text())
			if btnVal, exists := btn.Attr("value"); exists {
				btnText += " " + strings.ToLower(btnVal)
			}
			if strings.Contains(btnText, "login") || strings.Contains(btnText, "sign in") || strings.Contains(btnText, "log in") {
				formScore += 2
			}
		})

		// Check for remember me checkbox
		rememberMe := form.Find("input[type='checkbox']").FilterFunction(func(_ int, s *goquery.Selection) bool {
			label := s.Parent().Text()
			if labelFor, exists := s.Attr("id"); exists {
				form.Find("label[for='" + labelFor + "']").Each(func(_ int, l *goquery.Selection) {
					label += " " + l.Text()
				})
			}
			labelLower := strings.ToLower(label)
			return strings.Contains(labelLower, "remember me") || strings.Contains(labelLower, "keep me signed in")
		})
		if rememberMe.Length() > 0 {
			formScore += 2
		}

		// Check for forgot password link near the form
		forgotPwd := form.Find("a").FilterFunction(func(_ int, s *goquery.Selection) bool {
			text := strings.ToLower(s.Text())
			return strings.Contains(text, "forgot") && strings.Contains(text, "password")
		})
		if forgotPwd.Length() > 0 {
			formScore += 2
		}

		// Check for OAuth/SSO buttons with proper context
		oauthButtons := form.Find("button, a").FilterFunction(func(_ int, s *goquery.Selection) bool {
			text := strings.ToLower(s.Text())
			classes, _ := s.Attr("class")
			classesLower := strings.ToLower(classes)
			
			// Look for common OAuth provider patterns with proper context
			providers := []string{"google", "facebook", "github", "twitter", "microsoft"}
			for _, provider := range providers {
				if (strings.Contains(text, "sign in with "+provider) || 
					strings.Contains(text, "login with "+provider) ||
					(strings.Contains(classesLower, provider) && 
					(strings.Contains(classesLower, "auth") || strings.Contains(classesLower, "login") || strings.Contains(classesLower, "oauth")))) {
					return true
				}
			}
			return false
		})
		if oauthButtons.Length() > 0 {
			formScore += 2
		}

		// Add the highest scoring form to the total score
		if formScore > score {
			score = formScore
		}
	})

	// Check for login-specific meta tags or links
	doc.Find("meta[name*='sign' i], meta[name*='auth' i], link[rel*='authorization' i]").Each(func(_ int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists && strings.Contains(strings.ToLower(content), "auth") {
			score++
		}
	})

	// Return true if the score meets the threshold
	return score >= requiredScore
} 