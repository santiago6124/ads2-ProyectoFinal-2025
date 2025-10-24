"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { TrendingUp, TrendingDown, Star, Loader2 } from "lucide-react"
import Link from "next/link"
import { marketApiService, PriceData } from "@/lib/market-api"

interface MarketTableProps {
  searchQuery: string
}

export function MarketTable({ searchQuery }: MarketTableProps) {
  const [favorites, setFavorites] = useState<Set<string>>(new Set())
  const [cryptoData, setCryptoData] = useState<PriceData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await marketApiService.getTop100()
        setCryptoData(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch market data')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchData, 30000)
    return () => clearInterval(interval)
  }, [])

  const filteredData = cryptoData.filter(
    (crypto) =>
      crypto.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      crypto.symbol.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  const toggleFavorite = (symbol: string) => {
    setFavorites((prev) => {
      const newFavorites = new Set(prev)
      if (newFavorites.has(symbol)) {
        newFavorites.delete(symbol)
      } else {
        newFavorites.add(symbol)
      }
      return newFavorites
    })
  }

  const formatCurrency = (amount: number) => {
    if (amount >= 1e12) {
      return `$${(amount / 1e12).toFixed(2)}T`
    } else if (amount >= 1e9) {
      return `$${(amount / 1e9).toFixed(2)}B`
    } else if (amount >= 1e6) {
      return `$${(amount / 1e6).toFixed(2)}M`
    } else if (amount >= 1e3) {
      return `$${(amount / 1e3).toFixed(2)}K`
    } else {
      return `$${amount.toFixed(2)}`
    }
  }

  const formatPrice = (price: number) => {
    if (price >= 1000) {
      return `$${price.toLocaleString()}`
    } else if (price >= 1) {
      return `$${price.toFixed(2)}`
    } else {
      return `$${price.toFixed(6)}`
    }
  }

  if (loading) {
    return (
      <Card className="p-8 bg-black border border-white/10 shadow-lg">
        <div className="flex items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-white" />
          <span className="ml-2 text-white">Loading market data...</span>
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="p-8 bg-black border border-white/10 shadow-lg">
        <div className="text-center text-red-400">
          <p className="text-lg font-semibold">Error loading market data</p>
          <p className="text-sm mt-2">{error}</p>
        </div>
      </Card>
    )
  }

  return (
    <Card className="overflow-hidden bg-black border border-white/10 shadow-lg">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="border-b border-white/10 bg-black">
            <tr>
              <th className="text-left p-4 text-sm font-semibold text-white/60">#</th>
              <th className="text-left p-4 text-sm font-semibold text-white/60">Name</th>
              <th className="text-right p-4 text-sm font-semibold text-white/60">Price</th>
              <th className="text-right p-4 text-sm font-semibold text-white/60">24h %</th>
              <th className="text-right p-4 text-sm font-semibold text-white/60 hidden md:table-cell">
                Volume (24h)
              </th>
              <th className="text-right p-4 text-sm font-semibold text-white/60 hidden lg:table-cell">
                Market Cap
              </th>
              <th className="text-center p-4 text-sm font-semibold text-white/60 hidden xl:table-cell">
                Last 7 Days
              </th>
              <th className="text-right p-4 text-sm font-semibold text-white/60">Action</th>
            </tr>
          </thead>
          <tbody>
            {filteredData.map((crypto, index) => (
              <tr key={crypto.symbol} className="border-b border-white/10 hover:bg-white/5 transition-colors">
                <td className="p-4">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => toggleFavorite(crypto.symbol)}
                      className="text-white/60 hover:text-yellow-400 transition-colors"
                    >
                      <Star
                        className={`h-4 w-4 ${favorites.has(crypto.symbol) ? "fill-yellow-400 text-yellow-400" : ""}`}
                      />
                    </button>
                    <span className="text-sm text-white/60">{index + 1}</span>
                  </div>
                </td>
                <td className="p-4">
                  <div className="flex items-center gap-3">
                    <div className="h-8 w-8 rounded-full bg-blue-500 flex items-center justify-center border border-white/10">
                      <span className="text-xs font-bold text-white">{crypto.symbol}</span>
                    </div>
                    <div>
                      <p className="font-semibold text-white">{crypto.name}</p>
                      <p className="text-sm text-white/60">{crypto.symbol}</p>
                    </div>
                  </div>
                </td>
                <td className="p-4 text-right font-semibold text-white">{formatPrice(crypto.price)}</td>
                <td className="p-4 text-right">
                  <div
                    className={`inline-flex items-center gap-1 font-semibold ${
                      crypto.change_24h >= 0 ? "text-green-400" : "text-red-400"
                    }`}
                  >
                    {crypto.change_24h >= 0 ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
                    {Math.abs(crypto.change_24h).toFixed(2)}%
                  </div>
                </td>
                <td className="p-4 text-right text-white/60 hidden md:table-cell">{formatCurrency(crypto.volume)}</td>
                <td className="p-4 text-right text-white/60 hidden lg:table-cell">{formatCurrency(crypto.market_cap)}</td>
                <td className="p-4 hidden xl:table-cell">
                  <div className="flex items-center justify-center">
                    <div className="w-20 h-8 bg-white/10 rounded flex items-center justify-center">
                      <span className="text-xs text-white/60">Chart</span>
                    </div>
                  </div>
                </td>
                <td className="p-4 text-right">
                  <Button size="sm" asChild className="bg-blue-500 hover:bg-blue-600 text-white border-0">
                    <Link href={`/trade?coin=${crypto.symbol.toLowerCase()}`}>Trade</Link>
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}
