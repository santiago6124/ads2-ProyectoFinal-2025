// Market Data API service for communicating with market-data-api microservice
import { config } from './config'

const MARKET_API_BASE_URL = config.marketApiUrl || "http://localhost:8004"

export interface PriceData {
  symbol: string
  price: number
  timestamp: number
  source?: string
}

export interface PriceHistory {
  symbol: string
  history: Array<{
    timestamp: number
    price: number
  }>
}

export interface MarketStats {
  symbol: string
  high_24h: number
  low_24h: number
  volume_24h: number
  market_cap: number
  price_change_24h: number
  price_change_percentage_24h: number
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
    return this.request<PriceData>(`/api/v1/prices/${symbol.toUpperCase()}`, {
      method: 'GET',
    })
  }

  // Get prices for multiple symbols
  async getPrices(symbols?: string[]): Promise<{ message: string; data: string[] }> {
    const endpoint = symbols 
      ? `/api/v1/prices?symbols=${symbols.join(',')}`
      : '/api/v1/prices'
    
    return this.request<{ message: string; data: string[] }>(endpoint, {
      method: 'GET',
    })
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
