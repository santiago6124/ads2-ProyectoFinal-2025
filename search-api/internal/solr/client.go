package solr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"search-api/internal/models"
)

// Client represents a Solr client with connection pooling and retry logic
type Client struct {
	baseURL    string
	core       string
	httpClient *http.Client
	config     *Config
}

// Config represents Solr client configuration
type Config struct {
	BaseURL         string
	Core            string
	Timeout         time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
}

// SolrResponse represents a generic Solr response
type SolrResponse struct {
	ResponseHeader ResponseHeader `json:"responseHeader"`
	Response       ResponseData   `json:"response"`
	Facets         interface{}    `json:"facet_counts,omitempty"`
	Highlighting   interface{}    `json:"highlighting,omitempty"`
}

// ResponseHeader represents Solr response header
type ResponseHeader struct {
	Status int `json:"status"`
	QTime  int `json:"QTime"`
}

// ResponseData represents Solr response data
type ResponseData struct {
	NumFound int64                    `json:"numFound"`
	Start    int64                    `json:"start"`
	Docs     []map[string]interface{} `json:"docs"`
}

// FacetCounts represents facet counts in Solr response
type FacetCounts struct {
	FacetFields map[string][]interface{} `json:"facet_fields"`
	FacetRanges map[string]interface{}   `json:"facet_ranges"`
}

// NewClient creates a new Solr client
func NewClient(config *Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.MaxConnsPerHost == 0 {
		config.MaxConnsPerHost = 50
	}

	// Configure HTTP client with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxConnsPerHost,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	return &Client{
		baseURL:    strings.TrimSuffix(config.BaseURL, "/"),
		core:       config.Core,
		httpClient: httpClient,
		config:     config,
	}
}

// Search performs a search query
func (c *Client) Search(ctx context.Context, params map[string]interface{}) (*SolrResponse, error) {
	endpoint := fmt.Sprintf("%s/%s/select", c.baseURL, c.core)

	var response *SolrResponse
	var lastError error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
			}
		}

		response, lastError = c.performSearch(ctx, endpoint, params)
		if lastError == nil {
			return response, nil
		}

		// Don't retry on certain errors
		if isNonRetryableError(lastError) {
			break
		}
	}

	return nil, fmt.Errorf("search failed after %d attempts: %w", c.config.MaxRetries+1, lastError)
}

// performSearch executes a single search request
func (c *Client) performSearch(ctx context.Context, endpoint string, params map[string]interface{}) (*SolrResponse, error) {
	// Build query string
	queryParams := c.buildQueryParams(params)
	fullURL := endpoint + "?" + queryParams.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Solr returned status %d", resp.StatusCode)
	}

	// Parse response
	var solrResponse SolrResponse
	if err := json.NewDecoder(resp.Body).Decode(&solrResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if solrResponse.ResponseHeader.Status != 0 {
		return nil, fmt.Errorf("Solr query failed with status %d", solrResponse.ResponseHeader.Status)
	}

	return &solrResponse, nil
}

// Update adds or updates documents in Solr
func (c *Client) Update(ctx context.Context, docs []interface{}) error {
	endpoint := fmt.Sprintf("%s/%s/update/json/docs", c.baseURL, c.core)

	var payload interface{}
	if len(docs) == 1 {
		payload = docs[0]
	} else {
		payload = docs
	}

	return c.performUpdate(ctx, endpoint, payload)
}

// Delete deletes documents by ID
func (c *Client) Delete(ctx context.Context, ids []string) error {
	endpoint := fmt.Sprintf("%s/%s/update", c.baseURL, c.core)

	deleteData := map[string]interface{}{
		"delete": ids,
	}

	return c.performUpdate(ctx, endpoint, deleteData)
}

// DeleteByQuery deletes documents by query
func (c *Client) DeleteByQuery(ctx context.Context, query string) error {
	endpoint := fmt.Sprintf("%s/%s/update", c.baseURL, c.core)

	deleteData := map[string]interface{}{
		"delete": map[string]string{
			"query": query,
		},
	}

	return c.performUpdate(ctx, endpoint, deleteData)
}

// Commit commits changes to Solr
func (c *Client) Commit(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/%s/update", c.baseURL, c.core)

	commitData := map[string]interface{}{
		"commit": map[string]interface{}{},
	}

	return c.performUpdate(ctx, endpoint, commitData)
}

// Optimize optimizes the Solr index
func (c *Client) Optimize(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/%s/update", c.baseURL, c.core)

	optimizeData := map[string]interface{}{
		"optimize": map[string]interface{}{},
	}

	return c.performUpdate(ctx, endpoint, optimizeData)
}

// performUpdate executes an update request
func (c *Client) performUpdate(ctx context.Context, endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Solr update returned status %d", resp.StatusCode)
	}

	return nil
}

// Ping checks if Solr is available
func (c *Client) Ping(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/%s/admin/ping", c.baseURL, c.core)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Solr ping returned status %d", resp.StatusCode)
	}

	return nil
}

// GetDocumentCount returns the total number of documents in the core
func (c *Client) GetDocumentCount(ctx context.Context) (int64, error) {
	params := map[string]interface{}{
		"q":    "*:*",
		"rows": 0,
	}

	response, err := c.Search(ctx, params)
	if err != nil {
		return 0, err
	}

	return response.Response.NumFound, nil
}

// Suggest performs autocompletion suggestions
func (c *Client) Suggest(ctx context.Context, query string, limit int) ([]models.Suggestion, error) {
	params := map[string]interface{}{
		"q":       fmt.Sprintf("name:%s* OR symbol:%s*", query, strings.ToUpper(query)),
		"fl":      "id,symbol,name,current_price,market_cap_rank",
		"rows":    limit,
		"sort":    "market_cap_rank asc",
		"defType": "edismax",
		"qf":      "symbol^10 name^5",
	}

	response, err := c.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	suggestions := make([]models.Suggestion, 0, len(response.Response.Docs))
	for i, doc := range response.Response.Docs {
		suggestion := models.Suggestion{
			ID:        getString(doc, "id"),
			Symbol:    getString(doc, "symbol"),
			Name:      getString(doc, "name"),
			Score:     float64(100 - i), // Simple scoring based on order
			MatchType: determineMatchType(query, getString(doc, "symbol"), getString(doc, "name")),
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// buildQueryParams converts parameters map to URL values
func (c *Client) buildQueryParams(params map[string]interface{}) url.Values {
	values := url.Values{}

	for key, value := range params {
		switch v := value.(type) {
		case string:
			values.Set(key, v)
		case int:
			values.Set(key, strconv.Itoa(v))
		case int64:
			values.Set(key, strconv.FormatInt(v, 10))
		case float64:
			values.Set(key, strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			values.Set(key, strconv.FormatBool(v))
		case []string:
			for _, item := range v {
				values.Add(key, item)
			}
		case []interface{}:
			for _, item := range v {
				values.Add(key, fmt.Sprintf("%v", item))
			}
		default:
			values.Set(key, fmt.Sprintf("%v", v))
		}
	}

	return values
}

// Helper functions

func getString(doc map[string]interface{}, key string) string {
	if value, exists := doc[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64(doc map[string]interface{}, key string) float64 {
	if value, exists := doc[key]; exists {
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

func getInt64(doc map[string]interface{}, key string) int64 {
	if value, exists := doc[key]; exists {
		switch v := value.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

func getBool(doc map[string]interface{}, key string) bool {
	if value, exists := doc[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func getStringSlice(doc map[string]interface{}, key string) []string {
	if value, exists := doc[key]; exists {
		switch v := value.(type) {
		case []string:
			return v
		case []interface{}:
			result := make([]string, len(v))
			for i, item := range v {
				result[i] = fmt.Sprintf("%v", item)
			}
			return result
		}
	}
	return nil
}

func getTime(doc map[string]interface{}, key string) time.Time {
	if value, exists := doc[key]; exists {
		if timeStr, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func determineMatchType(query, symbol, name string) string {
	queryLower := strings.ToLower(query)
	symbolLower := strings.ToLower(symbol)
	nameLower := strings.ToLower(name)

	if strings.HasPrefix(symbolLower, queryLower) {
		return "symbol"
	}
	if strings.HasPrefix(nameLower, queryLower) {
		return "name"
	}
	if strings.Contains(symbolLower, queryLower) {
		return "symbol_partial"
	}
	if strings.Contains(nameLower, queryLower) {
		return "name_partial"
	}
	return "other"
}

func isNonRetryableError(err error) bool {
	// Don't retry on certain errors like 400 Bad Request
	if strings.Contains(err.Error(), "status 400") {
		return true
	}
	if strings.Contains(err.Error(), "status 401") {
		return true
	}
	if strings.Contains(err.Error(), "status 403") {
		return true
	}
	if strings.Contains(err.Error(), "status 404") {
		return true
	}
	return false
}

// DocToSearchResult converts a Solr document to SearchResult
func DocToSearchResult(doc map[string]interface{}, score float64) models.SearchResult {
	crypto := models.Crypto{
		ID:                getString(doc, "id"),
		Symbol:            getString(doc, "symbol"),
		Name:              getString(doc, "name"),
		Description:       getString(doc, "description"),
		CurrentPrice:      getFloat64(doc, "current_price"),
		MarketCap:         getInt64(doc, "market_cap"),
		MarketCapRank:     int(getInt64(doc, "market_cap_rank")),
		Volume24h:         getInt64(doc, "volume_24h"),
		PriceChange24h:    getFloat64(doc, "price_change_24h"),
		PriceChange7d:     getFloat64(doc, "price_change_7d"),
		PriceChange30d:    getFloat64(doc, "price_change_30d"),
		CirculatingSupply: getInt64(doc, "circulating_supply"),
		TrendingScore:     float32(getFloat64(doc, "trending_score")),
		PopularityScore:   float32(getFloat64(doc, "popularity_score")),
		Category:          getStringSlice(doc, "category"),
		Tags:              getStringSlice(doc, "tags"),
		Platform:          getString(doc, "platform"),
		ATH:               getFloat64(doc, "ath"),
		ATHDate:           getTime(doc, "ath_date"),
		ATL:               getFloat64(doc, "atl"),
		ATLDate:           getTime(doc, "atl_date"),
		IsActive:          getBool(doc, "is_active"),
		IsTrending:        getBool(doc, "is_trending"),
		LastUpdated:       getTime(doc, "last_updated"),
		IndexedAt:         getTime(doc, "indexed_at"),
	}

	// Handle nullable fields
	if totalSupply := getInt64(doc, "total_supply"); totalSupply > 0 {
		crypto.TotalSupply = &totalSupply
	}
	if maxSupply := getInt64(doc, "max_supply"); maxSupply > 0 {
		crypto.MaxSupply = &maxSupply
	}

	return models.SearchResult{
		Crypto: crypto,
		Score:  score,
	}
}

// DefaultConfig returns default Solr configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:         "http://localhost:8983/solr",
		Core:            "cryptos",
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      time.Second,
		MaxIdleConns:    100,
		MaxConnsPerHost: 50,
	}
}
