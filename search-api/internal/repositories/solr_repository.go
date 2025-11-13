package repositories

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"search-api/internal/dto"
	"search-api/internal/models"
	"search-api/internal/solr"
)

// SolrRepository handles Solr operations
type SolrRepository struct {
	client *solr.Client
}

// SearchRepository defines the search repository interface
type SearchRepository interface {
	Search(ctx context.Context, req *dto.SearchRequest) (*SearchResult, error)
	SearchTrending(ctx context.Context, period string, limit int) ([]models.TrendingCrypto, error)
	GetSuggestions(ctx context.Context, query string, limit int) ([]models.Suggestion, error)
	GetByID(ctx context.Context, id string) (*models.Crypto, error)
	GetOrderFilters(ctx context.Context) (*models.OrderFilter, error)
	IndexCrypto(ctx context.Context, crypto *models.Crypto) error
	IndexCryptos(ctx context.Context, cryptos []models.Crypto) error
	UpdateTrendingScore(ctx context.Context, id string, score float32) error
	DeleteByID(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) error
	GetDocumentCount(ctx context.Context) (int64, error)
	Ping(ctx context.Context) error
	// Order indexing methods
	IndexOrder(ctx context.Context, orderDoc map[string]interface{}) error
	DeleteOrderByID(ctx context.Context, orderID string) error
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, error)
}

// SearchResult represents the complete search result
// Note: Results and Facets are interface{} to support both Crypto and Order searches
type SearchResult struct {
	Results   []interface{} // Can be []models.SearchResult or []models.OrderSearchResult
	Total     int64
	Facets    interface{} // Can be models.Facets or models.OrderFacets
	QueryTime time.Duration
}

// NewSolrRepository creates a new Solr repository
func NewSolrRepository(client *solr.Client) SearchRepository {
	return &SolrRepository{
		client: client,
	}
}

// Search performs a search query for orders
func (r *SolrRepository) Search(ctx context.Context, req *dto.SearchRequest) (*SearchResult, error) {
	startTime := time.Now()

	// Build query parameters
	params := solr.BuildFromRequest(req)

	// Execute search
	response, err := r.client.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}

	// Convert documents to order search results
	results := make([]models.OrderSearchResult, 0, len(response.Response.Docs))
	for _, doc := range response.Response.Docs {
		result := docToOrderSearchResult(doc, getDocScore(doc))

		// Add highlighting if available
		if response.Highlighting != nil {
			if highlighting, ok := response.Highlighting.(map[string]interface{}); ok {
				if docHighlight, exists := highlighting[result.ID]; exists {
					if highlight, ok := docHighlight.(map[string]interface{}); ok {
						result.Highlighting = convertHighlighting(highlight)
					}
				}
			}
		}

		// Determine match type for search results
		if req.Query != "" {
			result.MatchType = determineOrderMatchType(req.Query, result.CryptoSymbol, result.CryptoName)
		}

		results = append(results, result)
	}

	// Extract facets for orders
	facets := extractOrderFacets(response.Facets)

	return &SearchResult{
		Results:   convertOrderResultsToInterface(results),
		Total:     response.Response.NumFound,
		Facets:    convertOrderFacetsToInterface(facets),
		QueryTime: time.Since(startTime),
	}, nil
}

// SearchTrending searches for trending cryptocurrencies
func (r *SolrRepository) SearchTrending(ctx context.Context, period string, limit int) ([]models.TrendingCrypto, error) {
	params := solr.BuildTrendingQuery(period, limit)

	response, err := r.client.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("trending search failed: %w", err)
	}

	trending := make([]models.TrendingCrypto, 0, len(response.Response.Docs))
	for i, doc := range response.Response.Docs {
		crypto := models.TrendingCrypto{
			Rank:           i + 1,
			ID:             getString(doc, "id"),
			Symbol:         getString(doc, "symbol"),
			Name:           getString(doc, "name"),
			CurrentPrice:   getFloat64(doc, "current_price"),
			PriceChange24h: getFloat64(doc, "price_change_24h"),
			Volume24h:      getInt64(doc, "volume_24h"),
			TrendingScore:  float32(getFloat64(doc, "trending_score")),
			// These would come from additional data sources in a real implementation
			SearchVolumeIncrease: calculateSearchVolumeIncrease(getFloat64(doc, "trending_score")),
			MentionsCount:        calculateMentionsCount(getFloat64(doc, "trending_score")),
		}
		trending = append(trending, crypto)
	}

	return trending, nil
}

// GetSuggestions gets autocomplete suggestions
func (r *SolrRepository) GetSuggestions(ctx context.Context, query string, limit int) ([]models.Suggestion, error) {
	suggestions, err := r.client.Suggest(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("suggestions query failed: %w", err)
	}

	return suggestions, nil
}

// GetByID gets a cryptocurrency by ID
func (r *SolrRepository) GetByID(ctx context.Context, id string) (*models.Crypto, error) {
	params := map[string]interface{}{
		"q":    fmt.Sprintf("id:\"%s\"", id),
		"rows": 1,
	}

	response, err := r.client.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get by ID failed: %w", err)
	}

	if len(response.Response.Docs) == 0 {
		return nil, fmt.Errorf("cryptocurrency with ID %s not found", id)
	}

	doc := response.Response.Docs[0]
	result := solr.DocToSearchResult(doc, 1.0)
	return &result.Crypto, nil
}

// GetOrderByID gets an order by ID from SolR
func (r *SolrRepository) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	params := map[string]interface{}{
		"q":    fmt.Sprintf("id:\"%s\"", orderID),
		"rows": 1,
	}

	response, err := r.client.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search for order %s: %w", orderID, err)
	}

	if len(response.Response.Docs) == 0 {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	result := docToOrderSearchResult(response.Response.Docs[0], 1.0)
	return &result.Order, nil
}

// GetOrderFilters retrieves available filters for orders
func (r *SolrRepository) GetOrderFilters(ctx context.Context) (*models.OrderFilter, error) {
	params := map[string]interface{}{
		"q":           "*:*",
		"rows":        0,
		"facet":       "true",
		"facet.field": []string{"status", "type", "order_kind", "crypto_symbol"},
	}

	filters := models.GetDefaultOrderFilters()

	response, err := r.client.Search(ctx, params)
	if err != nil {
		return &filters, nil
	}

	orderFacets := extractOrderFacets(response.Facets)

	for i, status := range filters.Statuses {
		if count, ok := orderFacets.Statuses[status.Value]; ok {
			filters.Statuses[i].Count = count
		}
	}

	for i, typ := range filters.Types {
		if count, ok := orderFacets.Types[typ.Value]; ok {
			filters.Types[i].Count = count
		}
	}

	for i, kind := range filters.OrderKinds {
		if count, ok := orderFacets.OrderKinds[kind.Value]; ok {
			filters.OrderKinds[i].Count = count
		}
	}

	filters.CryptoSymbols = make([]models.CryptoSymbolFilter, 0, len(orderFacets.CryptoSymbols))
	for symbol, count := range orderFacets.CryptoSymbols {
		filters.CryptoSymbols = append(filters.CryptoSymbols, models.CryptoSymbolFilter{
			Value: symbol,
			Label: strings.ToUpper(symbol),
			Count: count,
		})
		if len(filters.CryptoSymbols) >= 10 {
			break
		}
	}

	return &filters, nil
}

// IndexCrypto indexes a single cryptocurrency
func (r *SolrRepository) IndexCrypto(ctx context.Context, crypto *models.Crypto) error {
	// Set indexed timestamp
	crypto.IndexedAt = time.Now()

	docs := []interface{}{cryptoToSolrDoc(crypto)}

	if err := r.client.Update(ctx, docs); err != nil {
		return fmt.Errorf("failed to index crypto %s: %w", crypto.ID, err)
	}

	return r.client.Commit(ctx)
}

// IndexCryptos indexes multiple cryptocurrencies
func (r *SolrRepository) IndexCryptos(ctx context.Context, cryptos []models.Crypto) error {
	if len(cryptos) == 0 {
		return nil
	}

	docs := make([]interface{}, len(cryptos))
	now := time.Now()

	for i, crypto := range cryptos {
		crypto.IndexedAt = now
		docs[i] = cryptoToSolrDoc(&crypto)
	}

	if err := r.client.Update(ctx, docs); err != nil {
		return fmt.Errorf("failed to index %d cryptos: %w", len(cryptos), err)
	}

	return r.client.Commit(ctx)
}

// UpdateTrendingScore updates the trending score for a cryptocurrency
func (r *SolrRepository) UpdateTrendingScore(ctx context.Context, id string, score float32) error {
	// Use atomic update
	updateDoc := map[string]interface{}{
		"id":             id,
		"trending_score": map[string]interface{}{"set": score},
		"is_trending":    map[string]interface{}{"set": score > 50.0},
		"last_updated":   map[string]interface{}{"set": time.Now().Format(time.RFC3339)},
	}

	docs := []interface{}{updateDoc}

	if err := r.client.Update(ctx, docs); err != nil {
		return fmt.Errorf("failed to update trending score for %s: %w", id, err)
	}

	return r.client.Commit(ctx)
}

// DeleteByID deletes a cryptocurrency by ID
func (r *SolrRepository) DeleteByID(ctx context.Context, id string) error {
	if err := r.client.Delete(ctx, []string{id}); err != nil {
		return fmt.Errorf("failed to delete crypto %s: %w", id, err)
	}

	return r.client.Commit(ctx)
}

// DeleteAll deletes all documents
func (r *SolrRepository) DeleteAll(ctx context.Context) error {
	if err := r.client.DeleteByQuery(ctx, "*:*"); err != nil {
		return fmt.Errorf("failed to delete all documents: %w", err)
	}

	return r.client.Commit(ctx)
}

// GetDocumentCount returns the total number of indexed documents
func (r *SolrRepository) GetDocumentCount(ctx context.Context) (int64, error) {
	return r.client.GetDocumentCount(ctx)
}

// Ping checks if Solr is available
func (r *SolrRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx)
}

// IndexOrder indexes an order document in SolR
func (r *SolrRepository) IndexOrder(ctx context.Context, orderDoc map[string]interface{}) error {
	docs := []interface{}{orderDoc}

	if err := r.client.Update(ctx, docs); err != nil {
		return fmt.Errorf("failed to index order %v: %w", orderDoc["id"], err)
	}

	return r.client.Commit(ctx)
}

// DeleteOrderByID deletes an order from SolR by ID
func (r *SolrRepository) DeleteOrderByID(ctx context.Context, orderID string) error {
	if err := r.client.Delete(ctx, []string{orderID}); err != nil {
		return fmt.Errorf("failed to delete order %s: %w", orderID, err)
	}

	return r.client.Commit(ctx)
}

// Helper functions

func getDocScore(doc map[string]interface{}) float64 {
	if score, exists := doc["score"]; exists {
		if s, ok := score.(float64); ok {
			return s
		}
	}
	return 1.0
}

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

func convertHighlighting(highlight map[string]interface{}) map[string][]string {
	result := make(map[string][]string)
	for field, snippets := range highlight {
		if snippetList, ok := snippets.([]interface{}); ok {
			stringSnippets := make([]string, len(snippetList))
			for i, snippet := range snippetList {
				stringSnippets[i] = fmt.Sprintf("%v", snippet)
			}
			result[field] = stringSnippets
		}
	}
	return result
}

// extractOrderFacets extracts facets for orders
func extractOrderFacets(facetsData interface{}) models.OrderFacets {
	facets := models.OrderFacets{
		Statuses:      make(map[string]int64),
		Types:         make(map[string]int64),
		OrderKinds:    make(map[string]int64),
		CryptoSymbols: make(map[string]int64),
	}

	if facetsData == nil {
		return facets
	}

	facetsMap, ok := facetsData.(map[string]interface{})
	if !ok {
		return facets
	}

	// Extract field facets
	if facetFields, exists := facetsMap["facet_fields"]; exists {
		if fieldsMap, ok := facetFields.(map[string]interface{}); ok {
			// Status facets
			if statusData, exists := fieldsMap["status"]; exists {
				if statusList, ok := statusData.([]interface{}); ok {
					for i := 0; i < len(statusList); i += 2 {
						if i+1 < len(statusList) {
							status := fmt.Sprintf("%v", statusList[i])
							count, _ := strconv.ParseInt(fmt.Sprintf("%v", statusList[i+1]), 10, 64)
							facets.Statuses[status] = count
						}
					}
				}
			}

			// Type facets (buy/sell)
			if typeData, exists := fieldsMap["type"]; exists {
				if typeList, ok := typeData.([]interface{}); ok {
					for i := 0; i < len(typeList); i += 2 {
						if i+1 < len(typeList) {
							orderType := fmt.Sprintf("%v", typeList[i])
							count, _ := strconv.ParseInt(fmt.Sprintf("%v", typeList[i+1]), 10, 64)
							facets.Types[orderType] = count
						}
					}
				}
			}

			// Order kind facets (market/limit)
			if kindData, exists := fieldsMap["order_kind"]; exists {
				if kindList, ok := kindData.([]interface{}); ok {
					for i := 0; i < len(kindList); i += 2 {
						if i+1 < len(kindList) {
							kind := fmt.Sprintf("%v", kindList[i])
							count, _ := strconv.ParseInt(fmt.Sprintf("%v", kindList[i+1]), 10, 64)
							facets.OrderKinds[kind] = count
						}
					}
				}
			}

			// Crypto symbol facets
			if symbolData, exists := fieldsMap["crypto_symbol"]; exists {
				if symbolList, ok := symbolData.([]interface{}); ok {
					for i := 0; i < len(symbolList); i += 2 {
						if i+1 < len(symbolList) {
							symbol := fmt.Sprintf("%v", symbolList[i])
							count, _ := strconv.ParseInt(fmt.Sprintf("%v", symbolList[i+1]), 10, 64)
							facets.CryptoSymbols[symbol] = count
						}
					}
				}
			}
		}
	}

	return facets
}

func cryptoToSolrDoc(crypto *models.Crypto) map[string]interface{} {
	doc := map[string]interface{}{
		"id":                 crypto.ID,
		"symbol":             crypto.Symbol,
		"name":               crypto.Name,
		"description":        crypto.Description,
		"current_price":      crypto.CurrentPrice,
		"market_cap":         crypto.MarketCap,
		"market_cap_rank":    crypto.MarketCapRank,
		"volume_24h":         crypto.Volume24h,
		"price_change_24h":   crypto.PriceChange24h,
		"price_change_7d":    crypto.PriceChange7d,
		"price_change_30d":   crypto.PriceChange30d,
		"circulating_supply": crypto.CirculatingSupply,
		"trending_score":     crypto.TrendingScore,
		"popularity_score":   crypto.PopularityScore,
		"category":           crypto.Category,
		"tags":               crypto.Tags,
		"platform":           crypto.Platform,
		"ath":                crypto.ATH,
		"ath_date":           crypto.ATHDate.Format(time.RFC3339),
		"atl":                crypto.ATL,
		"atl_date":           crypto.ATLDate.Format(time.RFC3339),
		"is_active":          crypto.IsActive,
		"is_trending":        crypto.IsTrending,
		"last_updated":       crypto.LastUpdated.Format(time.RFC3339),
		"indexed_at":         crypto.IndexedAt.Format(time.RFC3339),
	}

	// Handle nullable fields
	if crypto.TotalSupply != nil {
		doc["total_supply"] = *crypto.TotalSupply
	}
	if crypto.MaxSupply != nil {
		doc["max_supply"] = *crypto.MaxSupply
	}

	return doc
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
	return "partial"
}

// determineOrderMatchType determines match type for order search results
func determineOrderMatchType(query, cryptoSymbol, cryptoName string) string {
	queryLower := strings.ToLower(query)
	symbolLower := strings.ToLower(cryptoSymbol)
	nameLower := strings.ToLower(cryptoName)

	if strings.HasPrefix(symbolLower, queryLower) {
		return "crypto_symbol"
	}
	if strings.HasPrefix(nameLower, queryLower) {
		return "crypto_name"
	}
	if strings.Contains(symbolLower, queryLower) {
		return "crypto_symbol_partial"
	}
	if strings.Contains(nameLower, queryLower) {
		return "crypto_name_partial"
	}
	return "partial"
}

// docToOrderSearchResult converts a SolR document to OrderSearchResult
func docToOrderSearchResult(doc map[string]interface{}, score float64) models.OrderSearchResult {
	order := models.Order{
		ID:           getString(doc, "id"),
		UserID:       int(getInt64(doc, "user_id")),
		Type:         getString(doc, "type"),
		Status:       getString(doc, "status"),
		OrderKind:    getString(doc, "order_kind"),
		CryptoSymbol: getString(doc, "crypto_symbol"),
		CryptoName:   getString(doc, "crypto_name"),
		Quantity:     getString(doc, "quantity_s"),
		Price:        getString(doc, "price_s"),
		TotalAmount:  getString(doc, "total_amount_s"),
		Fee:          getString(doc, "fee_s"),
	}

	// Parse dates
	if createdStr := getString(doc, "created_at"); createdStr != "" {
		if created, err := time.Parse(time.RFC3339, createdStr); err == nil {
			order.CreatedAt = created
		} else {
			order.CreatedAt = time.Now() // Fallback
		}
	} else {
		order.CreatedAt = time.Now()
	}

	if updatedStr := getString(doc, "updated_at"); updatedStr != "" {
		if updated, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			order.UpdatedAt = updated
		} else {
			order.UpdatedAt = time.Now() // Fallback
		}
	} else {
		order.UpdatedAt = time.Now()
	}

	if executedStr := getString(doc, "executed_at"); executedStr != "" {
		if executed, err := time.Parse(time.RFC3339, executedStr); err == nil {
			order.ExecutedAt = &executed
		}
	}

	if cancelledStr := getString(doc, "cancelled_at"); cancelledStr != "" {
		if cancelled, err := time.Parse(time.RFC3339, cancelledStr); err == nil {
			order.CancelledAt = &cancelled
		}
	}

	if errorMsg := getString(doc, "error_message"); errorMsg != "" {
		order.ErrorMessage = errorMsg
	}

	return models.OrderSearchResult{
		Order:     order,
		Score:     score,
		MatchType: "",
	}
}

// convertOrderResultsToInterface converts []OrderSearchResult to []interface{}
func convertOrderResultsToInterface(results []models.OrderSearchResult) []interface{} {
	interfaces := make([]interface{}, len(results))
	for i, r := range results {
		interfaces[i] = r
	}
	return interfaces
}

// convertOrderFacetsToInterface converts OrderFacets to interface{}
func convertOrderFacetsToInterface(facets models.OrderFacets) interface{} {
	return facets
}

func calculateSearchVolumeIncrease(trendingScore float64) string {
	// Simple calculation based on trending score
	increase := trendingScore * 10
	return fmt.Sprintf("%.0f%%", increase)
}

func calculateMentionsCount(trendingScore float64) int64 {
	// Simple calculation based on trending score
	return int64(trendingScore * 100)
}
