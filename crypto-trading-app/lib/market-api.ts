// Market Data API service for communicating with market-data-api microservice
import { config } from './config'

const MARKET_API_BASE_URL = config.marketApiUrl || "http://localhost:8004"

export interface PriceData {
  symbol: string
  name: string
  price: number
  change_24h: number
  market_cap: number
  volume: number
  timestamp: number
}

export interface PriceHistory {
  symbol: string
  history: Array<{
    timestamp: number
    price: number
  }>
}

export interface MarketStats {
  totalMarketCap: number
  totalVolume24h: number
  btcDominance: number
}

export interface TrendingCoin {
  id: string
  symbol: string
  name: string
  price: number
  change_24h: number
  market_cap: number
  volume: number
  rank: number
}

export interface MarketCategory {
  id: string
  name: string
  market_cap: number
  volume_24h: number
  change_24h: number
}

class MarketApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'MarketApiError'
  }
}

class MarketApiService {
  private baseURL: string

  constructor() {
    this.baseURL = MARKET_API_BASE_URL
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`
    
    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    }

    try {
      const response = await fetch(url, config)
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new MarketApiError(response.status, errorData.error || 'Request failed')
      }

      return await response.json()
    } catch (error) {
      if (error instanceof MarketApiError) {
        throw error
      }
      throw new MarketApiError(0, 'Network error')
    }
  }

  // Get current price for a specific symbol
  async getPrice(symbol: string): Promise<PriceData> {
    const response = await this.request<PriceData>(`/api/v1/prices/${symbol.toUpperCase()}`, {
      method: 'GET',
    })
    return response
  }

  // Get prices for multiple symbols
  async getPrices(symbols?: string[]): Promise<PriceData[]> {
    const endpoint = symbols 
      ? `/api/v1/prices?symbols=${symbols.join(',')}`
      : '/api/v1/prices'
    
    const response = await this.request<{ data: PriceData[] }>(endpoint, {
      method: 'GET',
    })
    return response.data
  }

  // Get all major cryptocurrency prices
  async getAllPrices(): Promise<PriceData[]> {
    const symbols = ['BTC', 'ETH', 'ADA', 'SOL', 'MATIC', 'AVAX', 'BNB', 'XRP', 'DOGE', 'DOT', 'LINK', 'UNI', 'LTC', 'BCH', 'ATOM', 'NEAR', 'ALGO', 'VET', 'ICP', 'FIL', 'TRX', 'ETC', 'XLM', 'MANA', 'SAND', 'AXS', 'CHZ', 'ENJ', 'BAT', 'ZEC']
    return this.getPrices(symbols)
  }

  // Get top 5 cryptocurrencies by market cap
  async getTop5(): Promise<PriceData[]> {
    const symbols = ['BTC', 'ETH', 'BNB', 'SOL', 'XRP']
    return this.getPrices(symbols)
  }

  // Get top 100 cryptocurrencies by market cap (optimized to top 40)
  async getTop100(): Promise<PriceData[]> {
    // Top 40 most traded cryptocurrencies for better performance
    const symbols = [
      'BTC', 'ETH', 'USDT', 'BNB', 'SOL', 'USDC', 'XRP', 'ADA', 'AVAX', 'DOGE',
      'TRX', 'LINK', 'DOT', 'MATIC', 'LTC', 'BCH', 'ATOM', 'NEAR', 'UNI', 'ETC',
      'XLM', 'ICP', 'FIL', 'VET', 'ALGO', 'MANA', 'SAND', 'AXS', 'CHZ', 'ENJ',
      'BAT', 'ZEC', 'FLOW', 'THETA', 'HBAR', 'EGLD', 'XTZ', 'AAVE', 'MKR', 'SUSHI'
    ]
    return this.getPrices(symbols)
  }

  // Search cryptocurrencies by name or symbol
  async searchCryptos(query: string): Promise<PriceData[]> {
    // Define a broader list of searchable cryptocurrencies
    const allSymbols = [
      'BTC', 'ETH', 'USDT', 'BNB', 'SOL', 'USDC', 'XRP', 'ADA', 'AVAX', 'DOGE',
      'TRX', 'LINK', 'DOT', 'MATIC', 'LTC', 'BCH', 'ATOM', 'NEAR', 'UNI', 'ETC',
      'XLM', 'ICP', 'FIL', 'VET', 'ALGO', 'MANA', 'SAND', 'AXS', 'CHZ', 'ENJ',
      'BAT', 'ZEC', 'FLOW', 'THETA', 'HBAR', 'EGLD', 'XTZ', 'AAVE', 'MKR', 'SUSHI',
      'CRV', '1INCH', 'BAL', 'LRC', 'KNC', 'STORJ', 'BAND', 'KAVA', 'ZRX', 'CAKE'
    ]

    const lowerQuery = query.toLowerCase()

    // Filter symbols that match the query
    const matchingSymbols = allSymbols.filter(symbol =>
      symbol.toLowerCase().includes(lowerQuery)
    )

    // If we have matching symbols, fetch their prices
    if (matchingSymbols.length > 0) {
      const prices = await this.getPrices(matchingSymbols)
      // Further filter by name after getting the full data
      return prices.filter(crypto =>
        crypto.name.toLowerCase().includes(lowerQuery) ||
        crypto.symbol.toLowerCase().includes(lowerQuery)
      ).slice(0, 10) // Return max 10 results
    }

    return []
  }

  // Get trending cryptocurrencies
  async getTrending(): Promise<TrendingCoin[]> {
    // Get trending coins based on highest 24h change (both positive and negative)
    const allPrices = await this.getAllPrices()
    return allPrices
      .sort((a, b) => Math.abs(b.change_24h) - Math.abs(a.change_24h))
      .slice(0, 15) // Show top 15 trending
      .map((coin, index) => ({
        id: coin.symbol.toLowerCase(),
        symbol: coin.symbol,
        name: coin.name,
        price: coin.price,
        change_24h: coin.change_24h,
        market_cap: coin.market_cap,
        volume: coin.volume,
        rank: index + 1
      }))
  }

  // Get market categories (simulated for now)
  async getCategories(): Promise<MarketCategory[]> {
    return [
      {
        id: 'defi',
        name: 'DeFi',
        market_cap: 45000000000,
        volume_24h: 3200000000,
        change_24h: 5.2
      },
      {
        id: 'gaming',
        name: 'Gaming',
        market_cap: 28000000000,
        volume_24h: 1800000000,
        change_24h: 3.8
      },
      {
        id: 'layer1',
        name: 'Layer 1',
        market_cap: 120000000000,
        volume_24h: 8500000000,
        change_24h: 2.1
      },
      {
        id: 'meme',
        name: 'Meme',
        market_cap: 15000000000,
        volume_24h: 1200000000,
        change_24h: -1.5
      },
      {
        id: 'ai',
        name: 'AI & Big Data',
        market_cap: 8500000000,
        volume_24h: 650000000,
        change_24h: 8.3
      }
    ]
  }

  // Get market statistics
  async getMarketStats(): Promise<MarketStats> {
    const prices = await this.getAllPrices()
    
    const totalMarketCap = prices.reduce((sum, coin) => sum + coin.market_cap, 0)
    const totalVolume24h = prices.reduce((sum, coin) => sum + coin.volume, 0)
    
    // Calculate BTC dominance
    const btcData = prices.find(coin => coin.symbol === 'BTC')
    const btcDominance = btcData ? (btcData.market_cap / totalMarketCap) * 100 : 0

    return {
      totalMarketCap,
      totalVolume24h,
      btcDominance
    }
  }

  // Get price history for a symbol
  async getPriceHistory(
    symbol: string, 
    interval?: string, 
    from?: number, 
    to?: number
  ): Promise<PriceHistory> {
    let endpoint = `/api/v1/history/${symbol.toUpperCase()}`
    const params = new URLSearchParams()
    
    if (interval) params.append('interval', interval)
    if (from) params.append('from', from.toString())
    if (to) params.append('to', to.toString())
    
    if (params.toString()) {
      endpoint += `?${params.toString()}`
    }

    return this.request<PriceHistory>(endpoint, {
      method: 'GET',
    })
  }

  // Health check
  async healthCheck(): Promise<{ status: string; timestamp: number; service: string }> {
    return this.request<{ status: string; timestamp: number; service: string }>('/health', {
      method: 'GET',
    })
  }
}

export const marketApiService = new MarketApiService()
export { MarketApiError }
