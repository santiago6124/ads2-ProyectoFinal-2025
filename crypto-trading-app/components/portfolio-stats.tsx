"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Wallet, DollarSign } from "lucide-react"
import { useAuth } from "@/lib/auth-context"
import { getPortfolio } from "@/lib/portfolio-api"

export function PortfolioStats() {
  const { user } = useAuth()
  const [availableCash, setAvailableCash] = useState(0)
  const [totalBalance, setTotalBalance] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchPortfolioData = async () => {
      if (!user?.id) return

      try {
        // Get complete portfolio data from portfolio-api (includes current prices, P&L, etc.)
        const portfolio = await getPortfolio(user.id)

        // Use pre-calculated values from backend
        const totalValue = parseFloat(portfolio.total_value) || 0
        const cash = parseFloat(portfolio.total_cash) || 0

        setTotalBalance(totalValue)
        setAvailableCash(cash)
      } catch (error) {
        console.error('Error fetching portfolio data:', error)
        // On error, try to fallback to user balance
        const fallbackCash = user.initial_balance || 0
        setAvailableCash(fallbackCash)
        setTotalBalance(fallbackCash)
      } finally {
        setLoading(false)
      }
    }

    fetchPortfolioData()
  }, [user])

  if (loading) {
    return (
      <div className="grid sm:grid-cols-2 lg:grid-cols-2 gap-4">
        {[1, 2].map((i) => (
          <Card key={i} className="p-6 animate-pulse">
            <div className="h-20 bg-muted rounded" />
          </Card>
        ))}
      </div>
    )
  }

  const stats = [
    {
      name: "Total Balance",
      value: `$${totalBalance.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
      icon: Wallet,
      description: "Total portfolio value",
    },
    {
      name: "Available Cash",
      value: `$${availableCash.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
      icon: DollarSign,
      description: "USD balance ready to trade",
    },
  ]

  return (
    <div className="grid sm:grid-cols-2 gap-4">
      {stats.map((stat) => (
        <Card key={stat.name} className="p-6">
          <div className="flex items-start justify-between mb-4">
            <div className="h-12 w-12 rounded-xl bg-primary/10 flex items-center justify-center">
              <stat.icon className="h-6 w-6 text-primary" />
            </div>
          </div>
          <div>
            <p className="text-sm text-muted-foreground mb-1">{stat.name}</p>
            <p className="text-2xl font-bold mb-1">{stat.value}</p>
            <p className="text-xs text-muted-foreground">{stat.description}</p>
          </div>
        </Card>
      ))}
    </div>
  )
}
