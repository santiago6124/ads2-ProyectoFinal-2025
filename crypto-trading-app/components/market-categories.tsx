"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { TrendingUp, TrendingDown, Loader2 } from "lucide-react"
import { marketApiService, MarketCategory } from "@/lib/market-api"

export function MarketCategories() {
  const [categories, setCategories] = useState<MarketCategory[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchCategories = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await marketApiService.getCategories()
        setCategories(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch categories')
      } finally {
        setLoading(false)
      }
    }

    fetchCategories()
    
    // Refresh every 60 seconds
    const interval = setInterval(fetchCategories, 60000)
    return () => clearInterval(interval)
  }, [])

  const formatCurrency = (amount: number) => {
    if (amount >= 1e12) {
      return `$${(amount / 1e12).toFixed(2)}T`
    } else if (amount >= 1e9) {
      return `$${(amount / 1e9).toFixed(2)}B`
    } else if (amount >= 1e6) {
      return `$${(amount / 1e6).toFixed(2)}M`
    } else {
      return `$${amount.toFixed(2)}`
    }
  }

  if (loading) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="flex items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-white" />
          <span className="ml-2 text-white">Loading categories...</span>
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="text-center text-red-400">
          <p className="text-lg font-semibold">Error loading categories</p>
          <p className="text-sm mt-2">{error}</p>
        </div>
      </Card>
    )
  }

  return (
    <Card className="p-6 bg-black border border-white/10 shadow-lg">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-white">Market Categories</h2>
        <p className="text-sm text-white/60 mt-1">Sector performance overview</p>
      </div>

      <div className="space-y-4">
        {categories.map((category) => (
          <div
            key={category.id}
            className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300 cursor-pointer"
          >
            <div className="flex items-center gap-4">
              <div className="h-12 w-12 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg border border-white/10">
                <span className="text-sm font-bold text-white">
                  {category.name.charAt(0)}
                </span>
              </div>
              <div>
                <p className="font-bold text-white">{category.name}</p>
                <p className="text-sm text-white/60">
                  Market Cap: {formatCurrency(category.market_cap)}
                </p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-bold text-white">{formatCurrency(category.volume_24h)}</p>
              <div
                className={`flex items-center justify-end gap-1 text-sm font-bold px-2 py-1 rounded-full border ${
                  category.change_24h >= 0 
                    ? "bg-green-500/20 text-green-400 border-green-500/30" 
                    : "bg-red-500/20 text-red-400 border-red-500/30"
                }`}
              >
                {category.change_24h >= 0 ? (
                  <TrendingUp className="h-3 w-3" />
                ) : (
                  <TrendingDown className="h-3 w-3" />
                )}
                {Math.abs(category.change_24h).toFixed(2)}%
              </div>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
