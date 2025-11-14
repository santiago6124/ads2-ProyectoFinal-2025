"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { ArrowUpRight, ArrowDownRight, Loader2 } from "lucide-react"
import { useAuth } from "@/lib/auth-context"
import { searchApiService, OrderSearchResult } from "@/lib/search-api"

export function RecentActivity() {
  const { user } = useAuth()
  const [orders, setOrders] = useState<OrderSearchResult[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchRecentOrders = async () => {
      if (!user?.id) {
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        const response = await searchApiService.getRecentOrders(user.id, 4)

        console.log('🔍 Recent orders raw response:', response.results)

        // Filter to only show executed orders (double-check in case API returns others)
        const executedOrders = (response.results || []).filter(order => {
          const isExecuted = order.status === 'executed'
          if (!isExecuted) {
            console.log(`⚠️ Filtering out order ${order.id} with status: ${order.status}`)
          }
          return isExecuted
        })

        console.log('✅ Filtered executed orders:', executedOrders.length, 'orders')
        setOrders(executedOrders)
      } catch (error) {
        console.error('Failed to fetch recent orders:', error)
        setOrders([])
      } finally {
        setLoading(false)
      }
    }

    fetchRecentOrders()
  }, [user])

  const getTimeAgo = (dateStr: string): string => {
    try {
      const date = new Date(dateStr)
      const now = new Date()
      const diffMs = now.getTime() - date.getTime()
      const diffMins = Math.floor(diffMs / 60000)
      const diffHours = Math.floor(diffMs / 3600000)
      const diffDays = Math.floor(diffMs / 86400000)

      if (diffMins < 60) return `${diffMins} minute${diffMins !== 1 ? 's' : ''} ago`
      if (diffHours < 24) return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`
      return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`
    } catch {
      return 'Recently'
    }
  }

  const formatValue = (value: string): string => {
    const num = parseFloat(value)
    return isNaN(num) ? '$0.00' : `$${num.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  }

  if (loading) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 text-white animate-spin" />
        </div>
      </Card>
    )
  }

  return (
    <Card className="p-6 bg-black border border-white/10 shadow-lg">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-white">Recent Activity</h2>
        <p className="text-sm text-white/60 mt-1">Your latest transactions</p>
      </div>

      {orders.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-white/60">No recent activity</p>
          <p className="text-sm text-white/40 mt-2">Start trading to see your transactions</p>
        </div>
      ) : (
        <div className="space-y-4">
          {orders.map((order) => (
            <div key={order.id} className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300">
              <div className="flex items-center gap-4">
                <div
                  className={`h-12 w-12 rounded-xl flex items-center justify-center shadow-lg border border-white/10 ${
                    order.type === "buy" ? "bg-green-500" : "bg-red-500"
                  }`}
                >
                  {order.type === "buy" ? (
                    <ArrowDownRight className="h-6 w-6 text-white" />
                  ) : (
                    <ArrowUpRight className="h-6 w-6 text-white" />
                  )}
                </div>
                <div>
                  <p className="font-bold text-white">
                    {order.type === "buy" ? "Bought" : "Sold"} {order.crypto_symbol}
                  </p>
                  <p className="text-sm text-white/60">{getTimeAgo(order.created_at)}</p>
                </div>
              </div>
              <div className="text-right">
                <p className="font-bold text-white">{formatValue(order.total_amount)}</p>
                <p className="text-sm text-white/60">
                  {parseFloat(order.quantity).toFixed(4)} {order.crypto_symbol}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  )
}
