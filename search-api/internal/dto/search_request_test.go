package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions for creating pointers
func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func TestSearchRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   SearchRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid basic request",
			request: SearchRequest{
				Query: "bitcoin",
				Page:  1,
				Limit: 20,
			},
			wantError: false,
		},
		{
			name: "valid empty query",
			request: SearchRequest{
				Query: "",
				Page:  1,
				Limit: 20,
			},
			wantError: false,
		},
		{
			name: "invalid page - zero",
			request: SearchRequest{
				Query: "bitcoin",
				Page:  0,
				Limit: 20,
			},
			wantError: true,
			errorMsg:  "page must be greater than 0",
		},
		{
			name: "invalid page - negative",
			request: SearchRequest{
				Query: "bitcoin",
				Page:  -1,
				Limit: 20,
			},
			wantError: true,
			errorMsg:  "page must be greater than 0",
		},
		{
			name: "invalid limit - zero",
			request: SearchRequest{
				Query: "bitcoin",
				Page:  1,
				Limit: 0,
			},
			wantError: true,
			errorMsg:  "limit must be between 1 and 100",
		},
		{
			name: "invalid limit - too high",
			request: SearchRequest{
				Query: "bitcoin",
				Page:  1,
				Limit: 101,
			},
			wantError: true,
			errorMsg:  "limit must be between 1 and 100",
		},
		{
			name: "valid with filters",
			request: SearchRequest{
				Query:    "ethereum",
				Page:     1,
				Limit:    50,
				Category: []string{"DeFi", "Smart Contract Platform"},
				MinPrice: float64Ptr(10.0),
				MaxPrice: float64Ptr(1000.0),
				Sort:     "market_cap_desc",
			},
			wantError: false,
		},
		{
			name: "invalid price range",
			request: SearchRequest{
				Query:    "bitcoin",
				Page:     1,
				Limit:    20,
				MinPrice: float64Ptr(1000.0),
				MaxPrice: float64Ptr(100.0),
			},
			wantError: true,
			errorMsg:  "min_price cannot be greater than max_price",
		},
		{
			name: "invalid market cap range",
			request: SearchRequest{
				Query:        "bitcoin",
				Page:         1,
				Limit:        20,
				MinMarketCap: int64Ptr(int64(1000000000.0)),
				MaxMarketCap: int64Ptr(int64(100000000.0)),
			},
			wantError: true,
			errorMsg:  "min_market_cap cannot be greater than max_market_cap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSearchRequest_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		request        SearchRequest
		expectedPage   int
		expectedLimit  int
		expectedSort   string
	}{
		{
			name:          "empty request",
			request:       SearchRequest{},
			expectedPage:  1,
			expectedLimit: 20,
			expectedSort:  "market_cap_desc",
		},
		{
			name: "partial request",
			request: SearchRequest{
				Query: "bitcoin",
			},
			expectedPage:  1,
			expectedLimit: 20,
			expectedSort:  "market_cap_desc",
		},
		{
			name: "request with values",
			request: SearchRequest{
				Query: "ethereum",
				Page:  3,
				Limit: 50,
				Sort:  "market_cap_desc",
			},
			expectedPage:  3,
			expectedLimit: 50,
			expectedSort:  "market_cap_desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.request.SetDefaults()
			assert.Equal(t, tt.expectedPage, tt.request.Page)
			assert.Equal(t, tt.expectedLimit, tt.request.Limit)
			assert.Equal(t, tt.expectedSort, tt.request.Sort)
		})
	}
}

func TestTrendingRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   TrendingRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid request",
			request: TrendingRequest{
				Period: "24h",
				Limit:  10,
			},
			wantError: false,
		},
		{
			name: "invalid period",
			request: TrendingRequest{
				Period: "invalid",
				Limit:  10,
			},
			wantError: true,
			errorMsg:  "invalid period: must be one of 1h, 24h, 7d, 30d",
		},
		{
			name: "invalid limit - zero",
			request: TrendingRequest{
				Period: "24h",
				Limit:  0,
			},
			wantError: true,
			errorMsg:  "limit must be between 1 and 50",
		},
		{
			name: "invalid limit - too high",
			request: TrendingRequest{
				Period: "24h",
				Limit:  51,
			},
			wantError: true,
			errorMsg:  "limit must be between 1 and 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTrendingRequest_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		request        TrendingRequest
		expectedPeriod string
		expectedLimit  int
	}{
		{
			name:           "empty request",
			request:        TrendingRequest{},
			expectedPeriod: "24h",
			expectedLimit:  10,
		},
		{
			name: "partial request",
			request: TrendingRequest{
				Period: "1h",
			},
			expectedPeriod: "1h",
			expectedLimit:  10,
		},
		{
			name: "complete request",
			request: TrendingRequest{
				Period: "7d",
				Limit:  25,
			},
			expectedPeriod: "7d",
			expectedLimit:  25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.request.SetDefaults()
			assert.Equal(t, tt.expectedPeriod, tt.request.Period)
			assert.Equal(t, tt.expectedLimit, tt.request.Limit)
		})
	}
}

func TestSuggestionRequest_SetDefaults(t *testing.T) {
	tests := []struct {
		name          string
		request       SuggestionRequest
		expectedLimit int
	}{
		{
			name: "empty limit",
			request: SuggestionRequest{
				Query: "bit",
			},
			expectedLimit: 5,
		},
		{
			name: "with limit",
			request: SuggestionRequest{
				Query: "eth",
				Limit: 5,
			},
			expectedLimit: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.request.SetDefaults()
			assert.Equal(t, tt.expectedLimit, tt.request.Limit)
		})
	}
}

func TestSearchRequest_ToSolrParams(t *testing.T) {
	tests := []struct {
		name     string
		request  SearchRequest
		expected map[string]interface{}
	}{
		{
			name: "basic request",
			request: SearchRequest{
				Query: "bitcoin",
				Page:  1,
				Limit: 20,
				Sort:  "relevance",
			},
			expected: map[string]interface{}{
				"q":     "bitcoin",
				"start": 0,
				"rows":  20,
				"sort":  "score desc",
			},
		},
		{
			name: "request with filters",
			request: SearchRequest{
				Query:    "ethereum",
				Page:     2,
				Limit:    10,
				Category: []string{"DeFi"},
				MinPrice: float64Ptr(10.0),
				MaxPrice: float64Ptr(1000.0),
				Sort:     "market_cap_desc",
			},
			expected: map[string]interface{}{
				"q":     "ethereum",
				"start": 10,
				"rows":  10,
				"sort":  "market_cap desc",
				"fq":    []string{"categories:(\"DeFi\")", "current_price:[10 TO 1000]"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.request.ToSolrParams()

			// Check basic params
			assert.Equal(t, tt.expected["q"], params["q"])
			assert.Equal(t, tt.expected["start"], params["start"])
			assert.Equal(t, tt.expected["rows"], params["rows"])
			assert.Equal(t, tt.expected["sort"], params["sort"])

			// Check filter queries if present
			if expectedFq, ok := tt.expected["fq"]; ok {
				actualFq := params["fq"]
				assert.Equal(t, expectedFq, actualFq)
			}
		})
	}
}

func TestSearchRequest_BuildCacheKey(t *testing.T) {
	request1 := SearchRequest{
		Query: "bitcoin",
		Page:  1,
		Limit: 20,
		Sort:  "relevance",
	}

	request2 := SearchRequest{
		Query: "bitcoin",
		Page:  1,
		Limit: 20,
		Sort:  "relevance",
	}

	request3 := SearchRequest{
		Query: "ethereum",
		Page:  1,
		Limit: 20,
		Sort:  "relevance",
	}

	key1 := request1.BuildCacheKey()
	key2 := request2.BuildCacheKey()
	key3 := request3.BuildCacheKey()

	// Same requests should generate same keys
	assert.Equal(t, key1, key2)

	// Different requests should generate different keys
	assert.NotEqual(t, key1, key3)

	// Keys should not be empty
	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key3)
}

func TestSearchRequest_ComplexFiltering(t *testing.T) {
	request := SearchRequest{
		Query:           "defi",
		Page:            1,
		Limit:           50,
		Category:        []string{"DeFi", "Lending"},
		Platform:        "ethereum",
		Tags:            []string{"yield-farming", "governance"},
		MinPrice:        float64Ptr(1.0),
		MaxPrice:        float64Ptr(100.0),
		MinMarketCap:    int64Ptr(1000000),
		MaxMarketCap:    int64Ptr(1000000000),
		PriceChange24h:  "positive",
		IsTrending:      boolPtr(true),
		Sort:            "volume_desc",
	}

	err := request.Validate()
	require.NoError(t, err)

	params := request.ToSolrParams()

	// Check that all filters are properly converted
	assert.Equal(t, "defi", params["q"])
	assert.Equal(t, 0, params["start"])
	assert.Equal(t, 50, params["rows"])
	assert.Equal(t, "volume_24h desc", params["sort"])

	// Check filter queries
	fq, ok := params["fq"].([]string)
	require.True(t, ok)
	assert.Greater(t, len(fq), 0)

	// Should contain category filter
	found := false
	for _, filter := range fq {
		if filter == "categories:(\"DeFi\" OR \"Lending\")" {
			found = true
			break
		}
	}
	assert.True(t, found, "Category filter not found in fq")
}