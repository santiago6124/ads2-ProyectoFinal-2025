package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrypto_Validate(t *testing.T) {
	tests := []struct {
		name      string
		crypto    Crypto
		wantError bool
	}{
		{
			name: "valid crypto",
			crypto: Crypto{
				ID:     "bitcoin",
				Symbol: "BTC",
				Name:   "Bitcoin",
			},
			wantError: false,
		},
		{
			name: "missing ID",
			crypto: Crypto{
				Symbol: "BTC",
				Name:   "Bitcoin",
			},
			wantError: true,
		},
		{
			name: "missing symbol",
			crypto: Crypto{
				ID:   "bitcoin",
				Name: "Bitcoin",
			},
			wantError: true,
		},
		{
			name: "missing name",
			crypto: Crypto{
				ID:     "bitcoin",
				Symbol: "BTC",
			},
			wantError: true,
		},
		{
			name: "symbol too long",
			crypto: Crypto{
				ID:     "bitcoin",
				Symbol: "BTCVERYLONGSYMBOL",
				Name:   "Bitcoin",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.crypto.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilter_GetDefaultFilters(t *testing.T) {
	filters := GetDefaultFilters()

	assert.NotNil(t, filters)
	assert.NotEmpty(t, filters.Categories)
	assert.NotEmpty(t, filters.Platforms)
	assert.NotEmpty(t, filters.Tags)
	assert.NotEmpty(t, filters.PriceRanges)
	assert.NotEmpty(t, filters.MarketCapRanges)

	// Check that common categories are present
	categoryNames := make([]string, len(filters.Categories))
	for i, cat := range filters.Categories {
		categoryNames[i] = cat.Name
	}
	assert.Contains(t, categoryNames, "Currency")
	assert.Contains(t, categoryNames, "DeFi")
	assert.Contains(t, categoryNames, "Smart Contract Platform")

	// Check that common platforms are present
	platformNames := make([]string, len(filters.Platforms))
	for i, platform := range filters.Platforms {
		platformNames[i] = platform.Name
	}
	assert.Contains(t, platformNames, "Ethereum")
	assert.Contains(t, platformNames, "Binance Smart Chain")
}

func TestTrendingCrypto_Validation(t *testing.T) {
	trending := TrendingCrypto{
		ID:                   "bitcoin",
		Symbol:               "BTC",
		Name:                 "Bitcoin",
		Rank:                 1,
		TrendingScore:        85.5,
		SearchVolumeIncrease: "150%",
		MentionsCount:        1250,
	}

	assert.Equal(t, "bitcoin", trending.ID)
	assert.Equal(t, "BTC", trending.Symbol)
	assert.Equal(t, 1, trending.Rank)
	assert.Equal(t, float32(85.5), trending.TrendingScore)
}

func TestSuggestion_Validation(t *testing.T) {
	suggestion := Suggestion{
		ID:     "ethereum",
		Symbol: "ETH",
		Name:   "Ethereum",
		Score:  95.0,
		Type:   "cryptocurrency",
	}

	assert.Equal(t, "ethereum", suggestion.ID)
	assert.Equal(t, "ETH", suggestion.Symbol)
	assert.Equal(t, "Ethereum", suggestion.Name)
	assert.Equal(t, float32(95.0), suggestion.Score)
	assert.Equal(t, "cryptocurrency", suggestion.Type)
}

func TestPagination_Validation(t *testing.T) {
	pagination := Pagination{
		Total:      1000,
		Page:       5,
		Limit:      20,
		TotalPages: 50,
		HasNext:    true,
		HasPrev:    true,
	}

	assert.Equal(t, int64(1000), pagination.Total)
	assert.Equal(t, 5, pagination.Page)
	assert.Equal(t, 20, pagination.Limit)
	assert.Equal(t, int64(50), pagination.TotalPages)
	assert.True(t, pagination.HasNext)
	assert.True(t, pagination.HasPrev)
}

func TestQueryInfo_Validation(t *testing.T) {
	queryInfo := QueryInfo{
		Query:           "bitcoin",
		ExecutionTimeMS: 150,
		CacheHit:        true,
		TotalFound:      100,
	}

	assert.Equal(t, "bitcoin", queryInfo.Query)
	assert.Equal(t, int64(150), queryInfo.ExecutionTimeMS)
	assert.True(t, queryInfo.CacheHit)
	assert.Equal(t, int64(100), queryInfo.TotalFound)
}

func TestCrypto_FullValidation(t *testing.T) {
	now := time.Now()
	totalSupply := int64(21000000)
	maxSupply := int64(21000000)
	crypto := Crypto{
		ID:                    "bitcoin",
		Symbol:                "BTC",
		Name:                  "Bitcoin",
		Description:           "The first cryptocurrency",
		CurrentPrice:          50000.0,
		MarketCap:             950000000000,
		Volume24h:             30000000000,
		PriceChange24h:        5.2,
		PriceChangePercent24h: 0.1,
		MarketCapRank:         1,
		CirculatingSupply:     19000000,
		TotalSupply:           &totalSupply,
		MaxSupply:             &maxSupply,
		ATH:                   69000.0,
		ATL:                   0.05,
		Category:              []string{"Currency", "Store of Value"},
		Tags:                  []string{"pow", "sha-256"},
		Platform:              "",
		LastUpdated:           now,
		TrendingScore:         85.5,
		IsTrending:            true,
	}

	err := crypto.Validate()
	require.NoError(t, err)

	// Test edge cases
	crypto.Symbol = ""
	err = crypto.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbol is required")

	crypto.Symbol = "BTC"
	crypto.CurrentPrice = -100
	err = crypto.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "current price cannot be negative")

	crypto.CurrentPrice = 50000
	crypto.MarketCap = -1000
	err = crypto.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "market cap cannot be negative")

	crypto.MarketCap = 950000000000.0
	crypto.Volume24h = -500
	err = crypto.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "volume 24h cannot be negative")
}

func TestFilterCategory_Count(t *testing.T) {
	category := FilterCategory{
		Name:  "DeFi",
		Count: 150,
	}

	assert.Equal(t, "DeFi", category.Name)
	assert.Equal(t, int64(150), category.Count)
}

func TestFilterPlatform_Count(t *testing.T) {
	platform := FilterPlatform{
		Name:  "Ethereum",
		Count: 2500,
	}

	assert.Equal(t, "Ethereum", platform.Name)
	assert.Equal(t, int64(2500), platform.Count)
}

func TestFilterTag_Count(t *testing.T) {
	tag := FilterTag{
		Name:  "pos",
		Count: 300,
	}

	assert.Equal(t, "pos", tag.Name)
	assert.Equal(t, int64(300), tag.Count)
}

func TestPriceRange_Validation(t *testing.T) {
	priceRange := PriceRange{
		Label: "$1 - $10",
		Min:   1.0,
		Max:   10.0,
		Count: 500,
	}

	assert.Equal(t, "$1 - $10", priceRange.Label)
	assert.Equal(t, 1.0, priceRange.Min)
	assert.Equal(t, 10.0, priceRange.Max)
	assert.Equal(t, int64(500), priceRange.Count)
	assert.True(t, priceRange.Min < priceRange.Max)
}

func TestMarketCapRange_Validation(t *testing.T) {
	mcRange := MarketCapRange{
		Label: "$1M - $10M",
		Min:   1000000,
		Max:   10000000,
		Count: 200,
	}

	assert.Equal(t, "$1M - $10M", mcRange.Label)
	assert.Equal(t, int64(1000000), mcRange.Min)
	assert.Equal(t, int64(10000000), mcRange.Max)
	assert.Equal(t, int64(200), mcRange.Count)
	assert.True(t, mcRange.Min < mcRange.Max)
}