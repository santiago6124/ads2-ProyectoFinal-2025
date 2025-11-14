"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ArrowUpRight, ArrowDownRight, Search, Download, Loader2 } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { useAuth } from "@/lib/auth-context"
import { searchApiService, OrderSearchResult } from "@/lib/search-api"
import { useToast } from "@/hooks/use-toast"

export function TransactionHistory() {
  const { user } = useAuth()
  const { toast } = useToast()
  const [searchQuery, setSearchQuery] = useState("")
  const [filter, setFilter] = useState<"all" | "buy" | "sell">("all")
  const [orders, setOrders] = useState<OrderSearchResult[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)

  useEffect(() => {
    fetchOrders()
  }, [user, filter, page])

  const fetchOrders = async () => {
    if (!user?.id) return

    try {
      setLoading(true)
      const typeFilter = filter === "all" ? undefined : [filter]

      const response = await searchApiService.searchOrders({
        user_id: user.id,
        type: typeFilter,
        q: searchQuery || undefined,
        page,
        limit: 20,
        sort: 'created_at_desc'
      })

      setOrders(response.results || [])
      setTotalPages(response.total_pages || 1)
    } catch (error) {
      console.error('Failed to fetch orders:', error)
      toast({
        title: "Error",
        description: "Failed to load transaction history",
        variant: "destructive"
      })
      setOrders([])
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = () => {
    setPage(1)
    fetchOrders()
  }

  const formatPrice = (price: string) => {
    const num = parseFloat(price)
    return isNaN(num) ? '$0.00' : `$${num.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  }

  const formatDate = (dateStr: string) => {
    try {
      const date = new Date(dateStr)
      return {
        date: date.toLocaleDateString('en-US', { year: 'numeric', month: '2-digit', day: '2-digit' }),
        time: date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
      }
    } catch {
      return { date: 'N/A', time: 'N/A' }
    }
  }

  const getStatusVariant = (status: string): "default" | "secondary" | "destructive" => {
    switch (status.toLowerCase()) {
      case 'executed':
        return 'default'
      case 'pending':
        return 'secondary'
      case 'failed':
      case 'cancelled':
        return 'destructive'
      default:
        return 'secondary'
    }
  }

  return (
    <Card className="p-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h2 className="text-xl font-bold">Transaction History</h2>
          <p className="text-sm text-muted-foreground mt-1">View all your trading activity</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search transactions..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              className="pl-10 w-full sm:w-64"
            />
          </div>
          <Button variant="outline" size="icon" onClick={handleSearch} disabled={loading}>
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
          </Button>
        </div>
      </div>

      <div className="flex gap-2 mb-6">
        <Button
          variant={filter === "all" ? "default" : "outline"}
          size="sm"
          onClick={() => { setFilter("all"); setPage(1); }}
          className="bg-transparent"
        >
          All
        </Button>
        <Button
          variant={filter === "buy" ? "default" : "outline"}
          size="sm"
          onClick={() => { setFilter("buy"); setPage(1); }}
          className="bg-transparent"
        >
          Buy
        </Button>
        <Button
          variant={filter === "sell" ? "default" : "outline"}
          size="sm"
          onClick={() => { setFilter("sell"); setPage(1); }}
          className="bg-transparent"
        >
          Sell
        </Button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="border-b border-border">
                <tr>
                  <th className="text-left p-4 text-sm font-semibold text-muted-foreground">Type</th>
                  <th className="text-left p-4 text-sm font-semibold text-muted-foreground">Asset</th>
                  <th className="text-right p-4 text-sm font-semibold text-muted-foreground">Amount</th>
                  <th className="text-right p-4 text-sm font-semibold text-muted-foreground hidden md:table-cell">Price</th>
                  <th className="text-right p-4 text-sm font-semibold text-muted-foreground">Value</th>
                  <th className="text-center p-4 text-sm font-semibold text-muted-foreground hidden lg:table-cell">
                    Status
                  </th>
                  <th className="text-right p-4 text-sm font-semibold text-muted-foreground hidden xl:table-cell">
                    Date & Time
                  </th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => {
                  const { date, time } = formatDate(order.created_at)
                  return (
                    <tr key={order.id} className="border-b border-border hover:bg-accent/50 transition-colors">
                      <td className="p-4">
                        <div
                          className={`inline-flex items-center gap-2 px-3 py-1 rounded-full ${
                            order.type === "buy" ? "bg-green-500/10 text-green-500" : "bg-red-500/10 text-red-500"
                          }`}
                        >
                          {order.type === "buy" ? <ArrowDownRight className="h-4 w-4" /> : <ArrowUpRight className="h-4 w-4" />}
                          <span className="text-sm font-semibold capitalize">{order.type}</span>
                        </div>
                      </td>
                      <td className="p-4">
                        <div className="flex items-center gap-3">
                          <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                            <span className="text-xs font-bold text-primary">{order.crypto_symbol}</span>
                          </div>
                          <div>
                            <p className="font-semibold">{order.crypto_name}</p>
                            <p className="text-xs text-muted-foreground">{order.crypto_symbol}</p>
                          </div>
                        </div>
                      </td>
                      <td className="p-4 text-right font-medium">
                        {parseFloat(order.quantity).toFixed(4)} {order.crypto_symbol}
                      </td>
                      <td className="p-4 text-right text-muted-foreground hidden md:table-cell">
                        {formatPrice(order.price)}
                      </td>
                      <td className="p-4 text-right font-semibold">{formatPrice(order.total_amount)}</td>
                      <td className="p-4 text-center hidden lg:table-cell">
                        <Badge variant={getStatusVariant(order.status)} className="capitalize">
                          {order.status}
                        </Badge>
                      </td>
                      <td className="p-4 text-right text-sm text-muted-foreground hidden xl:table-cell">
                        <div>{date}</div>
                        <div className="text-xs">{time}</div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          {orders.length === 0 && !loading && (
            <div className="text-center py-12">
              <p className="text-muted-foreground">No transactions found</p>
              <p className="text-sm text-muted-foreground mt-2">Start trading to see your transaction history</p>
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-6">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1 || loading}
              >
                Previous
              </Button>
              <span className="text-sm text-muted-foreground">
                Page {page} of {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages || loading}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}
    </Card>
  )
}
