"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Wallet, DollarSign } from "lucide-react"
import { useAuth } from "@/lib/auth-context"
import { apiService } from "@/lib/api"

export function PortfolioStats() {
  const { user } = useAuth()
  const [availableCash, setAvailableCash] = useState(0)
  const [totalBalance, setTotalBalance] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchPortfolioData = async () => {
      if (!user?.id) return

      try {
        const accessToken = localStorage.getItem('crypto_access_token')
        if (!accessToken) return

        // Get cash balance from user
        const cash = user.initial_balance || 0
        setAvailableCash(cash)

        // Get orders to calculate crypto holdings value
        const ordersResponse = await apiService.getOrders(user.id, accessToken)
        const orders = ordersResponse.orders || []
        
        // Calculate total value of crypto holdings
        let cryptoValue = 0
        orders.forEach((order: any) => {
          if (order.status === 'executed' && order.type === 'buy') {
            const quantity = parseFloat(order.quantity)
            const price = parseFloat(order.order_price)
            if (quantity > 0 && price > 0) {
              cryptoValue += quantity * price
            }
          }
        })

        // Total balance = cash + crypto holdings value
        setTotalBalance(cash + cryptoValue)
      } catch (error) {
        console.error('Error fetching portfolio data:', error)
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
