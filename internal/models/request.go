package models

import (
	"fmt"
	"net/url"

	"github.com/webpage-analyser-server/internal/constants"
)

// AnalyzeRequest represents the request payload for webpage analysis
type AnalyzeRequest struct {
	URL string `json:"url" validate:"required,url"`
}

// Validate performs custom validation on the request
func (r *AnalyzeRequest) Validate() error {
	if len(r.URL) > constants.MaxURLLength {
		return fmt.Errorf("URL length exceeds maximum allowed length of %d characters", constants.MaxURLLength)
	}

	// Parse and validate URL using net/url package
	parsedURL, err := url.Parse(r.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check for required URL components
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("URL must have both scheme (http/https) and host")
	}

	// Only allow HTTP and HTTPS schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}

	return nil
} 