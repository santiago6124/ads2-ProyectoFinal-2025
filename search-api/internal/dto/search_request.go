package dto

import (
	"strconv"
	"strings"
	"time"
)

// SearchRequest represents a search request with all possible parameters
type SearchRequest struct {
	Query           string    `form:"q" json:"query"`
	Page            int       `form:"page" json:"page" binding:"min=1"`
	Limit           int       `form:"limit" json:"limit" binding:"min=1,max=100"`
	Sort            string    `form:"sort" json:"sort"`
	Category        []string  `form:"category" json:"category"`
	MinPrice        *float64  `form:"min_price" json:"min_price" binding:"omitempty,min=0"`
	MaxPrice        *float64  `form:"max_price" json:"max_price" binding:"omitempty,min=0"`
	MinMarketCap    *int64    `form:"min_market_cap" json:"min_market_cap" binding:"omitempty,min=0"`
	MaxMarketCap    *int64    `form:"max_market_cap" json:"max_market_cap" binding:"omitempty,min=0"`
	PriceChange24h  string    `form:"price_change_24h" json:"price_change_24h"`
	IsTrending      *bool     `form:"is_trending" json:"is_trending"`
	Platform        string    `form:"platform" json:"platform"`
	Tags            []string  `form:"tags" json:"tags"`
	IsActive        *bool     `form:"is_active" json:"is_active"`
}

// TrendingRequest represents a request for trending cryptocurrencies
type TrendingRequest struct {
	Period string `form:"period" json:"period" binding:"omitempty,oneof=1h 24h 7d 30d"`
	Limit  int    `form:"limit" json:"limit" binding:"min=1,max=50"`
}

// SuggestionRequest represents an autocomplete request
type SuggestionRequest struct {
	Query string `form:"q" json:"query" binding:"required,min=1,max=50"`
	Limit int    `form:"limit" json:"limit" binding:"min=1,max=10"`
}

// ReindexRequest represents a reindexing request
type ReindexRequest struct {
	FullReindex   bool `json:"full_reindex"`
	ClearExisting bool `json:"clear_existing"`
	BatchSize     int  `json:"batch_size" binding:"min=1,max=1000"`
}

// SetDefaults sets default values for search request
func (r *SearchRequest) SetDefaults() {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 100 {
		r.Limit = 100
	}
	if r.Sort == "" {
		r.Sort = "market_cap_desc"
	}
	if r.IsActive == nil {
		defaultActive := true
		r.IsActive = &defaultActive
	}
}

// SetDefaults sets default values for trending request
func (r *TrendingRequest) SetDefaults() {
	if r.Period == "" {
		r.Period = "24h"
	}
	if r.Limit <= 0 {
		r.Limit = 10
	}
	if r.Limit > 50 {
		r.Limit = 50
	}
}

// Validate validates the trending request
func (r *TrendingRequest) Validate() error {
	// Validate period
	validPeriods := map[string]bool{
		"1h": true, "24h": true, "7d": true, "30d": true,
	}

	if r.Period != "" && !validPeriods[r.Period] {
		return NewValidationError("invalid period: must be one of 1h, 24h, 7d, 30d")
	}

	// Validate limit
	if r.Limit < 1 || r.Limit > 50 {
		return NewValidationError("limit must be between 1 and 50")
	}

	return nil
}

// SetDefaults sets default values for suggestion request
func (r *SuggestionRequest) SetDefaults() {
	if r.Limit <= 0 {
		r.Limit = 5
	}
	if r.Limit > 10 {
		r.Limit = 10
	}
}

// SetDefaults sets default values for reindex request
func (r *ReindexRequest) SetDefaults() {
	if r.BatchSize <= 0 {
		r.BatchSize = 100
	}
	if r.BatchSize > 1000 {
		r.BatchSize = 1000
	}
}

// Validate validates the search request
func (r *SearchRequest) Validate() error {
	// Validate price range
	if r.MinPrice != nil && r.MaxPrice != nil && *r.MinPrice > *r.MaxPrice {
		return NewValidationError("min_price cannot be greater than max_price")
	}

	// Validate market cap range
	if r.MinMarketCap != nil && r.MaxMarketCap != nil && *r.MinMarketCap > *r.MaxMarketCap {
		return NewValidationError("min_market_cap cannot be greater than max_market_cap")
	}

	// Validate price change filter
	if r.PriceChange24h != "" && r.PriceChange24h != "positive" && r.PriceChange24h != "negative" {
		return NewValidationError("price_change_24h must be 'positive' or 'negative'")
	}

	// Validate category
	validCategories := map[string]bool{
		"DeFi": true, "NFT": true, "Gaming": true, "Layer1": true, "Layer2": true,
		"Metaverse": true, "Web3": true, "AI": true, "Infrastructure": true,
		"Privacy": true, "Oracle": true, "Exchange": true,
	}

	for _, cat := range r.Category {
		if !validCategories[cat] {
			return NewValidationError("invalid category: " + cat)
		}
	}

	// Validate sort
	validSorts := map[string]bool{
		"": true, "price_asc": true, "price_desc": true, "market_cap_desc": true,
		"market_cap_asc": true, "trending_desc": true, "trending_asc": true,
		"name_asc": true, "name_desc": true, "volume_desc": true, "volume_asc": true,
		"change_desc": true, "change_asc": true,
	}

	if !validSorts[r.Sort] {
		return NewValidationError("invalid sort option: " + r.Sort)
	}

	return nil
}

// GetOffset calculates the offset for pagination
func (r *SearchRequest) GetOffset() int {
	return (r.Page - 1) * r.Limit
}

// ToSolrQuery converts search request to Solr query parameters
func (r *SearchRequest) ToSolrQuery() map[string]interface{} {
	params := make(map[string]interface{})

	// Main query
	if r.Query == "" {
		params["q"] = "*:*"
	} else {
		// Use dismax parser for better relevance
		params["q"] = r.Query
		params["defType"] = "edismax"
		params["qf"] = "name^10 symbol^8 search_text^2"
		params["pf"] = "name^20 symbol^15"
		params["ps"] = "2"
		params["qs"] = "1"
	}

	// Filters
	filters := make([]string, 0)

	// Active filter
	if r.IsActive != nil {
		filters = append(filters, "is_active:"+strconv.FormatBool(*r.IsActive))
	}

	// Price filters
	if r.MinPrice != nil || r.MaxPrice != nil {
		priceFilter := "current_price:["
		if r.MinPrice != nil {
			priceFilter += strconv.FormatFloat(*r.MinPrice, 'f', -1, 64)
		} else {
			priceFilter += "*"
		}
		priceFilter += " TO "
		if r.MaxPrice != nil {
			priceFilter += strconv.FormatFloat(*r.MaxPrice, 'f', -1, 64)
		} else {
			priceFilter += "*"
		}
		priceFilter += "]"
		filters = append(filters, priceFilter)
	}

	// Market cap filters
	if r.MinMarketCap != nil || r.MaxMarketCap != nil {
		mcFilter := "market_cap:["
		if r.MinMarketCap != nil {
			mcFilter += strconv.FormatInt(*r.MinMarketCap, 10)
		} else {
			mcFilter += "*"
		}
		mcFilter += " TO "
		if r.MaxMarketCap != nil {
			mcFilter += strconv.FormatInt(*r.MaxMarketCap, 10)
		} else {
			mcFilter += "*"
		}
		mcFilter += "]"
		filters = append(filters, mcFilter)
	}

	// Category filter
	if len(r.Category) > 0 {
		categoryFilter := "category:(" + strings.Join(r.Category, " OR ") + ")"
		filters = append(filters, categoryFilter)
	}

	// Tags filter
	if len(r.Tags) > 0 {
		tagsFilter := "tags:(" + strings.Join(r.Tags, " OR ") + ")"
		filters = append(filters, tagsFilter)
	}

	// Platform filter
	if r.Platform != "" {
		filters = append(filters, "platform:\""+r.Platform+"\"")
	}

	// Price change filter
	if r.PriceChange24h != "" {
		if r.PriceChange24h == "positive" {
			filters = append(filters, "price_change_24h:[0 TO *]")
		} else if r.PriceChange24h == "negative" {
			filters = append(filters, "price_change_24h:[* TO 0]")
		}
	}

	// Trending filter
	if r.IsTrending != nil {
		filters = append(filters, "is_trending:"+strconv.FormatBool(*r.IsTrending))
	}

	if len(filters) > 0 {
		params["fq"] = filters
	}

	// Sorting
	if r.Sort != "" && r.Sort != "relevance" {
		sortMap := map[string]string{
			"price_asc":       "current_price asc",
			"price_desc":      "current_price desc",
			"market_cap_desc": "market_cap desc",
			"market_cap_asc":  "market_cap asc",
			"trending_desc":   "trending_score desc",
			"trending_asc":    "trending_score asc",
			"name_asc":        "name_exact asc",
			"name_desc":       "name_exact desc",
			"volume_desc":     "volume_24h desc",
			"volume_asc":      "volume_24h asc",
			"change_desc":     "price_change_24h desc",
			"change_asc":      "price_change_24h asc",
		}

		if solrSort, exists := sortMap[r.Sort]; exists {
			params["sort"] = solrSort
		}
	}

	// Pagination
	params["start"] = r.GetOffset()
	params["rows"] = r.Limit

	// Faceting
	params["facet"] = "true"
	params["facet.field"] = []string{"category", "platform"}
	params["facet.range"] = []string{"current_price", "market_cap"}

	// Price range facets
	params["facet.range.start"] = 0
	params["facet.range.end"] = 10000
	params["facet.range.gap"] = 100

	// Highlighting
	if r.Query != "" {
		params["hl"] = "true"
		params["hl.fl"] = "name,description"
		params["hl.simple.pre"] = "<mark>"
		params["hl.simple.post"] = "</mark>"
		params["hl.fragsize"] = 100
		params["hl.maxAnalyzedChars"] = 500
	}

	// Response format
	params["wt"] = "json"
	params["indent"] = "true"

	return params
}

// ToCacheKey generates a cache key for the search request
func (r *SearchRequest) ToCacheKey() string {
	parts := []string{
		"search",
		"q:" + r.Query,
		"page:" + strconv.Itoa(r.Page),
		"limit:" + strconv.Itoa(r.Limit),
		"sort:" + r.Sort,
	}

	if len(r.Category) > 0 {
		parts = append(parts, "cat:"+strings.Join(r.Category, ","))
	}

	if r.MinPrice != nil {
		parts = append(parts, "minp:"+strconv.FormatFloat(*r.MinPrice, 'f', -1, 64))
	}

	if r.MaxPrice != nil {
		parts = append(parts, "maxp:"+strconv.FormatFloat(*r.MaxPrice, 'f', -1, 64))
	}

	if r.IsTrending != nil {
		parts = append(parts, "trending:"+strconv.FormatBool(*r.IsTrending))
	}

	return strings.Join(parts, ":")
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// IsEmpty checks if the search request is empty (no filters applied)
func (r *SearchRequest) IsEmpty() bool {
	return r.Query == "" &&
		len(r.Category) == 0 &&
		r.MinPrice == nil &&
		r.MaxPrice == nil &&
		r.MinMarketCap == nil &&
		r.MaxMarketCap == nil &&
		r.PriceChange24h == "" &&
		r.IsTrending == nil &&
		r.Platform == "" &&
		len(r.Tags) == 0
}

// GetCacheTTL returns the appropriate cache TTL based on request type
func (r *SearchRequest) GetCacheTTL() time.Duration {
	// Popular searches (empty query, trending) get longer cache
	if r.IsEmpty() || (r.IsTrending != nil && *r.IsTrending) {
		return 10 * time.Minute
	}

	// Specific searches get shorter cache
	if r.Query != "" {
		return 5 * time.Minute
	}

	// Default cache duration
	return 3 * time.Minute
}