"use client"

import { useEffect, useState } from "react"
import { TrendingUp, TrendingDown, DollarSign, Activity } from "lucide-react"
import { Card } from "@/components/ui/card"
import { useAuth } from "@/lib/auth-context"
import { getPortfolio } from "@/lib/portfolio-api"

export function QuickStats() {
  const { user } = useAuth()
  const [portfolioValue, setPortfolioValue] = useState(0)
  const [availableBalance, setAvailableBalance] = useState(0)
  const [dailyChange, setDailyChange] = useState(0)
  const [dailyChangePercent, setDailyChangePercent] = useState(0)
  const [totalProfit, setTotalProfit] = useState(0)
  const [totalProfitPercent, setTotalProfitPercent] = useState(0)
  const [loading, setLoading] = useState(true)

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(amount)
  }

  const formatPercent = (percent: number) => {
    const sign = percent >= 0 ? '+' : ''
    return `${sign}${percent.toFixed(2)}%`
  }

  // Fetch real portfolio data
  const fetchPortfolioData = async () => {
    if (!user?.id) return

    try {
      const portfolio = await getPortfolio(user.id)
      const totalValue = parseFloat(portfolio.total_value) || 0
      const cash = user.balance || 0  // Use user balance instead of portfolio.total_cash

      // Use real performance metrics from backend
      const daily24h = parseFloat(portfolio.performance?.daily_change || '0')
      const daily24hPercent = parseFloat(portfolio.performance?.daily_change_percentage || '0')
      const profit = parseFloat(portfolio.profit_loss || '0')
      const profitPercent = parseFloat(portfolio.profit_loss_percentage || '0')

      setPortfolioValue(totalValue)
      setAvailableBalance(cash)
      setDailyChange(daily24h)
      setDailyChangePercent(daily24hPercent)
      setTotalProfit(profit)
      setTotalProfitPercent(profitPercent)
    } catch (error) {
      console.error('Error fetching portfolio data:', error)
      // Fallback to user balance
      const fallback = user.balance || 0
      setPortfolioValue(fallback)
      setAvailableBalance(fallback)
      setDailyChange(0)
      setDailyChangePercent(0)
      setTotalProfit(0)
      setTotalProfitPercent(0)
    } finally {
      setLoading(false)
    }
  }

  // Initial fetch
  useEffect(() => {
    fetchPortfolioData()
  }, [user])

  // Listen for portfolio refresh events
  useEffect(() => {
    const handlePortfolioRefresh = () => {
      console.log('QuickStats: portfolio-refresh event received, refetching data...')
      fetchPortfolioData()
    }

    window.addEventListener('portfolio-refresh', handlePortfolioRefresh)

    return () => {
      window.removeEventListener('portfolio-refresh', handlePortfolioRefresh)
    }
  }, [user])

  const stats = [
    {
      name: "Portfolio Value",
      value: formatCurrency(portfolioValue),
      change: formatPercent(totalProfitPercent),
      trend: totalProfitPercent >= 0 ? "up" : "down",
      icon: DollarSign,
    },
    {
      name: "24h Change",
      value: formatCurrency(dailyChange),
      change: formatPercent(dailyChangePercent),
      trend: dailyChangePercent >= 0 ? "up" : "down",
      icon: dailyChangePercent >= 0 ? TrendingUp : TrendingDown,
    },
    {
      name: "Total Profit",
      value: formatCurrency(totalProfit),
      change: formatPercent(totalProfitPercent),
      trend: totalProfit >= 0 ? "up" : "down",
      icon: Activity,
    },
    {
      name: "Available Balance",
      value: formatCurrency(availableBalance),
      change: "Cash",
      trend: "neutral",
      icon: DollarSign,
    },
  ]

  return (
    <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
      {stats.map((stat, index) => (
        <Card key={stat.name} className="p-6 bg-black border border-white/10 shadow-lg hover:shadow-xl transition-all duration-300 hover:scale-105">
          <div className="flex items-center justify-between mb-4">
            <div className="h-12 w-12 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg border border-white/10">
              <stat.icon className="h-6 w-6 text-white" />
            </div>
            {stat.trend !== "neutral" && (
              <div
                className={`flex items-center gap-1 text-sm font-bold px-2 py-1 rounded-full border ${
                  stat.trend === "up" ? "bg-green-500/20 text-green-400 border-green-500/30" : "bg-red-500/20 text-red-400 border-red-500/30"
                }`}
              >
                {stat.trend === "up" ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
                {stat.change}
              </div>
            )}
          </div>
          <div>
            <p className="text-sm text-white/60 mb-2 font-medium">{stat.name}</p>
            <p className="text-2xl font-bold text-white">{stat.value}</p>
          </div>
        </Card>
      ))}
    </div>
  )
}
