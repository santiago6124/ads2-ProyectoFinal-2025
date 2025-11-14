package solr

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"search-api/internal/dto"
)

// QueryBuilder helps build complex Solr queries
type QueryBuilder struct {
	query   string
	filters []string
	sort    string
	facets  []FacetConfig
	fields  []string
	start   int
	rows    int
	params  map[string]interface{}
}

// FacetConfig represents a facet configuration
type FacetConfig struct {
	Field   string
	Type    string // field, range, query
	Options map[string]interface{}
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query:   "*:*",
		filters: make([]string, 0),
		facets:  make([]FacetConfig, 0),
		fields:  make([]string, 0),
		params:  make(map[string]interface{}),
	}
}

// SetQuery sets the main query
func (qb *QueryBuilder) SetQuery(query string) *QueryBuilder {
	if query == "" {
		qb.query = "*:*"
	} else {
		qb.query = query
	}
	return qb
}

// AddFilter adds a filter query
func (qb *QueryBuilder) AddFilter(filter string) *QueryBuilder {
	if filter != "" {
		qb.filters = append(qb.filters, filter)
	}
	return qb
}

// AddFilters adds multiple filter queries
func (qb *QueryBuilder) AddFilters(filters []string) *QueryBuilder {
	for _, filter := range filters {
		qb.AddFilter(filter)
	}
	return qb
}

// SetSort sets the sort parameter
func (qb *QueryBuilder) SetSort(sort string) *QueryBuilder {
	qb.sort = sort
	return qb
}

// SetFields sets the fields to return
func (qb *QueryBuilder) SetFields(fields ...string) *QueryBuilder {
	qb.fields = fields
	return qb
}

// SetPagination sets pagination parameters
func (qb *QueryBuilder) SetPagination(start, rows int) *QueryBuilder {
	qb.start = start
	qb.rows = rows
	return qb
}

// AddFieldFacet adds a field facet
func (qb *QueryBuilder) AddFieldFacet(field string, options map[string]interface{}) *QueryBuilder {
	facet := FacetConfig{
		Field:   field,
		Type:    "field",
		Options: options,
	}
	qb.facets = append(qb.facets, facet)
	return qb
}

// AddRangeFacet adds a range facet
func (qb *QueryBuilder) AddRangeFacet(field string, start, end, gap float64) *QueryBuilder {
	facet := FacetConfig{
		Field: field,
		Type:  "range",
		Options: map[string]interface{}{
			"start": start,
			"end":   end,
			"gap":   gap,
		},
	}
	qb.facets = append(qb.facets, facet)
	return qb
}

// SetParam sets a custom parameter
func (qb *QueryBuilder) SetParam(key string, value interface{}) *QueryBuilder {
	qb.params[key] = value
	return qb
}

// EnableDisMax enables DisMax query parser
func (qb *QueryBuilder) EnableDisMax(queryFields string, phraseFields string) *QueryBuilder {
	qb.SetParam("defType", "edismax")
	if queryFields != "" {
		qb.SetParam("qf", queryFields)
	}
	if phraseFields != "" {
		qb.SetParam("pf", phraseFields)
	}
	return qb
}

// EnableHighlighting enables result highlighting
func (qb *QueryBuilder) EnableHighlighting(fields ...string) *QueryBuilder {
	qb.SetParam("hl", "true")
	if len(fields) > 0 {
		qb.SetParam("hl.fl", strings.Join(fields, ","))
	}
	qb.SetParam("hl.simple.pre", "<mark>")
	qb.SetParam("hl.simple.post", "</mark>")
	qb.SetParam("hl.fragsize", 150)
	return qb
}

// Build builds the final query parameters
func (qb *QueryBuilder) Build() map[string]interface{} {
	params := make(map[string]interface{})

	// Copy custom parameters first
	for k, v := range qb.params {
		params[k] = v
	}

	// Main query
	params["q"] = qb.query

	// Filters
	if len(qb.filters) > 0 {
		params["fq"] = qb.filters
	}

	// Sort
	if qb.sort != "" {
		params["sort"] = qb.sort
	}

	// Fields
	if len(qb.fields) > 0 {
		params["fl"] = strings.Join(qb.fields, ",")
	}

	// Pagination
	if qb.start > 0 {
		params["start"] = qb.start
	}
	if qb.rows > 0 {
		params["rows"] = qb.rows
	}

	// Facets
	if len(qb.facets) > 0 {
		params["facet"] = "true"

		fieldFacets := make([]string, 0)
		rangeFacets := make([]string, 0)

		for _, facet := range qb.facets {
			switch facet.Type {
			case "field":
				fieldFacets = append(fieldFacets, facet.Field)
				// Add field-specific options
				for k, v := range facet.Options {
					params[fmt.Sprintf("f.%s.facet.%s", facet.Field, k)] = v
				}
			case "range":
				rangeFacets = append(rangeFacets, facet.Field)
				// Add range-specific options
				for k, v := range facet.Options {
					params[fmt.Sprintf("facet.range.%s", k)] = v
				}
			}
		}

		if len(fieldFacets) > 0 {
			params["facet.field"] = fieldFacets
		}
		if len(rangeFacets) > 0 {
			params["facet.range"] = rangeFacets
		}
	}

	// Default response format
	params["wt"] = "json"

	return params
}

// BuildFromRequest builds a query from SearchRequest for orders
func BuildFromRequest(req *dto.SearchRequest) map[string]interface{} {
	qb := NewQueryBuilder()

	// Main query
	if req.Query != "" {
		qb.SetQuery(req.Query)
		qb.EnableDisMax("crypto_name^10 crypto_symbol^8 search_text^2", "crypto_name^20 crypto_symbol^15")
		qb.EnableHighlighting("crypto_name", "crypto_symbol")
	} else {
		qb.SetQuery("*:*")
	}

	// Status filter
	if len(req.Status) > 0 {
		statusFilter := fmt.Sprintf("status:(%s)", strings.Join(req.Status, " OR "))
		qb.AddFilter(statusFilter)
	}

	// Type filter (buy/sell)
	if len(req.Type) > 0 {
		typeFilter := fmt.Sprintf("type:(%s)", strings.Join(req.Type, " OR "))
		qb.AddFilter(typeFilter)
	}

	// Order kind filter (market/limit)
	if len(req.OrderKind) > 0 {
		kindFilter := fmt.Sprintf("order_kind:(%s)", strings.Join(req.OrderKind, " OR "))
		qb.AddFilter(kindFilter)
	}

	// Crypto symbol filter
	if len(req.CryptoSymbol) > 0 {
		symbolFilter := fmt.Sprintf("crypto_symbol:(%s)", strings.Join(req.CryptoSymbol, " OR "))
		qb.AddFilter(symbolFilter)
	}

	// User ID filter
	if req.UserID != nil {
		qb.AddFilter(fmt.Sprintf("user_id:%d", *req.UserID))
	}

	// Total amount filters
	if req.MinTotalAmount != nil || req.MaxTotalAmount != nil {
		amountFilter := buildRangeFilter("total_amount_d", req.MinTotalAmount, req.MaxTotalAmount)
		qb.AddFilter(amountFilter)
	}

	// Date range filters
	if req.DateFrom != "" || req.DateTo != "" {
		dateFilter := buildDateRangeFilter("created_at", req.DateFrom, req.DateTo)
		if dateFilter != "" {
			qb.AddFilter(dateFilter)
		}
	}

	// Sort
	if req.Sort != "" {
		solrSort := convertOrderSortParam(req.Sort)
		qb.SetSort(solrSort)
	} else {
		// Default sort: newest first
		qb.SetSort("created_at desc")
	}

	// Pagination
	qb.SetPagination(req.GetOffset(), req.Limit)

	// Add facets for filtering
	qb.AddFieldFacet("status", map[string]interface{}{
		"mincount": 1,
		"limit":    10,
	})

	qb.AddFieldFacet("type", map[string]interface{}{
		"mincount": 1,
		"limit":    10,
	})

	// NOTE: order_kind field doesn't exist in Solr schema, removed to fix 400 errors
	// qb.AddFieldFacet("order_kind", map[string]interface{}{
	// 	"mincount": 1,
	// 	"limit":    10,
	// })

	qb.AddFieldFacet("crypto_symbol", map[string]interface{}{
		"mincount": 1,
		"limit":    20,
	})

	return qb.Build()
}

// BuildTrendingQuery builds a query for trending cryptocurrencies
func BuildTrendingQuery(period string, limit int) map[string]interface{} {
	qb := NewQueryBuilder()

	// Query for trending cryptos
	qb.SetQuery("*:*")
	qb.AddFilter("is_active:true")
	qb.AddFilter("is_trending:true")

	// Sort by trending score
	qb.SetSort("trending_score desc, market_cap desc")

	// Fields to return
	qb.SetFields(
		"id", "symbol", "name", "current_price", "price_change_24h",
		"volume_24h", "trending_score", "market_cap_rank",
	)

	// Pagination
	qb.SetPagination(0, limit)

	return qb.Build()
}

// BuildSuggestionQuery builds a query for autocomplete suggestions
func BuildSuggestionQuery(query string, limit int) map[string]interface{} {
	qb := NewQueryBuilder()

	// Build suggestion query
	escapedQuery := escapeQueryChars(query)
	suggestionQuery := fmt.Sprintf("(symbol:%s* OR name:%s*) OR (symbol_exact:\"%s\" OR name_exact:\"%s\")",
		strings.ToUpper(escapedQuery), escapedQuery, strings.ToUpper(query), query)

	qb.SetQuery(suggestionQuery)
	qb.AddFilter("is_active:true")

	// Boost exact matches
	qb.SetParam("bq", []string{
		fmt.Sprintf("symbol_exact:\"%s\"^10", strings.ToUpper(query)),
		fmt.Sprintf("name_exact:\"%s\"^5", query),
	})

	// Sort by relevance and market cap rank
	qb.SetSort("score desc, market_cap_rank asc")

	// Fields to return
	qb.SetFields("id", "symbol", "name", "current_price", "market_cap_rank")

	// Pagination
	qb.SetPagination(0, limit)

	return qb.Build()
}

// Helper functions

func buildRangeFilter(field string, min, max *float64) string {
	minVal := "*"
	maxVal := "*"

	if min != nil {
		minVal = strconv.FormatFloat(*min, 'f', -1, 64)
	}
	if max != nil {
		maxVal = strconv.FormatFloat(*max, 'f', -1, 64)
	}

	return fmt.Sprintf("%s:[%s TO %s]", field, minVal, maxVal)
}

func buildRangeFilterInt(field string, min, max *int64) string {
	minVal := "*"
	maxVal := "*"

	if min != nil {
		minVal = strconv.FormatInt(*min, 10)
	}
	if max != nil {
		maxVal = strconv.FormatInt(*max, 10)
	}

	return fmt.Sprintf("%s:[%s TO %s]", field, minVal, maxVal)
}

func convertSortParam(sort string) string {
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

	if solrSort, exists := sortMap[sort]; exists {
		return solrSort
	}

	return ""
}

// convertOrderSortParam converts order sort parameters to Solr sort syntax
func convertOrderSortParam(sort string) string {
	sortMap := map[string]string{
		"created_at_desc":   "created_at desc",
		"created_at_asc":    "created_at asc",
		"updated_at_desc":   "updated_at desc",
		"updated_at_asc":    "updated_at asc",
		"executed_at_desc":  "executed_at desc",
		"executed_at_asc":   "executed_at asc",
		"total_amount_desc": "total_amount_d desc",
		"total_amount_asc":  "total_amount_d asc",
		"price_desc":        "price desc",
		"price_asc":         "price asc",
		"quantity_desc":     "quantity desc",
		"quantity_asc":      "quantity asc",
	}

	if solrSort, exists := sortMap[sort]; exists {
		return solrSort
	}

	return "created_at desc" // Default sort
}

// buildDateRangeFilter builds a date range filter for Solr
func buildDateRangeFilter(field, dateFrom, dateTo string) string {
	fromStr := "*"
	toStr := "*"

	if dateFrom != "" {
		fromStr = dateFrom
	}
	if dateTo != "" {
		toStr = dateTo
	}

	if fromStr == "*" && toStr == "*" {
		return ""
	}

	return fmt.Sprintf("%s:[%s TO %s]", field, fromStr, toStr)
}

func escapeQueryChars(query string) string {
	// Escape Solr special characters
	specialChars := []string{"+", "-", "&&", "||", "!", "(", ")", "{", "}", "[", "]", "^", "\"", "~", "*", "?", ":", "\\"}

	escaped := query
	for _, char := range specialChars {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}

	return escaped
}

// AdvancedQueryBuilder provides more advanced query building capabilities
type AdvancedQueryBuilder struct {
	*QueryBuilder
}

// NewAdvancedQueryBuilder creates a new advanced query builder
func NewAdvancedQueryBuilder() *AdvancedQueryBuilder {
	return &AdvancedQueryBuilder{
		QueryBuilder: NewQueryBuilder(),
	}
}

// AddDateRangeFilter adds a date range filter
func (aqb *AdvancedQueryBuilder) AddDateRangeFilter(field string, from, to *time.Time) *AdvancedQueryBuilder {
	fromStr := "*"
	toStr := "*"

	if from != nil {
		fromStr = from.Format(time.RFC3339)
	}
	if to != nil {
		toStr = to.Format(time.RFC3339)
	}

	filter := fmt.Sprintf("%s:[%s TO %s]", field, fromStr, toStr)
	aqb.AddFilter(filter)
	return aqb
}

// AddBoostQuery adds a boost query
func (aqb *AdvancedQueryBuilder) AddBoostQuery(query string, boost float64) *AdvancedQueryBuilder {
	boostQuery := fmt.Sprintf("%s^%.2f", query, boost)

	if existingBq, exists := aqb.params["bq"]; exists {
		if bqSlice, ok := existingBq.([]string); ok {
			aqb.params["bq"] = append(bqSlice, boostQuery)
		} else {
			aqb.params["bq"] = []string{existingBq.(string), boostQuery}
		}
	} else {
		aqb.params["bq"] = boostQuery
	}

	return aqb
}

// SetMinimumMatch sets the minimum should match parameter for boolean queries
func (aqb *AdvancedQueryBuilder) SetMinimumMatch(mm string) *AdvancedQueryBuilder {
	aqb.SetParam("mm", mm)
	return aqb
}

// SetPhraseSlop sets the phrase slop for phrase queries
func (aqb *AdvancedQueryBuilder) SetPhraseSlop(slop int) *AdvancedQueryBuilder {
	aqb.SetParam("ps", slop)
	return aqb
}

// SetQuerySlop sets the query slop for proximity matching
func (aqb *AdvancedQueryBuilder) SetQuerySlop(slop int) *AdvancedQueryBuilder {
	aqb.SetParam("qs", slop)
	return aqb
}
