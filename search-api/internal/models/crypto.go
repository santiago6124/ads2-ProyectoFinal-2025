package models

import (
	"fmt"
	"time"
)

// Crypto represents a cryptocurrency in the search index
type Crypto struct {
	ID                  string    `json:"id" solr:"id"`
	Symbol              string    `json:"symbol" solr:"symbol"`
	Name                string    `json:"name" solr:"name"`
	Description         string    `json:"description" solr:"description"`
	CurrentPrice        float64   `json:"current_price" solr:"current_price"`
	MarketCap           int64     `json:"market_cap" solr:"market_cap"`
	MarketCapRank       int       `json:"market_cap_rank" solr:"market_cap_rank"`
	Volume24h           int64     `json:"volume_24h" solr:"volume_24h"`
	PriceChange24h      float64   `json:"price_change_24h" solr:"price_change_24h"`
	PriceChangePercent24h float64 `json:"price_change_percent_24h" solr:"price_change_percent_24h"`
	PriceChange7d       float64   `json:"price_change_7d" solr:"price_change_7d"`
	PriceChange30d      float64   `json:"price_change_30d" solr:"price_change_30d"`
	CirculatingSupply   int64     `json:"circulating_supply" solr:"circulating_supply"`
	TotalSupply         *int64    `json:"total_supply" solr:"total_supply"`
	MaxSupply           *int64    `json:"max_supply" solr:"max_supply"`
	TrendingScore       float32   `json:"trending_score" solr:"trending_score"`
	PopularityScore     float32   `json:"popularity_score" solr:"popularity_score"`
	Category            []string  `json:"category" solr:"category"`
	Tags                []string  `json:"tags" solr:"tags"`
	Platform            string    `json:"platform" solr:"platform"`
	ATH                 float64   `json:"ath" solr:"ath"`
	ATHDate             time.Time `json:"ath_date" solr:"ath_date"`
	ATL                 float64   `json:"atl" solr:"atl"`
	ATLDate             time.Time `json:"atl_date" solr:"atl_date"`
	IsActive            bool      `json:"is_active" solr:"is_active"`
	IsTrending          bool      `json:"is_trending" solr:"is_trending"`
	LastUpdated         time.Time `json:"last_updated" solr:"last_updated"`
	IndexedAt           time.Time `json:"indexed_at" solr:"indexed_at"`
}

// SearchResult represents a search result with additional metadata
type SearchResult struct {
	Crypto
	Score        float64 `json:"score"`
	MatchType    string  `json:"match_type"`
	Highlighting map[string][]string `json:"highlighting,omitempty"`
}

// TrendingCrypto represents a cryptocurrency in trending results
type TrendingCrypto struct {
	Rank                    int     `json:"rank"`
	ID                      string  `json:"id"`
	Symbol                  string  `json:"symbol"`
	Name                    string  `json:"name"`
	CurrentPrice            float64 `json:"current_price"`
	PriceChange24h          float64 `json:"price_change_24h"`
	Volume24h               int64   `json:"volume_24h"`
	TrendingScore           float32 `json:"trending_score"`
	SearchVolumeIncrease    string  `json:"search_volume_increase"`
	MentionsCount           int64   `json:"mentions_count"`
}

// Suggestion represents an autocomplete suggestion
type Suggestion struct {
	ID        string  `json:"id"`
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	MatchType string  `json:"match_type"`
	Score     float32 `json:"score"`
}

// Filter represents available search filters
type Filter struct {
	Categories      []CategoryFilter       `json:"categories"`
	Platforms       []PlatformFilter       `json:"platforms"`
	Tags            []TagFilter            `json:"tags"`
	PriceRanges     []PriceRangeFilter     `json:"price_ranges"`
	MarketCapRanges []MarketCapRangeFilter `json:"market_cap_ranges"`
	SortOptions     []SortOption           `json:"sort_options"`
}

// PlatformFilter represents a platform filter with count
type PlatformFilter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// TagFilter represents a tag filter with count
type TagFilter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// CategoryFilter represents a category filter with count
type CategoryFilter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Count int64  `json:"count"`
	Label string `json:"label"`
}

// PriceRangeFilter represents a price range filter
type PriceRangeFilter struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int64   `json:"count"`
	Label string  `json:"label"`
}

// MarketCapRangeFilter represents a market cap range filter
type MarketCapRangeFilter struct {
	Min   int64  `json:"min"`
	Max   int64  `json:"max"`
	Count int64  `json:"count"`
	Label string `json:"label"`
}

// Type aliases for backward compatibility with tests
type FilterCategory = CategoryFilter
type FilterPlatform = PlatformFilter
type FilterTag = TagFilter
type PriceRange = PriceRangeFilter
type MarketCapRange = MarketCapRangeFilter

// SortOption represents a sorting option
type SortOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Pagination represents pagination information
type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int64 `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// Facets represents faceted search results
type Facets struct {
	Categories    map[string]int64 `json:"categories"`
	PriceRanges   map[string]int64 `json:"price_ranges"`
}

// QueryInfo represents query execution information
type QueryInfo struct {
	Query           string `json:"query"`
	ExecutionTimeMS int64  `json:"execution_time_ms"`
	CacheHit        bool   `json:"cache_hit"`
	TotalFound      int64  `json:"total_found"`
}

// SearchResultsResponse represents the complete search response
type SearchResultsResponse struct {
	Results    []SearchResult `json:"results"`
	Pagination Pagination     `json:"pagination"`
	Facets     Facets         `json:"facets"`
	QueryInfo  QueryInfo      `json:"query_info"`
}

// TrendingResponse represents trending cryptocurrencies response
type TrendingResponse struct {
	Trending  []TrendingCrypto `json:"trending"`
	Period    string           `json:"period"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// SuggestionsResponse represents autocomplete suggestions response
type SuggestionsResponse struct {
	Suggestions     []Suggestion `json:"suggestions"`
	Query           string       `json:"query"`
	ExecutionTimeMS int64        `json:"execution_time_ms"`
}

// FiltersResponse represents available filters response
type FiltersResponse struct {
	Filters Filter `json:"filters"`
}

// Validation methods

// Validate validates the Crypto struct
func (c *Crypto) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("crypto ID is required")
	}
	if c.Symbol == "" {
		return fmt.Errorf("crypto symbol is required")
	}
	if len(c.Symbol) > 10 {
		return fmt.Errorf("crypto symbol cannot exceed 10 characters")
	}
	if c.Name == "" {
		return fmt.Errorf("crypto name is required")
	}
	if c.CurrentPrice < 0 {
		return fmt.Errorf("current price cannot be negative")
	}
	if c.MarketCap < 0 {
		return fmt.Errorf("market cap cannot be negative")
	}
	if c.Volume24h < 0 {
		return fmt.Errorf("volume 24h cannot be negative")
	}
	return nil
}

// IsValidSort checks if the sort option is valid
func IsValidSort(sort string) bool {
	validSorts := map[string]bool{
		"":                true, // default
		"price_asc":       true,
		"price_desc":      true,
		"market_cap_desc": true,
		"market_cap_asc":  true,
		"trending_desc":   true,
		"trending_asc":    true,
		"name_asc":        true,
		"name_desc":       true,
		"volume_desc":     true,
		"volume_asc":      true,
		"change_desc":     true,
		"change_asc":      true,
	}
	return validSorts[sort]
}

// IsValidPeriod checks if the trending period is valid
func IsValidPeriod(period string) bool {
	validPeriods := map[string]bool{
		"1h":  true,
		"24h": true,
		"7d":  true,
		"30d": true,
	}
	return validPeriods[period]
}

// IsValidCategory checks if the category is valid
func IsValidCategory(category string) bool {
	validCategories := map[string]bool{
		"DeFi":         true,
		"NFT":          true,
		"Gaming":       true,
		"Layer1":       true,
		"Layer2":       true,
		"Metaverse":    true,
		"Web3":         true,
		"AI":           true,
		"Infrastructure": true,
		"Privacy":      true,
		"Oracle":       true,
		"Exchange":     true,
	}
	return validCategories[category]
}

// GetDefaultFilters returns the default filter configuration
func GetDefaultFilters() Filter {
	return Filter{
		Categories: []CategoryFilter{
			{Name: "Currency", Value: "Currency", Label: "Currency", Count: 0},
			{Name: "DeFi", Value: "DeFi", Label: "Decentralized Finance", Count: 0},
			{Name: "NFT", Value: "NFT", Label: "Non-Fungible Tokens", Count: 0},
			{Name: "Gaming", Value: "Gaming", Label: "Gaming & Metaverse", Count: 0},
			{Name: "Layer1", Value: "Layer1", Label: "Layer 1 Blockchains", Count: 0},
			{Name: "Layer2", Value: "Layer2", Label: "Layer 2 Solutions", Count: 0},
			{Name: "Smart Contract Platform", Value: "Smart Contract Platform", Label: "Smart Contract Platform", Count: 0},
			{Name: "Web3", Value: "Web3", Label: "Web3 Infrastructure", Count: 0},
			{Name: "AI", Value: "AI", Label: "Artificial Intelligence", Count: 0},
			{Name: "Privacy", Value: "Privacy", Label: "Privacy Coins", Count: 0},
		},
		Platforms: []PlatformFilter{
			{Name: "Ethereum", Value: "ethereum", Count: 0},
			{Name: "Binance Smart Chain", Value: "bsc", Count: 0},
			{Name: "Polygon", Value: "polygon", Count: 0},
			{Name: "Solana", Value: "solana", Count: 0},
			{Name: "Avalanche", Value: "avalanche", Count: 0},
		},
		Tags: []TagFilter{
			{Name: "DeFi", Value: "defi", Count: 0},
			{Name: "NFT", Value: "nft", Count: 0},
			{Name: "Gaming", Value: "gaming", Count: 0},
			{Name: "Layer1", Value: "layer1", Count: 0},
			{Name: "Layer2", Value: "layer2", Count: 0},
		},
		PriceRanges: []PriceRangeFilter{
			{Min: 0, Max: 1, Label: "Under $1", Count: 0},
			{Min: 1, Max: 10, Label: "$1 - $10", Count: 0},
			{Min: 10, Max: 100, Label: "$10 - $100", Count: 0},
			{Min: 100, Max: 1000, Label: "$100 - $1,000", Count: 0},
			{Min: 1000, Max: -1, Label: "$1,000+", Count: 0},
		},
		MarketCapRanges: []MarketCapRangeFilter{
			{Min: 0, Max: 1000000, Label: "Micro Cap (Under $1M)", Count: 0},
			{Min: 1000000, Max: 10000000, Label: "Small Cap ($1M - $10M)", Count: 0},
			{Min: 10000000, Max: 100000000, Label: "Mid Cap ($10M - $100M)", Count: 0},
			{Min: 100000000, Max: 1000000000, Label: "Large Cap ($100M - $1B)", Count: 0},
			{Min: 1000000000, Max: -1, Label: "Mega Cap ($1B+)", Count: 0},
		},
		SortOptions: []SortOption{
			{Value: "market_cap_desc", Label: "Market Cap ↓"},
			{Value: "price_desc", Label: "Price ↓"},
			{Value: "price_asc", Label: "Price ↑"},
			{Value: "trending_desc", Label: "Trending ↓"},
			{Value: "volume_desc", Label: "Volume 24h ↓"},
			{Value: "change_desc", Label: "Change 24h ↓"},
			{Value: "change_asc", Label: "Change 24h ↑"},
			{Value: "name_asc", Label: "Name A-Z"},
			{Value: "name_desc", Label: "Name Z-A"},
		},
	}
}