"use client"

import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { ArrowUpRight, ArrowDownRight, Loader2 } from "lucide-react"
import { useEffect, useState } from "react"
import { useAuth } from "@/lib/auth-context"
import { getPortfolio, formatCrypto, formatUSD, formatPercentage, getTrend, type Portfolio } from "@/lib/portfolio-api"

export function PortfolioOverview() {
  const { user } = useAuth()
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchPortfolio = async () => {
    if (!user?.id) return

    try {
      setLoading(true)
      setError(null)
      const data = await getPortfolio(user.id)
      setPortfolio(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load portfolio')
      console.error('Error loading portfolio:', err)
    } finally {
      setLoading(false)
    }
  }

  // Initial fetch
  useEffect(() => {
    fetchPortfolio()
  }, [user])

  // Listen for portfolio refresh events
  useEffect(() => {
    const handlePortfolioRefresh = () => {
      console.log('PortfolioOverview: portfolio-refresh event received, refetching data...')
      fetchPortfolio()
    }

    window.addEventListener('portfolio-refresh', handlePortfolioRefresh)

    return () => {
      window.removeEventListener('portfolio-refresh', handlePortfolioRefresh)
    }
  }, [user])

  if (loading) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 text-white animate-spin" />
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="flex flex-col items-center justify-center py-12">
          <p className="text-red-400 mb-4">Error loading portfolio</p>
          <p className="text-sm text-white/60">{error}</p>
        </div>
      </Card>
    )
  }

  if (!portfolio || !portfolio.holdings || portfolio.holdings.length === 0) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-white">Your Holdings</h2>
            <p className="text-sm text-white/60 mt-1">Current cryptocurrency positions</p>
          </div>
        </div>
        <div className="flex flex-col items-center justify-center py-12">
          <p className="text-white/60 mb-2">No holdings yet</p>
          <p className="text-sm text-white/40">Start trading to see your portfolio</p>
        </div>
      </Card>
    )
  }

  return (
    <Card className="p-6 bg-black border border-white/10 shadow-lg">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-white">Your Holdings</h2>
          <p className="text-sm text-white/60 mt-1">
            Total Value: {formatUSD(portfolio.total_value)} •
            P/L: <span className={getTrend(portfolio.profit_loss) === "up" ? "text-green-400" : "text-red-400"}>
              {formatPercentage(portfolio.profit_loss_percentage)}
            </span>
          </p>
        </div>
        <Button variant="outline" size="sm" className="border-white/20 text-white hover:bg-white/10">
          View All
        </Button>
      </div>

      <div className="space-y-4">
        {portfolio.holdings.map((holding) => {
          const trend = getTrend(holding.profit_loss)
          const change = formatPercentage(holding.profit_loss_percentage)

          return (
            <div
              key={holding.symbol}
              className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300"
            >
              <div className="flex items-center gap-4">
                <div className="h-14 w-14 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg border border-white/10">
                  <span className="text-sm font-bold text-white">{holding.symbol}</span>
                </div>
                <div>
                  <p className="font-bold text-white">{holding.name || holding.symbol}</p>
                  <p className="text-sm text-white/60">
                    {formatCrypto(holding.quantity)} {holding.symbol}
                  </p>
                </div>
              </div>
              <div className="text-right">
                <p className="font-bold text-white">{formatUSD(holding.current_value)}</p>
                <div
                  className={`flex items-center justify-end gap-1 text-sm font-bold px-2 py-1 rounded-full border ${
                    trend === "up"
                      ? "bg-green-500/20 text-green-400 border-green-500/30"
                      : "bg-red-500/20 text-red-400 border-red-500/30"
                  }`}
                >
                  {trend === "up" ? <ArrowUpRight className="h-4 w-4" /> : <ArrowDownRight className="h-4 w-4" />}
                  {change}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </Card>
  )
}
