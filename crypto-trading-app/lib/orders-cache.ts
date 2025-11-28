/**
 * Frontend cache for orders to provide resilience when Search API is unavailable
 * Implements multi-level caching with TTL and stale-while-revalidate pattern
 */

import { OrderSearchResult, SearchResponse } from './search-api'

interface CacheEntry<T> {
  data: T
  timestamp: number
  expiresAt: number
}

interface CacheConfig {
  ttl: number // Time to live in milliseconds
  staleTtl: number // Extended TTL for stale-while-revalidate
}

const DEFAULT_CONFIG: CacheConfig = {
  ttl: 5 * 60 * 1000, // 5 minutes fresh
  staleTtl: 30 * 60 * 1000, // 30 minutes stale
}

export class OrdersCache {
  private config: CacheConfig
  private storagePrefix = 'orders_cache_'

  constructor(config: Partial<CacheConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config }
  }

  /**
   * Get recent orders for a user
   */
  getRecentOrders(userId: number): OrderSearchResult[] | null {
    const key = `${this.storagePrefix}recent_${userId}`
    return this.get<OrderSearchResult[]>(key)
  }

  /**
   * Cache recent orders for a user
   */
  setRecentOrders(userId: number, orders: OrderSearchResult[]): void {
    const key = `${this.storagePrefix}recent_${userId}`
    this.set(key, orders)
  }

  /**
   * Get user orders for a specific page
   */
  getUserOrders(userId: number, page: number): SearchResponse | null {
    const key = `${this.storagePrefix}user_${userId}_page_${page}`
    return this.get<SearchResponse>(key)
  }

  /**
   * Cache user orders for a specific page
   */
  setUserOrders(userId: number, page: number, response: SearchResponse): void {
    const key = `${this.storagePrefix}user_${userId}_page_${page}`
    this.set(key, response)
  }

  /**
   * Get search results for a query
   */
  getSearchResults(query: string, page: number): SearchResponse | null {
    const key = `${this.storagePrefix}search_${this.hashQuery(query)}_page_${page}`
    return this.get<SearchResponse>(key)
  }

  /**
   * Cache search results
   */
  setSearchResults(query: string, page: number, response: SearchResponse): void {
    const key = `${this.storagePrefix}search_${this.hashQuery(query)}_page_${page}`
    this.set(key, response)
  }

  /**
   * Invalidate all cache entries for a user
   */
  invalidateUser(userId: number): void {
    this.invalidatePattern(`user_${userId}`)
    this.invalidatePattern(`recent_${userId}`)
  }

  /**
   * Invalidate all search cache
   */
  invalidateSearch(): void {
    this.invalidatePattern('search_')
  }

  /**
   * Clear all cache entries
   */
  clearAll(): void {
    if (typeof window === 'undefined') return

    const keys = Object.keys(localStorage)
    keys.forEach(key => {
      if (key.startsWith(this.storagePrefix)) {
        localStorage.removeItem(key)
      }
    })
  }

  /**
   * Get cache statistics
   */
  getStats(): { entries: number; totalSize: number } {
    if (typeof window === 'undefined') return { entries: 0, totalSize: 0 }

    const keys = Object.keys(localStorage).filter(key => key.startsWith(this.storagePrefix))
    let totalSize = 0

    keys.forEach(key => {
      const item = localStorage.getItem(key)
      if (item) {
        totalSize += item.length
      }
    })

    return {
      entries: keys.length,
      totalSize: totalSize,
    }
  }

  /**
   * Generic get with TTL check
   */
  private get<T>(key: string): T | null {
    if (typeof window === 'undefined') return null

    try {
      const item = localStorage.getItem(key)
      if (!item) return null

      const entry: CacheEntry<T> = JSON.parse(item)
      const now = Date.now()

      // Check if entry is fresh
      if (now < entry.expiresAt) {
        return entry.data
      }

      // Check if entry is stale but still usable
      if (now < entry.timestamp + this.config.staleTtl) {
        console.warn(`[OrdersCache] Returning stale cache for ${key}`)
        return entry.data
      }

      // Entry is too old, remove it
      localStorage.removeItem(key)
      return null
    } catch (error) {
      console.error('[OrdersCache] Failed to get cache:', error)
      return null
    }
  }

  /**
   * Generic set with TTL
   */
  private set<T>(key: string, data: T): void {
    if (typeof window === 'undefined') return

    try {
      const now = Date.now()
      const entry: CacheEntry<T> = {
        data,
        timestamp: now,
        expiresAt: now + this.config.ttl,
      }

      localStorage.setItem(key, JSON.stringify(entry))
    } catch (error) {
      console.error('[OrdersCache] Failed to set cache:', error)
      // Handle quota exceeded error
      if (error instanceof DOMException && error.name === 'QuotaExceededError') {
        console.warn('[OrdersCache] LocalStorage quota exceeded, clearing old entries')
        this.clearOldest(5) // Clear 5 oldest entries
      }
    }
  }

  /**
   * Invalidate entries matching a pattern
   */
  private invalidatePattern(pattern: string): void {
    if (typeof window === 'undefined') return

    const keys = Object.keys(localStorage)
    keys.forEach(key => {
      if (key.includes(pattern)) {
        localStorage.removeItem(key)
      }
    })
  }

  /**
   * Clear oldest cache entries
   */
  private clearOldest(count: number): void {
    if (typeof window === 'undefined') return

    const entries: Array<{ key: string; timestamp: number }> = []

    Object.keys(localStorage).forEach(key => {
      if (key.startsWith(this.storagePrefix)) {
        try {
          const item = localStorage.getItem(key)
          if (item) {
            const entry: CacheEntry<any> = JSON.parse(item)
            entries.push({ key, timestamp: entry.timestamp })
          }
        } catch (error) {
          // Invalid entry, remove it
          localStorage.removeItem(key)
        }
      }
    })

    // Sort by timestamp (oldest first) and remove
    entries
      .sort((a, b) => a.timestamp - b.timestamp)
      .slice(0, count)
      .forEach(({ key }) => localStorage.removeItem(key))
  }

  /**
   * Simple hash function for query strings
   */
  private hashQuery(query: string): string {
    let hash = 0
    for (let i = 0; i < query.length; i++) {
      const char = query.charCodeAt(i)
      hash = (hash << 5) - hash + char
      hash = hash & hash // Convert to 32bit integer
    }
    return Math.abs(hash).toString(36)
  }
}

// Singleton instance
export const ordersCache = new OrdersCache()

/**
 * Wrapper function to add cache-then-network pattern to search API calls
 */
export async function cacheFirstSearch<T>(
  cacheKey: string,
  fetchFn: () => Promise<T>,
  cache: OrdersCache = ordersCache
): Promise<{ data: T; fromCache: boolean }> {
  // Try cache first
  const cached = cache.get<T>(cacheKey)
  if (cached !== null) {
    // Return cached data immediately and revalidate in background
    fetchFn()
      .then(freshData => cache.set(cacheKey, freshData))
      .catch(error => console.error('[OrdersCache] Background revalidation failed:', error))

    return { data: cached, fromCache: true }
  }

  // Cache miss, fetch fresh data
  try {
    const data = await fetchFn()
    cache.set(cacheKey, data)
    return { data, fromCache: false }
  } catch (error) {
    throw error
  }
}
