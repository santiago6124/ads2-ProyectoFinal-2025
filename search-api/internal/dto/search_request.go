package dto

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SearchRequest represents a search request for orders with all possible parameters
type SearchRequest struct {
	Query          string   `form:"q" json:"query"`
	Page           int      `form:"page" json:"page" binding:"min=1"`
	Limit          int      `form:"limit" json:"limit" binding:"min=1,max=100"`
	Sort           string   `form:"sort" json:"sort"`
	Status         []string `form:"status" json:"status"`               // pending, executed, cancelled, failed
	Type           []string `form:"type" json:"type"`                   // buy, sell
	OrderKind      []string `form:"order_kind" json:"order_kind"`       // market, limit
	CryptoSymbol   []string `form:"crypto_symbol" json:"crypto_symbol"` // BTC, ETH, etc
	UserID         *int     `form:"user_id" json:"user_id"`             // Filter by user ID
	MinTotalAmount *float64 `form:"min_total_amount" json:"min_total_amount" binding:"omitempty,min=0"`
	MaxTotalAmount *float64 `form:"max_total_amount" json:"max_total_amount" binding:"omitempty,min=0"`
	DateFrom       string   `form:"date_from" json:"date_from"` // ISO 8601 date
	DateTo         string   `form:"date_to" json:"date_to"`     // ISO 8601 date
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
		r.Sort = "created_at_desc" // Default: newest first
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

// Validate validates the search request for orders
func (r *SearchRequest) Validate() error {
	// Validate total amount range
	if r.MinTotalAmount != nil && r.MaxTotalAmount != nil && *r.MinTotalAmount > *r.MaxTotalAmount {
		return NewValidationError("min_total_amount cannot be greater than max_total_amount")
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending": true, "executed": true, "cancelled": true, "failed": true,
	}
	for _, status := range r.Status {
		if !validStatuses[status] {
			return NewValidationError("invalid status: " + status)
		}
	}

	// Validate type
	validTypes := map[string]bool{
		"buy": true, "sell": true,
	}
	for _, t := range r.Type {
		if !validTypes[t] {
			return NewValidationError("invalid type: " + t)
		}
	}

	// Validate order kind
	validOrderKinds := map[string]bool{
		"market": true, "limit": true,
	}
	for _, kind := range r.OrderKind {
		if !validOrderKinds[kind] {
			return NewValidationError("invalid order_kind: " + kind)
		}
	}

	// Validate sort
	validSorts := map[string]bool{
		"": true, "created_at_desc": true, "created_at_asc": true,
		"updated_at_desc": true, "updated_at_asc": true,
		"executed_at_desc": true, "executed_at_asc": true,
		"total_amount_desc": true, "total_amount_asc": true,
		"price_desc": true, "price_asc": true,
		"quantity_desc": true, "quantity_asc": true,
	}

	if !validSorts[r.Sort] {
		return NewValidationError("invalid sort option: " + r.Sort)
	}

	// Validate date range
	if r.DateFrom != "" && r.DateTo != "" {
		dateFrom, err1 := time.Parse(time.RFC3339, r.DateFrom)
		dateTo, err2 := time.Parse(time.RFC3339, r.DateTo)
		if err1 != nil || err2 != nil {
			return NewValidationError("date_from and date_to must be in ISO 8601 format")
		}
		if dateFrom.After(dateTo) {
			return NewValidationError("date_from cannot be after date_to")
		}
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
		params["q"] = r.Query
		params["defType"] = "edismax"
		params["qf"] = "crypto_name^10 crypto_symbol^8 search_text^2"
		params["pf"] = "crypto_name^15 crypto_symbol^10"
		params["ps"] = "2"
		params["qs"] = "1"
	}

	// Filters
	filters := make([]string, 0)

	// Status filter
	if len(r.Status) > 0 {
		filters = append(filters, fmt.Sprintf("status:(%s)", strings.Join(r.Status, " OR ")))
	}

	// Type filter
	if len(r.Type) > 0 {
		filters = append(filters, fmt.Sprintf("type:(%s)", strings.Join(r.Type, " OR ")))
	}

	// Order kind filter
	if len(r.OrderKind) > 0 {
		filters = append(filters, fmt.Sprintf("order_kind:(%s)", strings.Join(r.OrderKind, " OR ")))
	}

	// Crypto symbol filter
	if len(r.CryptoSymbol) > 0 {
		filters = append(filters, fmt.Sprintf("crypto_symbol:(%s)", strings.Join(r.CryptoSymbol, " OR ")))
	}

	// User ID filter
	if r.UserID != nil {
		filters = append(filters, fmt.Sprintf("user_id:%d", *r.UserID))
	}

	// Total amount filters
	if r.MinTotalAmount != nil || r.MaxTotalAmount != nil {
		minVal := "*"
		maxVal := "*"
		if r.MinTotalAmount != nil {
			minVal = strconv.FormatFloat(*r.MinTotalAmount, 'f', -1, 64)
		}
		if r.MaxTotalAmount != nil {
			maxVal = strconv.FormatFloat(*r.MaxTotalAmount, 'f', -1, 64)
		}
		filters = append(filters, fmt.Sprintf("total_amount_d:[%s TO %s]", minVal, maxVal))
	}

	// Date range filter (created_at)
	if r.DateFrom != "" || r.DateTo != "" {
		from := "*"
		to := "*"
		if r.DateFrom != "" {
			from = r.DateFrom
		}
		if r.DateTo != "" {
			to = r.DateTo
		}
		filters = append(filters, fmt.Sprintf("created_at:[%s TO %s]", from, to))
	}

	if len(filters) > 0 {
		params["fq"] = filters
	}

	// Sorting
	if r.Sort != "" {
		params["sort"] = convertOrderSortParam(r.Sort)
	} else {
		params["sort"] = "created_at desc"
	}

	// Pagination
	params["start"] = r.GetOffset()
	params["rows"] = r.Limit

	// Faceting for orders
	params["facet"] = "true"
	params["facet.field"] = []string{"status", "type", "order_kind", "crypto_symbol"}

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

	if len(r.Status) > 0 {
		parts = append(parts, "status:"+strings.Join(r.Status, ","))
	}
	if len(r.Type) > 0 {
		parts = append(parts, "type:"+strings.Join(r.Type, ","))
	}
	if len(r.OrderKind) > 0 {
		parts = append(parts, "kind:"+strings.Join(r.OrderKind, ","))
	}
	if len(r.CryptoSymbol) > 0 {
		parts = append(parts, "symbol:"+strings.Join(r.CryptoSymbol, ","))
	}
	if r.UserID != nil {
		parts = append(parts, "user:"+strconv.Itoa(*r.UserID))
	}
	if r.MinTotalAmount != nil {
		parts = append(parts, "min_total:"+strconv.FormatFloat(*r.MinTotalAmount, 'f', -1, 64))
	}
	if r.MaxTotalAmount != nil {
		parts = append(parts, "max_total:"+strconv.FormatFloat(*r.MaxTotalAmount, 'f', -1, 64))
	}
	if r.DateFrom != "" {
		parts = append(parts, "from:"+r.DateFrom)
	}
	if r.DateTo != "" {
		parts = append(parts, "to:"+r.DateTo)
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
		len(r.Status) == 0 &&
		len(r.Type) == 0 &&
		len(r.OrderKind) == 0 &&
		len(r.CryptoSymbol) == 0 &&
		r.UserID == nil &&
		r.MinTotalAmount == nil &&
		r.MaxTotalAmount == nil &&
		r.DateFrom == "" &&
		r.DateTo == ""
}

// GetCacheTTL returns the appropriate cache TTL based on request type
func (r *SearchRequest) GetCacheTTL() time.Duration {
	// Broader searches (empty filters) get a longer cache
	if r.IsEmpty() {
		return 10 * time.Minute
	}

	// Specific searches get shorter cache
	if r.Query != "" {
		return 5 * time.Minute
	}

	// Default cache duration
	return 3 * time.Minute
}

// convertOrderSortParam converts order sort values to Solr sort syntax
func convertOrderSortParam(sort string) string {
	switch sort {
	case "created_at_desc":
		return "created_at desc"
	case "created_at_asc":
		return "created_at asc"
	case "updated_at_desc":
		return "updated_at desc"
	case "updated_at_asc":
		return "updated_at asc"
	case "executed_at_desc":
		return "executed_at desc"
	case "executed_at_asc":
		return "executed_at asc"
	case "total_amount_desc":
		return "total_amount_d desc"
	case "total_amount_asc":
		return "total_amount_d asc"
	case "price_desc":
		return "price desc"
	case "price_asc":
		return "price asc"
	case "quantity_desc":
		return "quantity desc"
	case "quantity_asc":
		return "quantity asc"
	default:
		return "created_at desc"
	}
}
