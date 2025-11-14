"use client"

import { useEffect, useState, Suspense } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { apiService } from "@/lib/api"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Search, Loader2, TrendingUp, TrendingDown } from "lucide-react"
import { marketApiService, PriceData } from "@/lib/market-api"
import { ordersApiService, OrderRequest } from "@/lib/orders-api"
import { useToast } from "@/hooks/use-toast"

function TradeContent() {
  const { user, isLoading, updateUser } = useAuth()
  const router = useRouter()
  const { toast } = useToast()
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedCrypto, setSelectedCrypto] = useState<PriceData | null>(null)
  const [cryptoList, setCryptoList] = useState<PriceData[]>([])
  const [loading, setLoading] = useState(false)
  const [searchLoading, setSearchLoading] = useState(false)
  const [placing, setPlacing] = useState(false)
  const [quantity, setQuantity] = useState("")
  const [orderType, setOrderType] = useState<"buy" | "sell">("buy")

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  // Load only top 5 cryptocurrencies on initial load
  useEffect(() => {
    const fetchTopCryptos = async () => {
      try {
        setSearchLoading(true)
        const data = await marketApiService.getTop5()
        setCryptoList(data)
      } catch (error) {
        console.error('Failed to fetch top cryptocurrencies:', error)
        toast({
          title: "Error",
          description: "Failed to load top cryptocurrencies",
          variant: "destructive"
        })
      } finally {
        setSearchLoading(false)
      }
    }

    fetchTopCryptos()
  }, [toast])

  // Debounced search function
  useEffect(() => {
    const searchCryptos = async () => {
      if (searchQuery.length >= 2) {
        try {
          setSearchLoading(true)
          const results = await marketApiService.searchCryptos(searchQuery)
          setCryptoList(results)
        } catch (error) {
          console.error('Failed to search cryptocurrencies:', error)
        } finally {
          setSearchLoading(false)
        }
      } else if (searchQuery.length === 0) {
        // Reset to top 5 when search is cleared
        try {
          setSearchLoading(true)
          const data = await marketApiService.getTop5()
          setCryptoList(data)
        } catch (error) {
          console.error('Failed to fetch top cryptocurrencies:', error)
        } finally {
          setSearchLoading(false)
        }
      }
    }

    // Debounce search by 300ms
    const timeoutId = setTimeout(searchCryptos, 300)
    return () => clearTimeout(timeoutId)
  }, [searchQuery])

  const handleSearch = (query: string) => {
    setSearchQuery(query)
    // Clear selected crypto when typing
    if (query.length < 2) {
      setSelectedCrypto(null)
    }
  }

  const handleBuy = async () => {
    if (!selectedCrypto || !quantity || !user) return

    const qty = parseFloat(quantity)
    if (isNaN(qty) || qty <= 0) {
      toast({
        title: "Invalid Quantity",
        description: "Please enter a valid quantity greater than 0",
        variant: "destructive"
      })
      return
    }

    try {
      setPlacing(true)
      
      // Calculate total cost
      const totalCost = qty * selectedCrypto.price
      
      // Create order payload
      const orderData: OrderRequest = {
        type: "buy",
        crypto_symbol: selectedCrypto.symbol,
        quantity: qty.toString(),
        order_kind: "market",
        market_price: selectedCrypto.price.toString()  // Send current market price from frontend
      }
      
      console.log('=== BUY ORDER PLACED ===')
      console.log('Order Data:', orderData)
      console.log('========================')

      // Create order via Orders API
      try {
        const orderResponse = await ordersApiService.createOrder(orderData)
        console.log('Order response:', orderResponse)

        // Check order status and show appropriate message
        if (orderResponse.status === "executed") {
          // Success - order executed
          toast({
            title: "✅ Order Executed Successfully!",
            description: `Bought ${qty} ${selectedCrypto.symbol} for ${formatPrice(totalCost)}`,
          })

          // Wait for backend to process order and update balance/portfolio
          await new Promise(resolve => setTimeout(resolve, 1500))

          // Refresh user data to get updated balance
          if (user) {
            const accessToken = localStorage.getItem('crypto_access_token')
            if (accessToken) {
              try {
                const updatedUser = await apiService.getUserProfile(user.id, accessToken)
                updateUser(updatedUser)

                // Trigger portfolio refresh event for other components
                window.dispatchEvent(new CustomEvent('portfolio-refresh', {
                  detail: { userId: user.id, action: 'buy', symbol: selectedCrypto.symbol }
                }))
              } catch (error) {
                console.error('Error refreshing user data:', error)
              }
            }
          }
        } else if (orderResponse.status === "failed") {
          // Order failed - likely insufficient funds
          const failureReason = totalCost > (user?.balance || 0)
            ? `Insufficient funds. You need ${formatPrice(totalCost + (totalCost * 0.001))} (including 0.1% fee) but only have ${formatPrice(user?.balance || 0)}`
            : "Order failed to execute. Please check your balance and try again."

          toast({
            title: "❌ Order Failed",
            description: failureReason,
            variant: "destructive"
          })
          return
        } else if (orderResponse.status === "pending") {
          // Order pending - waiting for execution
          toast({
            title: "⏳ Order Pending",
            description: `Your buy order for ${qty} ${selectedCrypto.symbol} is being processed...`,
          })

          // Still refresh after a delay to check if it executed
          await new Promise(resolve => setTimeout(resolve, 2000))
          if (user) {
            const accessToken = localStorage.getItem('crypto_access_token')
            if (accessToken) {
              try {
                const updatedUser = await apiService.getUserProfile(user.id, accessToken)
                updateUser(updatedUser)
              } catch (error) {
                console.error('Error refreshing user data:', error)
              }
            }
          }
        } else if (orderResponse.status === "cancelled") {
          toast({
            title: "🚫 Order Cancelled",
            description: `Your buy order for ${qty} ${selectedCrypto.symbol} was cancelled`,
            variant: "destructive"
          })
          return
        }
      } catch (apiError) {
        console.error('API Error:', apiError)
        toast({
          title: "❌ Order Failed",
          description: apiError instanceof Error ? apiError.message : "Failed to place buy order. Please try again.",
          variant: "destructive"
        })
        return
      }

      // Reset form
      setQuantity("")
    } catch (error) {
      toast({
        title: "Order Failed",
        description: "Failed to place buy order",
        variant: "destructive"
      })
    } finally {
      setPlacing(false)
    }
  }

  const handleSell = async () => {
    if (!selectedCrypto || !quantity || !user) return

    const qty = parseFloat(quantity)
    if (isNaN(qty) || qty <= 0) {
      toast({
        title: "Invalid Quantity",
        description: "Please enter a valid quantity greater than 0",
        variant: "destructive"
      })
      return
    }

    try {
      setPlacing(true)
      
      // Calculate total value
      const totalValue = qty * selectedCrypto.price
      
      // Create order payload
      const orderData: OrderRequest = {
        type: "sell",
        crypto_symbol: selectedCrypto.symbol,
        quantity: qty.toString(),
        order_kind: "market",
        market_price: selectedCrypto.price.toString()  // Send current market price from frontend
      }
      
      console.log('=== SELL ORDER PLACED ===')
      console.log('Order Data:', orderData)
      console.log('========================')

      // Create order via Orders API
      try {
        const orderResponse = await ordersApiService.createOrder(orderData)
        console.log('Order response:', orderResponse)

        // Check order status and show appropriate message
        if (orderResponse.status === "executed") {
          // Success - order executed
          toast({
            title: "✅ Order Executed Successfully!",
            description: `Sold ${qty} ${selectedCrypto.symbol} for ${formatPrice(totalValue)}`,
          })

          // Wait for backend to process order and update balance/portfolio
          await new Promise(resolve => setTimeout(resolve, 1500))

          // Refresh user data to get updated balance
          if (user) {
            const accessToken = localStorage.getItem('crypto_access_token')
            if (accessToken) {
              try {
                const updatedUser = await apiService.getUserProfile(user.id, accessToken)
                updateUser(updatedUser)

                // Trigger portfolio refresh event for other components
                window.dispatchEvent(new CustomEvent('portfolio-refresh', {
                  detail: { userId: user.id, action: 'sell', symbol: selectedCrypto.symbol }
                }))
              } catch (error) {
                console.error('Error refreshing user data:', error)
              }
            }
          }
        } else if (orderResponse.status === "failed") {
          // Order failed - likely insufficient holdings
          toast({
            title: "❌ Order Failed",
            description: `Insufficient ${selectedCrypto.symbol} holdings. Please check your portfolio and try again.`,
            variant: "destructive"
          })
          return
        } else if (orderResponse.status === "pending") {
          // Order pending - waiting for execution
          toast({
            title: "⏳ Order Pending",
            description: `Your sell order for ${qty} ${selectedCrypto.symbol} is being processed...`,
          })

          // Still refresh after a delay to check if it executed
          await new Promise(resolve => setTimeout(resolve, 2000))
          if (user) {
            const accessToken = localStorage.getItem('crypto_access_token')
            if (accessToken) {
              try {
                const updatedUser = await apiService.getUserProfile(user.id, accessToken)
                updateUser(updatedUser)
              } catch (error) {
                console.error('Error refreshing user data:', error)
              }
            }
          }
        } else if (orderResponse.status === "cancelled") {
          toast({
            title: "🚫 Order Cancelled",
            description: `Your sell order for ${qty} ${selectedCrypto.symbol} was cancelled`,
            variant: "destructive"
          })
          return
        }
      } catch (apiError) {
        console.error('API Error:', apiError)
        toast({
          title: "❌ Order Failed",
          description: apiError instanceof Error ? apiError.message : "Failed to place sell order. Please try again.",
          variant: "destructive"
        })
        return
      }

      // Reset form
      setQuantity("")
    } catch (error) {
      toast({
        title: "Order Failed",
        description: "Failed to place sell order",
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

  const getCryptoIcon = (symbol: string) => {
    const iconMap: { [key: string]: string } = {
      'BTC': 'https://assets.coingecko.com/coins/images/1/large/bitcoin.png',
      'ETH': 'https://assets.coingecko.com/coins/images/279/large/ethereum.png',
      'BNB': 'https://assets.coingecko.com/coins/images/825/large/bnb-icon2_2x.png',
      'SOL': 'https://assets.coingecko.com/coins/images/4128/large/solana.png',
      'XRP': 'https://assets.coingecko.com/coins/images/44/large/xrp-symbol-white-128.png',
      'ADA': 'https://assets.coingecko.com/coins/images/975/large/cardano.png',
      'DOGE': 'https://assets.coingecko.com/coins/images/5/large/dogecoin.png',
      'AVAX': 'https://assets.coingecko.com/coins/images/12559/large/Avalanche_Circle_RedWhite_Trans.png',
      'DOT': 'https://assets.coingecko.com/coins/images/12171/large/polkadot.png',
      'MATIC': 'https://assets.coingecko.com/coins/images/4713/large/matic-token-icon.png'
    }
    return iconMap[symbol.toUpperCase()] || `https://assets.coingecko.com/coins/images/1/large/bitcoin.png`
  }

  if (isLoading || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-black">
        <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <DashboardLayout>
      <div className="space-y-8 bg-black min-h-screen p-6">
        {/* Header */}
        <div className="text-center">
          <h1 className="text-4xl font-bold tracking-tight text-white">Trade</h1>
          <p className="text-white/60 mt-2 text-lg">Search and trade cryptocurrencies</p>
        </div>

        {/* Search Section */}
        <Card className="p-6 bg-black border border-white/10 shadow-lg">
          <div className="space-y-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-white/60" />
              <Input
                placeholder="Type to search cryptocurrencies (e.g., Bitcoin, ETH, Doge)"
                value={searchQuery}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10 bg-black border-white/10 text-white placeholder:text-white/60 h-12 text-lg"
              />
              {searchLoading && (
                <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-white/60" />
              )}
            </div>

            {/* Search Results or Top 5 */}
            {(searchQuery.length >= 2 || cryptoList.length > 0) && (
              <div className="space-y-2">
                <p className="text-sm text-white/60 px-1">
                  {searchQuery.length >= 2 ? `Search Results (${cryptoList.length})` : 'Top 5 Cryptocurrencies'}
                </p>
                <div className="max-h-60 overflow-y-auto border border-white/10 rounded-lg">
                  {cryptoList.length > 0 ? (
                  cryptoList.map((crypto, index) => (
                    <div
                      key={`${crypto.symbol}-${crypto.name}-${index}`}
                      className="flex items-center gap-3 p-3 hover:bg-white/5 cursor-pointer border-b border-white/10 last:border-b-0"
                      onClick={() => {
                        setSelectedCrypto(crypto)
                        setSearchQuery(`${crypto.name} (${crypto.symbol})`)
                      }}
                    >
                      <div className="h-8 w-8 rounded-full bg-white/5 flex items-center justify-center border border-white/10 overflow-hidden">
                        <img 
                          src={getCryptoIcon(crypto.symbol)}
                          alt={crypto.symbol}
                          className="h-6 w-6 rounded-full"
                          onError={(e) => {
                            const target = e.target as HTMLImageElement;
                            target.style.display = 'none';
                            const parent = target.parentElement;
                            if (parent) {
                              parent.innerHTML = `<span class="text-xs font-bold text-white">${crypto.symbol}</span>`;
                              parent.className = "h-8 w-8 rounded-full bg-blue-500 flex items-center justify-center border border-white/10";
                            }
                          }}
                        />
                      </div>
                      <div className="flex-1">
                        <p className="font-semibold text-white">{crypto.name}</p>
                        <p className="text-sm text-white/60">{crypto.symbol}</p>
                      </div>
                      <div className="text-right">
                        <p className="font-semibold text-white">{formatPrice(crypto.price)}</p>
                        <div className={`flex items-center gap-1 text-sm ${
                          crypto.change_24h >= 0 ? "text-green-400" : "text-red-400"
                        }`}>
                          {crypto.change_24h >= 0 ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
                          {Math.abs(crypto.change_24h).toFixed(2)}%
                        </div>
                      </div>
                    </div>
                  ))
                  ) : (
                    <div className="p-6 text-center text-white/60">
                      <p className="text-lg">No results found</p>
                      <p className="text-sm mt-2">Try searching for a different cryptocurrency</p>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </Card>

        {/* Selected Crypto Info */}
        {selectedCrypto && (
          <Card className="p-6 bg-black border border-white/10 shadow-lg">
            <div className="text-center space-y-6">
              {/* Crypto Header */}
              <div className="flex items-center justify-center gap-4">
                <div className="h-16 w-16 rounded-full bg-white/5 flex items-center justify-center border border-white/10 overflow-hidden">
                  <img 
                    src={getCryptoIcon(selectedCrypto.symbol)}
                    alt={selectedCrypto.symbol}
                    className="h-12 w-12 rounded-full"
                    onError={(e) => {
                      const target = e.target as HTMLImageElement;
                      target.style.display = 'none';
                      const parent = target.parentElement;
                      if (parent) {
                        parent.innerHTML = `<span class="text-lg font-bold text-white">${selectedCrypto.symbol}</span>`;
                        parent.className = "h-16 w-16 rounded-full bg-blue-500 flex items-center justify-center border border-white/10";
                      }
                    }}
                  />
                </div>
                <div className="text-left">
                  <h2 className="text-2xl font-bold text-white">{selectedCrypto.name}</h2>
                  <p className="text-white/60">{selectedCrypto.symbol}</p>
                </div>
              </div>

              {/* Price Info */}
              <div className="space-y-2">
                <p className="text-4xl font-bold text-white">{formatPrice(selectedCrypto.price)}</p>
                <div className={`flex items-center justify-center gap-2 text-lg ${
                  selectedCrypto.change_24h >= 0 ? "text-green-400" : "text-red-400"
                }`}>
                  {selectedCrypto.change_24h >= 0 ? <TrendingUp className="h-5 w-5" /> : <TrendingDown className="h-5 w-5" />}
                  <span className="font-semibold">
                    {selectedCrypto.change_24h >= 0 ? '+' : ''}{selectedCrypto.change_24h.toFixed(2)}%
                  </span>
                  <span className="text-white/60">(24h)</span>
                </div>
              </div>

              {/* Quantity Input */}
              <div className="space-y-4">
                <div className="text-center">
                  <label className="block text-sm font-medium text-white/80 mb-2">
                    Quantity ({selectedCrypto.symbol})
                  </label>
                  <Input
                    type="number"
                    placeholder="Enter quantity"
                    value={quantity}
                    onChange={(e) => setQuantity(e.target.value)}
                    className="w-64 mx-auto bg-black border-white/10 text-white placeholder:text-white/60 text-center text-lg"
                    min="0"
                    step="0.000001"
                  />
                </div>
                
                {quantity && !isNaN(parseFloat(quantity)) && parseFloat(quantity) > 0 && (
                  <div className="text-center space-y-2">
                    <p className="text-white/60 text-sm">Total Cost</p>
                    <p className="text-2xl font-bold text-white">
                      {formatPrice(parseFloat(quantity) * selectedCrypto.price)}
                    </p>
                  </div>
                )}
              </div>

              {/* Action Buttons */}
              <div className="flex gap-4 justify-center">
                <Button
                  size="lg"
                  className="bg-green-600 hover:bg-green-700 text-white px-8 py-3 text-lg font-semibold"
                  onClick={handleBuy}
                  disabled={placing || !quantity || isNaN(parseFloat(quantity)) || parseFloat(quantity) <= 0}
                >
                  {placing ? (
                    <>
                      <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                      Processing...
                    </>
                  ) : (
                    `Buy ${selectedCrypto.symbol}`
                  )}
                </Button>
                <Button
                  size="lg"
                  className="bg-red-600 hover:bg-red-700 text-white px-8 py-3 text-lg font-semibold"
                  onClick={handleSell}
                  disabled={placing || !quantity || isNaN(parseFloat(quantity)) || parseFloat(quantity) <= 0}
                >
                  {placing ? (
                    <>
                      <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                      Processing...
                    </>
                  ) : (
                    `Sell ${selectedCrypto.symbol}`
                  )}
                </Button>
              </div>
            </div>
          </Card>
        )}
      </div>
    </DashboardLayout>
  )
}

export default function TradePage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center bg-black">
          <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full" />
        </div>
      }
    >
      <TradeContent />
    </Suspense>
  )
}
