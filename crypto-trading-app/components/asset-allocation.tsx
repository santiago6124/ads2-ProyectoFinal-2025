"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts"
import { useAuth } from "@/lib/auth-context"
import { apiService } from "@/lib/api"

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

  useEffect(() => {
    const fetchHoldings = async () => {
      if (!user?.id) return

      try {
        const accessToken = localStorage.getItem('crypto_access_token')
        if (!accessToken) return

        // Get orders to calculate holdings
        const ordersResponse = await apiService.getOrders(user.id, accessToken)
        
        const orders = ordersResponse.orders || []
        
        if (orders && orders.length > 0) {
          const holdingsMap = new Map<string, { quantity: number; totalValue: number }>()
          
          // Calculate holdings from executed orders (buy adds, sell subtracts)
          orders.forEach((order: any) => {
            if (order.status === 'executed') {
              const quantity = parseFloat(order.quantity)
              const price = parseFloat(order.order_price)
              
              // Skip orders with invalid data (0 quantity or price)
              if (quantity > 0 && price > 0) {
                const totalValue = quantity * price
                
                const existing = holdingsMap.get(order.crypto_symbol) || { quantity: 0, totalValue: 0 }
                
                if (order.type === 'buy') {
                  // Add holdings for buy orders
                  holdingsMap.set(order.crypto_symbol, {
                    quantity: existing.quantity + quantity,
                    totalValue: existing.totalValue + totalValue
                  })
                } else if (order.type === 'sell') {
                  // Subtract holdings for sell orders
                  holdingsMap.set(order.crypto_symbol, {
                    quantity: existing.quantity - quantity,
                    totalValue: existing.totalValue - totalValue
                  })
                }
              }
            }
          })

          // Convert to array, filter out holdings with quantity or value <= 0, and calculate percentages
          const holdingsArray = Array.from(holdingsMap.entries())
            .filter(([_, data]) => data.quantity > 0 && data.totalValue > 0) // Only keep positive holdings
            .map(([symbol, data], index) => ({
              name: symbol,
              quantity: data.quantity,
              value: data.totalValue,
              percentage: 0, // Will be calculated below
              color: COLORS[index % COLORS.length]
            }))

          // Calculate total portfolio value
          const totalValue = holdingsArray.reduce((sum, asset) => sum + asset.value, 0) + user.initial_balance
          
          // Calculate percentages
          holdingsArray.forEach(asset => {
            asset.percentage = totalValue > 0 ? (asset.value / totalValue) * 100 : 0
          })

          // Add cash allocation
          const cashPercentage = totalValue > 0 ? (user.initial_balance / totalValue) * 100 : 100
          if (cashPercentage > 0.01) { // Only show if > 0.01%
            holdingsArray.push({
              name: 'Cash',
              quantity: user.initial_balance,
              value: user.initial_balance,
              percentage: cashPercentage,
              color: COLORS[holdingsArray.length % COLORS.length]
            })
          }

          setAssets(holdingsArray.sort((a, b) => b.value - a.value))
        } else {
          // No holdings, show only cash
          setAssets([{
            name: 'Cash',
            quantity: user.initial_balance,
            value: user.initial_balance,
            percentage: 100,
            color: COLORS[0]
          }])
        }
      } catch (error) {
        console.error('Error fetching holdings:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchHoldings()
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
