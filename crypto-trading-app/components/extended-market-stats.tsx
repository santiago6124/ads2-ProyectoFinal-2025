"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { TrendingUp, TrendingDown, Activity, BarChart3, DollarSign, Loader2 } from "lucide-react"
import { marketApiService, MarketStats } from "@/lib/market-api"

export function ExtendedMarketStats() {
  const [marketStats, setMarketStats] = useState<MarketStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchStats = async () => {
      try {
        setLoading(true)
        setError(null)
        const stats = await marketApiService.getMarketStats()
        setMarketStats(stats)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch market stats')
      } finally {
        setLoading(false)
      }
    }

    fetchStats()
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchStats, 30000)
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

  const stats = [
    {
      name: "Total Market Cap",
      value: marketStats ? formatCurrency(marketStats.totalMarketCap) : "$0.00",
      change: "+2.4%",
      trend: "up" as const,
      icon: DollarSign,
      description: "Total value of all cryptocurrencies"
    },
    {
      name: "24h Volume",
      value: marketStats ? formatCurrency(marketStats.totalVolume24h) : "$0.00",
      change: "+5.2%",
      trend: "up" as const,
      icon: BarChart3,
      description: "Total trading volume in 24 hours"
    },
    {
      name: "BTC Dominance",
      value: marketStats ? `${marketStats.btcDominance.toFixed(1)}%` : "0.0%",
      change: "-0.8%",
      trend: "down" as const,
      icon: Activity,
      description: "Bitcoin's share of total market cap"
    },
    {
      name: "Fear & Greed Index",
      value: "68",
      change: "+3",
      trend: "up" as const,
      icon: TrendingUp,
      description: "Market sentiment indicator"
    }
  ]

  if (loading) {
    return (
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
        {[...Array(4)].map((_, i) => (
          <Card key={i} className="p-6 bg-black border border-white/10 shadow-lg">
            <div className="flex items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-white" />
            </div>
          </Card>
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="text-center text-red-400">
          <p className="text-lg font-semibold">Error loading market stats</p>
          <p className="text-sm mt-2">{error}</p>
        </div>
      </Card>
    )
  }

  return (
    <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
      {stats.map((stat, index) => (
        <Card key={index} className="p-6 bg-black border border-white/10 shadow-lg hover:shadow-xl transition-all duration-300 hover:scale-105">
          <div className="flex items-center justify-between mb-4">
            <div className="h-12 w-12 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg border border-white/10">
              <stat.icon className="h-6 w-6 text-white" />
            </div>
            <div
              className={`flex items-center gap-1 text-sm font-bold px-2 py-1 rounded-full border ${
                stat.trend === "up" 
                  ? "bg-green-500/20 text-green-400 border-green-500/30" 
                  : "bg-red-500/20 text-red-400 border-red-500/30"
              }`}
            >
              {stat.trend === "up" ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
              {stat.change}
            </div>
          </div>
          <div>
            <p className="text-sm text-white/60 mb-2 font-medium">{stat.name}</p>
            <p className="text-2xl font-bold text-white mb-1">{stat.value}</p>
            <p className="text-xs text-white/50">{stat.description}</p>
          </div>
        </Card>
      ))}
    </div>
  )
}
