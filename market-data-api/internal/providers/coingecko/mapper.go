package coingecko

import "time"

// SimplePriceResponse represents the response from CoinGecko's simple/price endpoint
type SimplePriceResponse struct {
	USD           float64 `json:"usd"`
	USD24hVol     float64 `json:"usd_24h_vol,omitempty"`
	USD24hChange  float64 `json:"usd_24h_change,omitempty"`
	USDMarketCap  float64 `json:"usd_market_cap,omitempty"`
	LastUpdatedAt int64   `json:"last_updated_at,omitempty"`
}

// MarketChartResponse represents the response from CoinGecko's market_chart endpoint
type MarketChartResponse struct {
	Prices       [][2]float64 `json:"prices"`
	MarketCaps   [][2]float64 `json:"market_caps"`
	TotalVolumes [][2]float64 `json:"total_volumes"`
}

// CoinResponse represents the response from CoinGecko's coins/{id} endpoint
type CoinResponse struct {
	ID          string      `json:"id"`
	Symbol      string      `json:"symbol"`
	Name        string      `json:"name"`
	MarketData  MarketData  `json:"market_data"`
	LastUpdated string      `json:"last_updated"`
}

// MarketData represents market data from CoinGecko
type MarketData struct {
	CurrentPrice                     CurrencyData `json:"current_price"`
	MarketCap                        CurrencyData `json:"market_cap"`
	FullyDilutedValuation           CurrencyData `json:"fully_diluted_valuation"`
	TotalVolume                     CurrencyData `json:"total_volume"`
	High24h                         CurrencyData `json:"high_24h"`
	Low24h                          CurrencyData `json:"low_24h"`
	PriceChange24h                  float64      `json:"price_change_24h"`
	PriceChangePercentage24h        float64      `json:"price_change_percentage_24h"`
	PriceChangePercentage7d         float64      `json:"price_change_percentage_7d"`
	PriceChangePercentage14d        float64      `json:"price_change_percentage_14d"`
	PriceChangePercentage30d        float64      `json:"price_change_percentage_30d"`
	PriceChangePercentage60d        float64      `json:"price_change_percentage_60d"`
	PriceChangePercentage200d       float64      `json:"price_change_percentage_200d"`
	PriceChangePercentage1y         float64      `json:"price_change_percentage_1y"`
	MarketCapChange24h              float64      `json:"market_cap_change_24h"`
	MarketCapChangePercentage24h    float64      `json:"market_cap_change_percentage_24h"`
	ATH                             CurrencyData `json:"ath"`
	ATHChangePercentage             CurrencyData `json:"ath_change_percentage"`
	ATHDate                         CurrencyDate `json:"ath_date"`
	ATL                             CurrencyData `json:"atl"`
	ATLChangePercentage             CurrencyData `json:"atl_change_percentage"`
	ATLDate                         CurrencyDate `json:"atl_date"`
	CirculatingSupply               float64      `json:"circulating_supply"`
	TotalSupply                     float64      `json:"total_supply"`
	MaxSupply                       float64      `json:"max_supply"`
	LastUpdated                     string       `json:"last_updated"`
}

// CurrencyData represents price data in different currencies
type CurrencyData struct {
	USD float64 `json:"usd"`
	EUR float64 `json:"eur,omitempty"`
	GBP float64 `json:"gbp,omitempty"`
	JPY float64 `json:"jpy,omitempty"`
	BTC float64 `json:"btc,omitempty"`
	ETH float64 `json:"eth,omitempty"`
}

// CurrencyDate represents date data in different currencies
type CurrencyDate struct {
	USD string `json:"usd"`
	EUR string `json:"eur,omitempty"`
	GBP string `json:"gbp,omitempty"`
	JPY string `json:"jpy,omitempty"`
	BTC string `json:"btc,omitempty"`
	ETH string `json:"eth,omitempty"`
}

// ErrorResponse represents error responses from CoinGecko
type ErrorResponse struct {
	Error string `json:"error"`
}

// CoinListResponse represents the response from CoinGecko's coins/list endpoint
type CoinListResponse struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// ExchangeTickerResponse represents ticker data from an exchange
type ExchangeTickerResponse struct {
	Name             string  `json:"name"`
	Base             string  `json:"base"`
	Target           string  `json:"target"`
	Market           Market  `json:"market"`
	Last             float64 `json:"last"`
	Volume           float64 `json:"volume"`
	ConvertedLast    CurrencyData `json:"converted_last"`
	ConvertedVolume  CurrencyData `json:"converted_volume"`
	TrustScore       string  `json:"trust_score"`
	BidAskSpreadPercentage float64 `json:"bid_ask_spread_percentage"`
	Timestamp        string  `json:"timestamp"`
	LastTradedAt     string  `json:"last_traded_at"`
	LastFetchAt      string  `json:"last_fetch_at"`
	IsAnomaly        bool    `json:"is_anomaly"`
	IsStale          bool    `json:"is_stale"`
	TradeURL         string  `json:"trade_url"`
	TokenInfoURL     string  `json:"token_info_url"`
	CoinID           string  `json:"coin_id"`
	TargetCoinID     string  `json:"target_coin_id"`
}

// Market represents market information
type Market struct {
	Name                string `json:"name"`
	Identifier          string `json:"identifier"`
	HasTradingIncentive bool   `json:"has_trading_incentive"`
}

// GlobalDataResponse represents global cryptocurrency data
type GlobalDataResponse struct {
	Data GlobalData `json:"data"`
}

// GlobalData represents global market data
type GlobalData struct {
	ActiveCryptocurrencies          int                    `json:"active_cryptocurrencies"`
	UpcomingICOs                    int                    `json:"upcoming_icos"`
	OngoingICOs                     int                    `json:"ongoing_icos"`
	EndedICOs                       int                    `json:"ended_icos"`
	Markets                         int                    `json:"markets"`
	TotalMarketCap                  map[string]float64     `json:"total_market_cap"`
	TotalVolume                     map[string]float64     `json:"total_volume"`
	MarketCapPercentage             map[string]float64     `json:"market_cap_percentage"`
	MarketCapChangePercentage24hUSD float64                `json:"market_cap_change_percentage_24h_usd"`
	UpdatedAt                       int64                  `json:"updated_at"`
}

// TrendingResponse represents trending coins response
type TrendingResponse struct {
	Coins []TrendingCoin `json:"coins"`
	NFTs  []TrendingNFT  `json:"nfts"`
}

// TrendingCoin represents a trending coin
type TrendingCoin struct {
	Item TrendingCoinItem `json:"item"`
}

// TrendingCoinItem represents trending coin details
type TrendingCoinItem struct {
	ID                 string  `json:"id"`
	CoinID             int     `json:"coin_id"`
	Name               string  `json:"name"`
	Symbol             string  `json:"symbol"`
	MarketCapRank      int     `json:"market_cap_rank"`
	Thumb              string  `json:"thumb"`
	Small              string  `json:"small"`
	Large              string  `json:"large"`
	Slug               string  `json:"slug"`
	PriceBTC           float64 `json:"price_btc"`
	Score              int     `json:"score"`
}

// TrendingNFT represents a trending NFT
type TrendingNFT struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Symbol             string  `json:"symbol"`
	Thumb              string  `json:"thumb"`
	NFTHashing         string  `json:"nft_hashing"`
	NativeCurrency     string  `json:"native_currency"`
	FloorPriceInNativeCurrency float64 `json:"floor_price_in_native_currency"`
	FloorPrice24hPercentageChange float64 `json:"floor_price_24h_percentage_change"`
}

// ExchangeRatesResponse represents exchange rates response
type ExchangeRatesResponse struct {
	Rates map[string]ExchangeRate `json:"rates"`
}

// ExchangeRate represents an exchange rate
type ExchangeRate struct {
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
	Type  string  `json:"type"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Coins      []SearchCoin     `json:"coins"`
	Exchanges  []SearchExchange `json:"exchanges"`
	ICOs       []SearchICO      `json:"icos"`
	Categories []SearchCategory `json:"categories"`
	NFTs       []SearchNFT      `json:"nfts"`
}

// SearchCoin represents a coin in search results
type SearchCoin struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	APISymbol         string `json:"api_symbol"`
	Symbol            string `json:"symbol"`
	MarketCapRank     int    `json:"market_cap_rank"`
	Thumb             string `json:"thumb"`
	Large             string `json:"large"`
}

// SearchExchange represents an exchange in search results
type SearchExchange struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	MarketType        string `json:"market_type"`
	Thumb             string `json:"thumb"`
	Large             string `json:"large"`
}

// SearchICO represents an ICO in search results
type SearchICO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Thumb  string `json:"thumb"`
	Large  string `json:"large"`
}

// SearchCategory represents a category in search results
type SearchCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SearchNFT represents an NFT in search results
type SearchNFT struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Thumb  string `json:"thumb"`
}

// PingResponse represents the response from the ping endpoint
type PingResponse struct {
	GeckoSays string `json:"gecko_says"`
}

// CompanyHoldingsResponse represents company treasury holdings
type CompanyHoldingsResponse struct {
	TotalHoldings           float64            `json:"total_holdings"`
	TotalValueUSD           float64            `json:"total_value_usd"`
	MarketCapDominance      float64            `json:"market_cap_dominance"`
	Companies               []CompanyHolding   `json:"companies"`
}

// CompanyHolding represents a company's cryptocurrency holdings
type CompanyHolding struct {
	Name                string  `json:"name"`
	Symbol              string  `json:"symbol"`
	Country             string  `json:"country"`
	TotalHoldings       float64 `json:"total_holdings"`
	TotalEntryValueUSD  float64 `json:"total_entry_value_usd"`
	TotalCurrentValueUSD float64 `json:"total_current_value_usd"`
	PercentageOfTotalSupply float64 `json:"percentage_of_total_supply"`
}

// HistoricalDataResponse represents historical data for a specific date
type HistoricalDataResponse struct {
	ID          string     `json:"id"`
	Symbol      string     `json:"symbol"`
	Name        string     `json:"name"`
	MarketData  MarketData `json:"market_data"`
}

// OHLCResponse represents OHLC data
type OHLCResponse [][4]float64 // [timestamp, open, high, low, close]

// Helper methods for response mapping

// ToTimestamp converts CoinGecko timestamp (milliseconds) to time.Time
func ToTimestamp(ms float64) time.Time {
	return time.Unix(int64(ms)/1000, 0)
}

// ToMilliseconds converts time.Time to CoinGecko timestamp format
func ToMilliseconds(t time.Time) int64 {
	return t.Unix() * 1000
}

// ValidateSymbol checks if a symbol is valid for CoinGecko
func ValidateSymbol(symbol string) bool {
	if len(symbol) < 1 || len(symbol) > 20 {
		return false
	}
	// Add more validation logic as needed
	return true
}

// NormalizeInterval converts common interval formats to CoinGecko format
func NormalizeInterval(interval string) string {
	switch interval {
	case "1m", "1min", "1minute":
		return "1"
	case "5m", "5min", "5minutes":
		return "5"
	case "15m", "15min", "15minutes":
		return "15"
	case "30m", "30min", "30minutes":
		return "30"
	case "1h", "1hour", "hourly":
		return "hourly"
	case "1d", "1day", "daily":
		return "daily"
	case "1w", "1week", "weekly":
		return "weekly"
	default:
		return "hourly"
	}
}

// GetSupportedCurrencies returns list of supported currencies
func GetSupportedCurrencies() []string {
	return []string{
		"usd", "eur", "gbp", "jpy", "cny", "cad", "aud", "nzd", "chf", "sek",
		"nok", "dkk", "pln", "huf", "czk", "rub", "inr", "krw", "sgd", "hkd",
		"mxn", "brl", "try", "zar", "btc", "eth", "ltc", "bch", "bnb", "eos",
		"xrp", "xlm", "link", "dot", "yfi",
	}
}

// GetSupportedIntervals returns list of supported intervals
func GetSupportedIntervals() []string {
	return []string{"1", "5", "15", "30", "hourly", "daily", "weekly"}
}

// IsValidCurrency checks if a currency is supported
func IsValidCurrency(currency string) bool {
	supported := GetSupportedCurrencies()
	for _, c := range supported {
		if c == currency {
			return true
		}
	}
	return false
}