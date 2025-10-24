"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { TrendingUp, TrendingDown, Loader2, Trophy, AlertTriangle } from "lucide-react"
import { marketApiService, PriceData } from "@/lib/market-api"

export function WinnersLosers() {
  const [winners, setWinners] = useState<PriceData[]>([])
  const [losers, setLosers] = useState<PriceData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await marketApiService.getAllPrices()
        
        // Sort by 24h change and get top 5 winners and losers
        const sortedByChange = data.sort((a, b) => b.change_24h - a.change_24h)
        setWinners(sortedByChange.slice(0, 5))
        setLosers(sortedByChange.slice(-5).reverse())
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch winners/losers')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchData, 30000)
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
      <div className="grid lg:grid-cols-2 gap-6">
        <Card className="p-6 bg-black border border-white/10 shadow-lg">
          <div className="flex items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-white" />
            <span className="ml-2 text-white">Loading winners...</span>
          </div>
        </Card>
        <Card className="p-6 bg-black border border-white/10 shadow-lg">
          <div className="flex items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-white" />
            <span className="ml-2 text-white">Loading losers...</span>
          </div>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="text-center text-red-400">
          <p className="text-lg font-semibold">Error loading data</p>
          <p className="text-sm mt-2">{error}</p>
        </div>
      </Card>
    )
  }

  return (
    <div className="grid lg:grid-cols-2 gap-6">
      {/* Winners */}
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-2">
            <Trophy className="h-6 w-6 text-yellow-500" />
            <h2 className="text-2xl font-bold text-white">Top Gainers</h2>
          </div>
          <p className="text-sm text-white/60">Best performing cryptocurrencies today</p>
        </div>

        <div className="space-y-4">
          {winners.map((coin, index) => (
            <div
              key={coin.symbol}
              className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300"
            >
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-bold text-white/60">#{index + 1}</span>
                  <div className="h-12 w-12 rounded-xl bg-green-500 flex items-center justify-center shadow-lg border border-white/10">
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
                <div className="flex items-center justify-end gap-1 text-sm font-bold px-2 py-1 rounded-full bg-green-500/20 text-green-400 border border-green-500/30">
                  <TrendingUp className="h-3 w-3" />
                  +{coin.change_24h.toFixed(2)}%
                </div>
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* Losers */}
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-2">
            <AlertTriangle className="h-6 w-6 text-red-500" />
            <h2 className="text-2xl font-bold text-white">Top Losers</h2>
          </div>
          <p className="text-sm text-white/60">Worst performing cryptocurrencies today</p>
        </div>

        <div className="space-y-4">
          {losers.map((coin, index) => (
            <div
              key={coin.symbol}
              className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300"
            >
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-bold text-white/60">#{index + 1}</span>
                  <div className="h-12 w-12 rounded-xl bg-red-500 flex items-center justify-center shadow-lg border border-white/10">
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
                <div className="flex items-center justify-end gap-1 text-sm font-bold px-2 py-1 rounded-full bg-red-500/20 text-red-400 border border-red-500/30">
                  <TrendingDown className="h-3 w-3" />
                  {coin.change_24h.toFixed(2)}%
                </div>
              </div>
            </div>
          ))}
        </div>
      </Card>
    </div>
  )
}
