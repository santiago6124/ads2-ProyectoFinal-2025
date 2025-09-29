package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	GetFacets(ctx context.Context) (*models.Filter, error)
	IndexCrypto(ctx context.Context, crypto *models.Crypto) error
	IndexCryptos(ctx context.Context, cryptos []models.Crypto) error
	UpdateTrendingScore(ctx context.Context, id string, score float32) error
	DeleteByID(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) error
	GetDocumentCount(ctx context.Context) (int64, error)
	Ping(ctx context.Context) error
}

// SearchResult represents the complete search result
type SearchResult struct {
	Results    []models.SearchResult
	Total      int64
	Facets     models.Facets
	QueryTime  time.Duration
}

// NewSolrRepository creates a new Solr repository
func NewSolrRepository(client *solr.Client) SearchRepository {
	return &SolrRepository{
		client: client,
	}
}

// Search performs a search query
func (r *SolrRepository) Search(ctx context.Context, req *dto.SearchRequest) (*SearchResult, error) {
	startTime := time.Now()

	// Build query parameters
	params := solr.BuildFromRequest(req)

	// Execute search
	response, err := r.client.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}

	queryTime := time.Duration(response.ResponseHeader.QTime) * time.Millisecond

	// Convert documents to search results
	results := make([]models.SearchResult, 0, len(response.Response.Docs))
	for _, doc := range response.Response.Docs {
		result := solr.DocToSearchResult(doc, getDocScore(doc))

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
			result.MatchType = determineMatchType(req.Query, result.Symbol, result.Name)
		}

		results = append(results, result)
	}

	// Extract facets
	facets := extractFacets(response.Facets)

	return &SearchResult{
		Results:   results,
		Total:     response.Response.NumFound,
		Facets:    facets,
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

// GetFacets gets available filters and facets
func (r *SolrRepository) GetFacets(ctx context.Context) (*models.Filter, error) {
	// Query with faceting enabled to get all available facets
	params := map[string]interface{}{
		"q":            "*:*",
		"rows":         0,
		"facet":        "true",
		"facet.field":  []string{"category", "platform"},
		"facet.range":  []string{"current_price", "market_cap"},
		"facet.range.start": 0,
		"facet.range.end":   10000,
		"facet.range.gap":   100,
		"fq":           "is_active:true",
	}

	response, err := r.client.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("facets query failed: %w", err)
	}

	return r.buildFiltersFromFacets(response.Facets), nil
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

func extractFacets(facetsData interface{}) models.Facets {
	facets := models.Facets{
		Categories:  make(map[string]int64),
		PriceRanges: make(map[string]int64),
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
			// Category facets
			if categoryData, exists := fieldsMap["category"]; exists {
				if categoryList, ok := categoryData.([]interface{}); ok {
					for i := 0; i < len(categoryList); i += 2 {
						if i+1 < len(categoryList) {
							category := fmt.Sprintf("%v", categoryList[i])
							count, _ := strconv.ParseInt(fmt.Sprintf("%v", categoryList[i+1]), 10, 64)
							facets.Categories[category] = count
						}
					}
				}
			}
		}
	}

	// Extract range facets
	if facetRanges, exists := facetsMap["facet_ranges"]; exists {
		if rangesMap, ok := facetRanges.(map[string]interface{}); ok {
			if priceData, exists := rangesMap["current_price"]; exists {
				if priceMap, ok := priceData.(map[string]interface{}); ok {
					if counts, exists := priceMap["counts"]; exists {
						if countsList, ok := counts.([]interface{}); ok {
							for i := 0; i < len(countsList); i += 2 {
								if i+1 < len(countsList) {
									priceRange := fmt.Sprintf("%v", countsList[i])
									count, _ := strconv.ParseInt(fmt.Sprintf("%v", countsList[i+1]), 10, 64)
									facets.PriceRanges[priceRange] = count
								}
							}
						}
					}
				}
			}
		}
	}

	return facets
}

func (r *SolrRepository) buildFiltersFromFacets(facetsData interface{}) *models.Filter {
	filters := models.GetDefaultFilters()

	if facetsData == nil {
		return &filters
	}

	// Extract counts from facets and update the default filters
	facets := extractFacets(facetsData)

	// Update category counts
	for i, categoryFilter := range filters.Categories {
		if count, exists := facets.Categories[categoryFilter.Value]; exists {
			filters.Categories[i].Count = count
		}
	}

	return &filters
}

func cryptoToSolrDoc(crypto *models.Crypto) map[string]interface{} {
	doc := map[string]interface{}{
		"id":                  crypto.ID,
		"symbol":              crypto.Symbol,
		"name":                crypto.Name,
		"description":         crypto.Description,
		"current_price":       crypto.CurrentPrice,
		"market_cap":          crypto.MarketCap,
		"market_cap_rank":     crypto.MarketCapRank,
		"volume_24h":          crypto.Volume24h,
		"price_change_24h":    crypto.PriceChange24h,
		"price_change_7d":     crypto.PriceChange7d,
		"price_change_30d":    crypto.PriceChange30d,
		"circulating_supply":  crypto.CirculatingSupply,
		"trending_score":      crypto.TrendingScore,
		"popularity_score":    crypto.PopularityScore,
		"category":            crypto.Category,
		"tags":                crypto.Tags,
		"platform":            crypto.Platform,
		"ath":                 crypto.ATH,
		"ath_date":            crypto.ATHDate.Format(time.RFC3339),
		"atl":                 crypto.ATL,
		"atl_date":            crypto.ATLDate.Format(time.RFC3339),
		"is_active":           crypto.IsActive,
		"is_trending":         crypto.IsTrending,
		"last_updated":        crypto.LastUpdated.Format(time.RFC3339),
		"indexed_at":          crypto.IndexedAt.Format(time.RFC3339),
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
	return "other"
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

// Import missing package
import "strings"