package models

import "time"

// AnalyzeResponse represents the response payload for webpage analysis
type AnalyzeResponse struct {
	URL         string            `json:"url"`
	HTMLVersion string            `json:"html_version"`
	Title       string            `json:"title"`
	Headings    map[string]int    `json:"headings"`
	Links       LinkAnalysis      `json:"links"`
	HasLoginForm bool             `json:"has_login_form"`
	AnalyzedAt  time.Time         `json:"analyzed_at"`
}

// LinkAnalysis represents the analysis of links in the webpage
type LinkAnalysis struct {
	Internal     int `json:"internal"`
	External     int `json:"external"`
	Inaccessible int `json:"inaccessible"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
} 