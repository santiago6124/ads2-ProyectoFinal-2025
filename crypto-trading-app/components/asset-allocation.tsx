"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts"
import { useAuth } from "@/lib/auth-context"
import { getPortfolio } from "@/lib/portfolio-api"

const COLORS = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
]

export function AssetAllocation() {
  const { user } = useAuth()
  const [assets, setAssets] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  // Fetch holdings data
  const fetchHoldings = async () => {
    if (!user?.id) return

    try {
      // Get complete portfolio data from portfolio-api (includes current prices, percentages, etc.)
      const portfolio = await getPortfolio(user.id)

      if (portfolio.holdings && portfolio.holdings.length > 0) {
        // Use pre-calculated holdings from backend (already in percentage format)
        const holdingsArray = portfolio.holdings.map((holding, index) => {
          const percentage = parseFloat(holding.allocation_percentage)
          console.log(`AssetAllocation DEBUG: ${holding.symbol}`, {
            allocation_percentage_raw: holding.allocation_percentage,
            percentage_parsed: percentage,
            isNaN: isNaN(percentage)
          })
          return {
            name: holding.symbol,
            quantity: parseFloat(holding.quantity),
            value: parseFloat(holding.total_value || holding.current_value),
            percentage: percentage, // API already returns as percentage
            color: COLORS[index % COLORS.length]
          }
        })

        // Add cash allocation using user balance
        const totalValue = parseFloat(portfolio.total_value) || 0
        const cash = user.initial_balance || 0  // Use initial_balance (current balance) from user
        const cashPercentage = totalValue > 0 ? (cash / totalValue) * 100 : 100

        if (cashPercentage > 0.01) { // Only show if > 0.01%
          holdingsArray.push({
            name: 'Cash',
            quantity: cash,
            value: cash,
            percentage: cashPercentage,
            color: COLORS[holdingsArray.length % COLORS.length]
          })
        }

        setAssets(holdingsArray.sort((a, b) => b.value - a.value))
      } else {
        // No holdings, show only cash
        const cash = user.initial_balance || 0
        setAssets([{
          name: 'Cash',
          quantity: cash,
          value: cash,
          percentage: 100,
          color: COLORS[0]
        }])
      }
    } catch (error) {
      console.error('Error fetching holdings:', error)
      // Fallback to cash only
      const fallbackCash = user.initial_balance || 0
      setAssets([{
        name: 'Cash',
        quantity: fallbackCash,
        value: fallbackCash,
        percentage: 100,
        color: COLORS[0]
      }])
    } finally {
      setLoading(false)
    }
  }

  // Initial fetch
  useEffect(() => {
    fetchHoldings()
  }, [user])

  // Listen for portfolio refresh events
  useEffect(() => {
    const handlePortfolioRefresh = () => {
      console.log('AssetAllocation: portfolio-refresh event received, refetching data...')
      fetchHoldings()
    }

    window.addEventListener('portfolio-refresh', handlePortfolioRefresh)

    return () => {
      window.removeEventListener('portfolio-refresh', handlePortfolioRefresh)
    }
  }, [user])

  if (loading) {
    return (
      <Card className="p-6">
        <div className="h-[400px] animate-pulse bg-muted rounded" />
      </Card>
    )
  }

  if (assets.length === 0) {
    return (
      <Card className="p-6">
        <h2 className="text-xl font-bold mb-6">Asset Allocation</h2>
        <div className="text-center py-12 text-muted-foreground">
          No holdings yet. Start trading to see your allocation here.
        </div>
      </Card>
    )
  }

  return (
    <Card className="p-6">
      <h2 className="text-xl font-bold mb-6">Asset Allocation</h2>

      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie 
            data={assets} 
            cx="50%" 
            cy="50%" 
            innerRadius={60} 
            outerRadius={90} 
            paddingAngle={2} 
            dataKey="percentage"
          >
            {assets.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip
            content={({ active, payload }) => {
              if (active && payload && payload.length) {
                const data = payload[0].payload
                return (
                  <div className="rounded-lg border bg-background p-3 shadow-sm">
                    <div className="font-semibold">{data.name}</div>
                    <div className="text-sm text-muted-foreground">
                      {data.percentage.toFixed(2)}% - ${data.value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                    </div>
                    {data.name !== 'Cash' && (
                      <div className="text-xs text-muted-foreground mt-1">
                        Qty: {data.quantity.toFixed(4)}
                      </div>
                    )}
                  </div>
                )
              }
              return null
            }}
          />
        </PieChart>
      </ResponsiveContainer>

      <div className="space-y-3 mt-6">
        {assets.map((asset) => (
          <div key={asset.name} className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="h-3 w-3 rounded-full" style={{ backgroundColor: asset.color }} />
              <span className="text-sm font-medium">{asset.name}</span>
              {asset.name !== 'Cash' && (
                <span className="text-xs text-muted-foreground">
                  {asset.quantity.toFixed(4)} {asset.name}
                </span>
              )}
            </div>
            <div className="text-right">
              <p className="text-sm font-semibold">${asset.value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</p>
              <p className="text-xs text-muted-foreground">{asset.percentage.toFixed(2)}%</p>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
