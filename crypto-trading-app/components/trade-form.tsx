"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAuth } from "@/lib/auth-context"
import { marketApiService } from "@/lib/market-api"
import { getPortfolio, type Portfolio, type Holding } from "@/lib/portfolio-api"
import { ordersApiService, type OrderRequest } from "@/lib/orders-api"
import { ArrowDownUp, Loader2, CheckCircle2 } from "lucide-react"
import { useToast } from "@/hooks/use-toast"

interface TradeFormProps {
  coin: string
}

interface OrderLog {
  id: string
  type: 'buy' | 'sell'
  coin: string
  amount: number
  price: number
  total: number
  timestamp: Date
}

export function TradeForm({ coin }: TradeFormProps) {
  const { user } = useAuth()
  const { toast } = useToast()
  const [buyAmount, setBuyAmount] = useState("")
  const [sellAmount, setSellAmount] = useState("")
  const [buyTotal, setBuyTotal] = useState("")
  const [sellTotal, setSellTotal] = useState("")
  const [currentPrice, setCurrentPrice] = useState<number>(0)
  const [loading, setLoading] = useState(true)
  const [placing, setPlacing] = useState(false)
  const [orderLogs, setOrderLogs] = useState<OrderLog[]>([])
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null)
  const [loadingPortfolio, setLoadingPortfolio] = useState(true)
  const [currentHolding, setCurrentHolding] = useState<Holding | null>(null)

  // Fetch portfolio holdings
  useEffect(() => {
    const fetchPortfolio = async () => {
      if (!user?.id) return

      try {
        setLoadingPortfolio(true)
        const portfolioData = await getPortfolio(user.id)
        setPortfolio(portfolioData)

        // Find holding for current coin
        const holding = portfolioData.holdings.find(
          h => h.symbol.toLowerCase() === coin.toLowerCase()
        )
        setCurrentHolding(holding || null)
      } catch (error) {
        console.error('Failed to fetch portfolio:', error)
      } finally {
        setLoadingPortfolio(false)
      }
    }

    fetchPortfolio()

    // Refresh portfolio every 30 seconds
    const interval = setInterval(fetchPortfolio, 30000)
    return () => clearInterval(interval)
  }, [user?.id, coin])

  // Fetch current price for the selected coin
  useEffect(() => {
    const fetchPrice = async () => {
      try {
        setLoading(true)
        const symbol = coin.toUpperCase()
        const data = await marketApiService.getPrice(symbol)
        setCurrentPrice(data.price)
      } catch (error) {
        console.error('Failed to fetch price:', error)
        toast({
          title: "Error",
          description: "Failed to fetch current price",
          variant: "destructive"
        })
      } finally {
        setLoading(false)
      }
    }

    fetchPrice()

    // Refresh price every 10 seconds
    const interval = setInterval(fetchPrice, 10000)
    return () => clearInterval(interval)
  }, [coin, toast])

  const handleBuyAmountChange = (value: string) => {
    setBuyAmount(value)
    const total = Number.parseFloat(value) * currentPrice
    setBuyTotal(isNaN(total) ? "" : total.toFixed(2))
  }

  const handleBuyTotalChange = (value: string) => {
    setBuyTotal(value)
    const amount = Number.parseFloat(value) / currentPrice
    setBuyAmount(isNaN(amount) ? "" : amount.toFixed(8))
  }

  const handleSellAmountChange = (value: string) => {
    setSellAmount(value)
    const total = Number.parseFloat(value) * currentPrice
    setSellTotal(isNaN(total) ? "" : total.toFixed(2))
  }

  const handleBuy = async () => {
    if (!buyAmount || Number.parseFloat(buyAmount) <= 0) {
      toast({
        title: "Invalid amount",
        description: "Please enter a valid amount to buy",
        variant: "destructive"
      })
      return
    }

    const amount = Number.parseFloat(buyAmount)
    const total = Number.parseFloat(buyTotal)

    if (total > (user?.balance || 0)) {
      toast({
        title: "Insufficient balance",
        description: `You need $${total.toFixed(2)} but only have $${user?.balance.toFixed(2)}`,
        variant: "destructive"
      })
      return
    }

    try {
      setPlacing(true)

      // Create order via API
      const orderRequest: OrderRequest = {
        type: "buy",
        crypto_symbol: coin.toUpperCase(),
        quantity: buyAmount,
        order_kind: "market",
        market_price: currentPrice.toString()
      }

      const orderResponse = await ordersApiService.createOrder(orderRequest)

      // Create order log
      const orderLog: OrderLog = {
        id: orderResponse.id,
        type: 'buy',
        coin: coin.toUpperCase(),
        amount: amount,
        price: currentPrice,
        total: total,
        timestamp: new Date()
      }

      // Add to order logs
      setOrderLogs(prev => [orderLog, ...prev])

      // Show success toast
      toast({
        title: "Order Placed Successfully",
        description: (
          <div className="space-y-1">
            <p><strong>Order ID:</strong> {orderResponse.id}</p>
            <p>Bought {amount} {coin.toUpperCase()} for ${total.toFixed(2)}</p>
            <p className="text-xs text-muted-foreground">Status: {orderResponse.status}</p>
          </div>
        )
      })

      // Reset form
      setBuyAmount("")
      setBuyTotal("")

      // Refresh portfolio to show updated holdings
      if (user?.id) {
        setTimeout(async () => {
          try {
            const portfolioData = await getPortfolio(user.id)
            setPortfolio(portfolioData)
            const holding = portfolioData.holdings.find(
              h => h.symbol.toLowerCase() === coin.toLowerCase()
            )
            setCurrentHolding(holding || null)
          } catch (error) {
            console.error('Failed to refresh portfolio:', error)
          }
        }, 2000)
      }
    } catch (error) {
      console.error('Failed to place order:', error)
      const errorMessage = error instanceof Error ? error.message : "Failed to place buy order. Please try again."
      toast({
        title: "Order Failed",
        description: errorMessage,
        variant: "destructive"
      })
    } finally {
      setPlacing(false)
    }
  }

  const handleSell = async () => {
    if (!sellAmount || Number.parseFloat(sellAmount) <= 0) {
      toast({
        title: "Invalid amount",
        description: "Please enter a valid amount to sell",
        variant: "destructive"
      })
      return
    }

    const amount = Number.parseFloat(sellAmount)
    const total = Number.parseFloat(sellTotal)

    // Validate holdings - THIS IS THE KEY FIX
    const availableAmount = currentHolding ? Number.parseFloat(currentHolding.quantity) : 0

    if (amount > availableAmount) {
      toast({
        title: "Insufficient holdings",
        description: `You need ${amount} ${coin.toUpperCase()} but only have ${availableAmount.toFixed(8)} ${coin.toUpperCase()}`,
        variant: "destructive"
      })
      return
    }

    try {
      setPlacing(true)

      // Create order via API
      const orderRequest: OrderRequest = {
        type: "sell",
        crypto_symbol: coin.toUpperCase(),
        quantity: sellAmount,
        order_kind: "market",
        market_price: currentPrice.toString()
      }

      const orderResponse = await ordersApiService.createOrder(orderRequest)

      // Create order log
      const orderLog: OrderLog = {
        id: orderResponse.id,
        type: 'sell',
        coin: coin.toUpperCase(),
        amount: amount,
        price: currentPrice,
        total: total,
        timestamp: new Date()
      }

      // Add to order logs
      setOrderLogs(prev => [orderLog, ...prev])

      // Show success toast
      toast({
        title: "Order Placed Successfully",
        description: (
          <div className="space-y-1">
            <p><strong>Order ID:</strong> {orderResponse.id}</p>
            <p>Sold {amount} {coin.toUpperCase()} for ${total.toFixed(2)}</p>
            <p className="text-xs text-muted-foreground">Status: {orderResponse.status}</p>
          </div>
        )
      })

      // Reset form
      setSellAmount("")
      setSellTotal("")

      // Refresh portfolio to show updated holdings
      if (user?.id) {
        setTimeout(async () => {
          try {
            const portfolioData = await getPortfolio(user.id)
            setPortfolio(portfolioData)
            const holding = portfolioData.holdings.find(
              h => h.symbol.toLowerCase() === coin.toLowerCase()
            )
            setCurrentHolding(holding || null)
          } catch (error) {
            console.error('Failed to refresh portfolio:', error)
          }
        }, 2000)
      }
    } catch (error) {
      console.error('Failed to place order:', error)
      const errorMessage = error instanceof Error ? error.message : "Failed to place sell order. Please try again."
      toast({
        title: "Order Failed",
        description: errorMessage,
        variant: "destructive"
      })
    } finally {
      setPlacing(false)
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

  return (
    <div className="space-y-4">
      <Card className="p-6">
        <Tabs defaultValue="buy" className="w-full">
          <TabsList className="grid w-full grid-cols-2 mb-6">
            <TabsTrigger value="buy">Buy</TabsTrigger>
            <TabsTrigger value="sell">Sell</TabsTrigger>
          </TabsList>

          <TabsContent value="buy" className="space-y-4">
            <div className="p-4 rounded-lg bg-accent/50 border border-border">
              <div className="flex items-center justify-between mb-1">
                <span className="text-sm text-muted-foreground">Available Balance</span>
                <span className="text-sm font-semibold">${(user?.balance || 0).toLocaleString()}</span>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="buy-amount">Amount ({coin.toUpperCase()})</Label>
              <Input
                id="buy-amount"
                type="number"
                placeholder="0.00"
                value={buyAmount}
                onChange={(e) => handleBuyAmountChange(e.target.value)}
                step="0.00000001"
                disabled={loading}
              />
            </div>

            <div className="flex justify-center">
              <div className="h-8 w-8 rounded-full bg-accent flex items-center justify-center">
                <ArrowDownUp className="h-4 w-4 text-muted-foreground" />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="buy-total">Total (USD)</Label>
              <Input
                id="buy-total"
                type="number"
                placeholder="0.00"
                value={buyTotal}
                onChange={(e) => handleBuyTotalChange(e.target.value)}
                step="0.01"
                disabled={loading}
              />
            </div>

            <div className="flex gap-2">
              {[25, 50, 75, 100].map((percent) => (
                <Button
                  key={percent}
                  variant="outline"
                  size="sm"
                  className="flex-1 bg-transparent"
                  onClick={() => {
                    const total = (user?.balance || 0) * (percent / 100)
                    handleBuyTotalChange(total.toString())
                  }}
                  disabled={loading}
                >
                  {percent}%
                </Button>
              ))}
            </div>

            <div className="p-4 rounded-lg bg-accent/50 border border-border space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Price</span>
                {loading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <span className="font-medium">{formatPrice(currentPrice)}</span>
                )}
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Fee (0.1%)</span>
                <span className="font-medium">${(Number.parseFloat(buyTotal || "0") * 0.001).toFixed(2)}</span>
              </div>
            </div>

            <Button
              className="w-full h-12 text-base font-semibold bg-green-600 hover:bg-green-700"
              onClick={handleBuy}
              disabled={loading || placing}
            >
              {placing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Placing Order...
                </>
              ) : (
                `Buy ${coin.toUpperCase()}`
              )}
            </Button>
          </TabsContent>

          <TabsContent value="sell" className="space-y-4">
            <div className="p-4 rounded-lg bg-accent/50 border border-border">
              <div className="flex items-center justify-between mb-1">
                <span className="text-sm text-muted-foreground">Available {coin.toUpperCase()}</span>
                {loadingPortfolio ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <span className="text-sm font-semibold">
                    {currentHolding ? Number.parseFloat(currentHolding.quantity).toFixed(8) : '0.00000000'} {coin.toUpperCase()}
                  </span>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="sell-amount">Amount ({coin.toUpperCase()})</Label>
              <Input
                id="sell-amount"
                type="number"
                placeholder="0.00"
                value={sellAmount}
                onChange={(e) => handleSellAmountChange(e.target.value)}
                step="0.00000001"
                disabled={loading}
              />
            </div>

            <div className="flex justify-center">
              <div className="h-8 w-8 rounded-full bg-accent flex items-center justify-center">
                <ArrowDownUp className="h-4 w-4 text-muted-foreground" />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="sell-total">Total (USD)</Label>
              <Input id="sell-total" type="number" placeholder="0.00" value={sellTotal} readOnly disabled={loading} />
            </div>

            <div className="flex gap-2">
              {[25, 50, 75, 100].map((percent) => (
                <Button
                  key={percent}
                  variant="outline"
                  size="sm"
                  className="flex-1 bg-transparent"
                  onClick={() => {
                    const availableAmount = currentHolding ? Number.parseFloat(currentHolding.quantity) : 0
                    const amount = availableAmount * (percent / 100)
                    handleSellAmountChange(amount.toString())
                  }}
                  disabled={loading || loadingPortfolio || !currentHolding}
                >
                  {percent}%
                </Button>
              ))}
            </div>

            <div className="p-4 rounded-lg bg-accent/50 border border-border space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Price</span>
                {loading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <span className="font-medium">{formatPrice(currentPrice)}</span>
                )}
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Fee (0.1%)</span>
                <span className="font-medium">${(Number.parseFloat(sellTotal || "0") * 0.001).toFixed(2)}</span>
              </div>
            </div>

            <Button
              className="w-full h-12 text-base font-semibold bg-red-600 hover:bg-red-700"
              onClick={handleSell}
              disabled={loading || placing}
            >
              {placing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Placing Order...
                </>
              ) : (
                `Sell ${coin.toUpperCase()}`
              )}
            </Button>
          </TabsContent>
        </Tabs>
      </Card>

      {/* Order Logs */}
      {orderLogs.length > 0 && (
        <Card className="p-4">
          <div className="flex items-center gap-2 mb-3">
            <CheckCircle2 className="h-5 w-5 text-green-500" />
            <h3 className="font-semibold">Recent Orders</h3>
          </div>
          <div className="space-y-2 max-h-[300px] overflow-auto">
            {orderLogs.map((order) => (
              <div
                key={order.id}
                className={`p-3 rounded-lg border ${
                  order.type === 'buy'
                    ? 'bg-green-500/10 border-green-500/30'
                    : 'bg-red-500/10 border-red-500/30'
                }`}
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex-1">
                    <p className="font-mono text-xs text-muted-foreground mb-1">{order.id}</p>
                    <p className="font-semibold text-sm">
                      {order.type === 'buy' ? '🟢 BUY' : '🔴 SELL'} {order.amount} {order.coin}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold">${order.total.toFixed(2)}</p>
                    <p className="text-xs text-muted-foreground">{formatPrice(order.price)}/unit</p>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  {order.timestamp?.toLocaleString() || 'Unknown time'}
                </p>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}
