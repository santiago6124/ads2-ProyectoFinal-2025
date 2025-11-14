"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { Card } from "@/components/ui/card"
import { TrendingUp, TrendingDown, Loader2, Flame } from "lucide-react"
import { marketApiService, TrendingCoin } from "@/lib/market-api"

export function TrendingCoins() {
  const router = useRouter()
  const [trendingCoins, setTrendingCoins] = useState<TrendingCoin[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchTrending = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await marketApiService.getTrending()
        setTrendingCoins(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch trending coins')
      } finally {
        setLoading(false)
      }
    }

    fetchTrending()

    // Refresh every 60 seconds (reduced from 30 to minimize constant updates)
    const interval = setInterval(fetchTrending, 60000)
    return () => clearInterval(interval)
  }, [])

  const formatPrice = (price: number) => {
    if (price >= 1000) {
      return `$${price.toLocaleString()}`
    } else if (price >= 1) {
      return `$${price.toFixed(2)}`
    } else {
      return `$${price.toFixed(6)}`
    }
  }

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
          <span className="ml-2 text-white">Loading trending...</span>
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="text-center text-red-400">
          <p className="text-lg font-semibold">Error loading trending</p>
          <p className="text-sm mt-2">{error}</p>
        </div>
      </Card>
    )
  }

  return (
    <Card className="p-6 bg-black border border-white/10 shadow-lg">
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-2">
          <Flame className="h-6 w-6 text-orange-500" />
          <h2 className="text-2xl font-bold text-white">Trending</h2>
        </div>
        <p className="text-sm text-white/60">Top movers by 24h change</p>
      </div>

      <div className="space-y-4">
        {trendingCoins.map((coin) => (
          <div
            key={coin.id}
            className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300 cursor-pointer"
            onClick={() => router.push(`/trade?crypto=${coin.symbol}`)}
          >
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <span className="text-sm font-bold text-white/60">#{coin.rank}</span>
                <div className="h-12 w-12 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg border border-white/10">
                  <span className="text-sm font-bold text-white">{coin.symbol}</span>
                </div>
              </div>
              <div>
                <p className="font-bold text-white">{coin.name}</p>
                <p className="text-sm text-white/60">{coin.symbol}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-bold text-white">{formatPrice(coin.price)}</p>
              <div
                className={`flex items-center justify-end gap-1 text-sm font-bold px-2 py-1 rounded-full border ${
                  coin.change_24h >= 0 
                    ? "bg-green-500/20 text-green-400 border-green-500/30" 
                    : "bg-red-500/20 text-red-400 border-red-500/30"
                }`}
              >
                {coin.change_24h >= 0 ? (
                  <TrendingUp className="h-3 w-3" />
                ) : (
                  <TrendingDown className="h-3 w-3" />
                )}
                {Math.abs(coin.change_24h).toFixed(2)}%
              </div>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
