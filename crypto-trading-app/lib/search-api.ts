import { config } from './config'

export interface SearchOrderRequest {
  q?: string
  page?: number
  limit?: number
  sort?: string
  status?: string[]
  type?: string[]
  order_kind?: string[]
  crypto_symbol?: string[]
  user_id?: number
  min_total_amount?: number
  max_total_amount?: number
  date_from?: string
  date_to?: string
}

export interface OrderSearchResult {
  id: string
  user_id: number
  type: string
  status: string
  order_kind: string
  crypto_symbol: string
  crypto_name: string
  quantity: string
  price: string
  total_amount: string
  fee: string
  created_at: string
  updated_at: string
  executed_at?: string
  cancelled_at?: string
}

export interface SearchResponse {
  results: OrderSearchResult[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export interface FilterOption {
  value: string
  label: string
  count: number
}

export interface FiltersResponse {
  statuses: FilterOption[]
  types: FilterOption[]
  order_kinds: FilterOption[]
  crypto_symbols: FilterOption[]
  sort_options: Array<{ value: string; label: string }>
}

class SearchApiService {
  private baseUrl: string

  constructor() {
    this.baseUrl = config.searchApiUrl
  }

  async searchOrders(params: SearchOrderRequest): Promise<SearchResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/search`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params),
      })

      if (!response.ok) {
        throw new Error(`Search failed: ${response.statusText}`)
      }

      return await response.json()
    } catch (error) {
      console.error('Search API error:', error)
      throw error
    }
  }

  async getOrderById(orderId: string): Promise<OrderSearchResult> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/orders/${orderId}`)

      if (!response.ok) {
        throw new Error(`Get order failed: ${response.statusText}`)
      }

      return await response.json()
    } catch (error) {
      console.error('Search API error:', error)
      throw error
    }
  }

  async getFilters(): Promise<FiltersResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/filters`)

      if (!response.ok) {
        throw new Error(`Get filters failed: ${response.statusText}`)
      }

      return await response.json()
    } catch (error) {
      console.error('Search API error:', error)
      throw error
    }
  }

  async getUserOrders(userId: number, page: number = 1, limit: number = 20): Promise<SearchResponse> {
    return this.searchOrders({
      user_id: userId,
      page,
      limit,
      sort: 'created_at_desc',
    })
  }

  async getRecentOrders(userId: number, limit: number = 10): Promise<SearchResponse> {
    return this.searchOrders({
      user_id: userId,
      page: 1,
      limit,
      sort: 'created_at_desc',
    })
  }
}

export const searchApiService = new SearchApiService()
