package models

import (
	"time"
)

// Order represents an order in the search index
type Order struct {
	ID             string    `json:"id" solr:"id"`
	UserID         int       `json:"user_id" solr:"user_id"`
	Type           string    `json:"type" solr:"type"`           // buy, sell
	Status         string    `json:"status" solr:"status"`        // pending, executed, cancelled, failed
	OrderKind      string    `json:"order_kind" solr:"order_kind"` // market, limit
	CryptoSymbol   string    `json:"crypto_symbol" solr:"crypto_symbol"`
	CryptoName     string    `json:"crypto_name" solr:"crypto_name"`
	Quantity       string    `json:"quantity" solr:"quantity"`
	Price          string    `json:"price" solr:"price"`
	TotalAmount    string    `json:"total_amount" solr:"total_amount"`
	Fee            string    `json:"fee" solr:"fee"`
	CreatedAt      time.Time `json:"created_at" solr:"created_at"`
	ExecutedAt     *time.Time `json:"executed_at,omitempty" solr:"executed_at"`
	UpdatedAt      time.Time `json:"updated_at" solr:"updated_at"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty" solr:"cancelled_at"`
	ErrorMessage   string    `json:"error_message,omitempty" solr:"error_message"`
	
	// Searchable text fields
	SearchText     string    `json:"search_text" solr:"search_text"`
}

// SearchResult represents a search result with additional metadata
type OrderSearchResult struct {
	Order
	Score        float64 `json:"score"`
	MatchType    string  `json:"match_type"`
	Highlighting map[string][]string `json:"highlighting,omitempty"`
}

// OrderPagination represents pagination information for orders
type OrderPagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int64 `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// Filter represents available search filters for orders
type OrderFilter struct {
	Statuses      []StatusFilter      `json:"statuses"`
	Types         []TypeFilter         `json:"types"`
	OrderKinds   []OrderKindFilter    `json:"order_kinds"`
	CryptoSymbols []CryptoSymbolFilter `json:"crypto_symbols"`
	SortOptions   []OrderSortOption    `json:"sort_options"`
}

// StatusFilter represents a status filter with count
type StatusFilter struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
	Label string `json:"label"`
}

// TypeFilter represents a type filter with count
type TypeFilter struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
	Label string `json:"label"`
}

// OrderKindFilter represents an order kind filter with count
type OrderKindFilter struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
	Label string `json:"label"`
}

// CryptoSymbolFilter represents a crypto symbol filter with count
type CryptoSymbolFilter struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
	Label string `json:"label"`
}

// OrderSortOption represents a sorting option for orders
type OrderSortOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Facets represents faceted search results
type OrderFacets struct {
	Statuses      map[string]int64 `json:"statuses"`
	Types         map[string]int64 `json:"types"`
	OrderKinds    map[string]int64 `json:"order_kinds"`
	CryptoSymbols map[string]int64 `json:"crypto_symbols"`
}

// OrderQueryInfo represents query execution information for orders
type OrderQueryInfo struct {
	Query           string `json:"query"`
	ExecutionTimeMS int64  `json:"execution_time_ms"`
	CacheHit        bool   `json:"cache_hit"`
	TotalFound      int64  `json:"total_found"`
}

// SearchResultsResponse represents the complete search response
type OrderSearchResultsResponse struct {
	Results    []OrderSearchResult `json:"results"`
	Pagination OrderPagination      `json:"pagination"`
	Facets     OrderFacets          `json:"facets"`
	QueryInfo  OrderQueryInfo       `json:"query_info"`
}

// Validation methods

// IsValidSort checks if the sort option is valid
func IsValidOrderSort(sort string) bool {
	validSorts := map[string]bool{
		"":                true, // default
		"created_at_desc": true,
		"created_at_asc":  true,
		"updated_at_desc": true,
		"updated_at_asc":  true,
		"executed_at_desc": true,
		"executed_at_asc":  true,
		"total_amount_desc": true,
		"total_amount_asc":  true,
		"price_desc":       true,
		"price_asc":        true,
		"quantity_desc":    true,
		"quantity_asc":     true,
	}
	return validSorts[sort]
}

// IsValidStatus checks if the status is valid
func IsValidStatus(status string) bool {
	validStatuses := map[string]bool{
		"pending":   true,
		"executed":  true,
		"cancelled": true,
		"failed":    true,
	}
	return validStatuses[status]
}

// IsValidOrderType checks if the order type is valid
func IsValidOrderType(orderType string) bool {
	validTypes := map[string]bool{
		"buy":  true,
		"sell": true,
	}
	return validTypes[orderType]
}

// IsValidOrderKind checks if the order kind is valid
func IsValidOrderKind(kind string) bool {
	validKinds := map[string]bool{
		"market": true,
		"limit":  true,
	}
	return validKinds[kind]
}

// GetDefaultOrderFilters returns the default filter configuration for orders
func GetDefaultOrderFilters() OrderFilter {
	return OrderFilter{
		Statuses: []StatusFilter{
			{Value: "pending", Label: "Pending", Count: 0},
			{Value: "executed", Label: "Executed", Count: 0},
			{Value: "cancelled", Label: "Cancelled", Count: 0},
			{Value: "failed", Label: "Failed", Count: 0},
		},
		Types: []TypeFilter{
			{Value: "buy", Label: "Buy Orders", Count: 0},
			{Value: "sell", Label: "Sell Orders", Count: 0},
		},
		OrderKinds: []OrderKindFilter{
			{Value: "market", Label: "Market Orders", Count: 0},
			{Value: "limit", Label: "Limit Orders", Count: 0},
		},
		SortOptions: []OrderSortOption{
			{Value: "created_at_desc", Label: "Newest First"},
			{Value: "created_at_asc", Label: "Oldest First"},
			{Value: "updated_at_desc", Label: "Recently Updated"},
			{Value: "total_amount_desc", Label: "Highest Amount"},
			{Value: "total_amount_asc", Label: "Lowest Amount"},
			{Value: "price_desc", Label: "Highest Price"},
			{Value: "price_asc", Label: "Lowest Price"},
		},
	}
}

