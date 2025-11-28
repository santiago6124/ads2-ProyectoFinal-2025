import { config } from './config'
import { ordersCache } from './orders-cache'

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
      console.log('🔍 Search API request params:', params)

      // Try cache first for user-specific searches
      if (params.user_id && params.page) {
        const cached = ordersCache.getUserOrders(params.user_id, params.page)
        if (cached) {
          console.log('✅ Cache hit for user orders')
          // Return cached data and revalidate in background
          this.revalidateUserOrders(params).catch(err =>
            console.warn('Background revalidation failed:', err)
          )
          return cached
        }
      }

      const response = await fetch(`${this.baseUrl}/api/v1/search`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params),
      })

      if (!response.ok) {
        // Handle rate limiting gracefully
        if (response.status === 429) {
          console.warn('⚠️ Search API rate limit reached, trying cache')

          // Try to return stale cache
          if (params.user_id && params.page) {
            const staleCache = ordersCache.getUserOrders(params.user_id, params.page)
            if (staleCache) {
              console.log('✅ Returning stale cache for rate limit')
              return staleCache
            }
          }
        }
        throw new Error(`Search failed: ${response.statusText}`)
      }

      const data = await response.json()

      // Cache successful results
      if (params.user_id && params.page && data.results?.length > 0) {
        ordersCache.setUserOrders(params.user_id, params.page, data)
      }

      return data
    } catch (error) {
      console.error('Search API error:', error)

      // Try to return cached data on any error
      if (params.user_id && params.page) {
        const cached = ordersCache.getUserOrders(params.user_id, params.page)
        if (cached) {
          console.log('✅ Returning cache after error')
          return cached
        }
      }

      throw error
    }
  }

  private async revalidateUserOrders(params: SearchOrderRequest): Promise<void> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/search`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params),
      })

      if (response.ok) {
        const data = await response.json()
        if (params.user_id && params.page && data.results?.length > 0) {
          ordersCache.setUserOrders(params.user_id, params.page, data)
        }
      }
    } catch (error) {
      // Silently fail background revalidation
      console.debug('Background revalidation failed:', error)
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
    // Try cache first
    const cached = ordersCache.getRecentOrders(userId)
    if (cached && cached.length > 0) {
      console.log('✅ Cache hit for recent orders')
      // Return cached orders wrapped in SearchResponse format
      const response: SearchResponse = {
        results: cached.slice(0, limit),
        total: cached.length,
        page: 1,
        limit,
        total_pages: 1,
      }

      // Revalidate in background
      this.revalidateRecentOrders(userId, limit).catch(err =>
        console.warn('Background revalidation of recent orders failed:', err)
      )

      return response
    }

    // Cache miss, fetch from API
    try {
      const response = await this.searchOrders({
        user_id: userId,
        page: 1,
        limit,
        sort: 'created_at_desc',
        status: ['executed'],
      })

      // Cache the results
      if (response.results && response.results.length > 0) {
        ordersCache.setRecentOrders(userId, response.results)
      }

      return response
    } catch (error) {
      // Try to return any cached data on error
      if (cached && cached.length > 0) {
        console.log('✅ Returning stale cache for recent orders after error')
        return {
          results: cached.slice(0, limit),
          total: cached.length,
          page: 1,
          limit,
          total_pages: 1,
        }
      }
      throw error
    }
  }

  private async revalidateRecentOrders(userId: number, limit: number): Promise<void> {
    try {
      const response = await this.searchOrders({
        user_id: userId,
        page: 1,
        limit,
        sort: 'created_at_desc',
        status: ['executed'],
      })

      if (response.results && response.results.length > 0) {
        ordersCache.setRecentOrders(userId, response.results)
      }
    } catch (error) {
      console.debug('Background revalidation of recent orders failed:', error)
    }
  }
}

export const searchApiService = new SearchApiService()
