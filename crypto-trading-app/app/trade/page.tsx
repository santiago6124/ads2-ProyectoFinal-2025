"use client"

import { useEffect, useState, Suspense } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Search, Loader2, TrendingUp, TrendingDown } from "lucide-react"
import { marketApiService, PriceData } from "@/lib/market-api"
import { useToast } from "@/hooks/use-toast"

function TradeContent() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const { toast } = useToast()
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedCrypto, setSelectedCrypto] = useState<PriceData | null>(null)
  const [cryptoList, setCryptoList] = useState<PriceData[]>([])
  const [loading, setLoading] = useState(false)
  const [searchLoading, setSearchLoading] = useState(false)
  const [placing, setPlacing] = useState(false)

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  useEffect(() => {
    const fetchCryptoList = async () => {
      try {
        setSearchLoading(true)
        const data = await marketApiService.getTop100()
        setCryptoList(data)
      } catch (error) {
        console.error('Failed to fetch crypto list:', error)
        toast({
          title: "Error",
          description: "Failed to load cryptocurrency list",
          variant: "destructive"
        })
      } finally {
        setSearchLoading(false)
      }
    }

    fetchCryptoList()
  }, [toast])

  const handleSearch = (query: string) => {
    setSearchQuery(query)
    if (query.length >= 2) {
      const filtered = cryptoList.filter(crypto =>
        crypto.name.toLowerCase().includes(query.toLowerCase()) ||
        crypto.symbol.toLowerCase().includes(query.toLowerCase())
      )
      if (filtered.length > 0) {
        setSelectedCrypto(filtered[0])
      }
    } else {
      setSelectedCrypto(null)
    }
  }

  const handleBuy = async () => {
    if (!selectedCrypto) return

    try {
      setPlacing(true)
      
      // Simulate buy order
      const orderId = `buy_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
      
      console.log('=== BUY ORDER PLACED ===')
      console.log('Order ID:', orderId)
      console.log('Crypto:', selectedCrypto.symbol)
      console.log('Price:', selectedCrypto.price)
      console.log('User ID:', user?.id)
      console.log('========================')

      toast({
        title: "Buy Order Placed",
        description: `Successfully placed buy order for ${selectedCrypto.symbol}`,
      })
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
    if (!selectedCrypto) return

    try {
      setPlacing(true)
      
      // Simulate sell order
      const orderId = `sell_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
      
      console.log('=== SELL ORDER PLACED ===')
      console.log('Order ID:', orderId)
      console.log('Crypto:', selectedCrypto.symbol)
      console.log('Price:', selectedCrypto.price)
      console.log('User ID:', user?.id)
      console.log('========================')

      toast({
        title: "Sell Order Placed",
        description: `Successfully placed sell order for ${selectedCrypto.symbol}`,
      })
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
                placeholder="Search for a cryptocurrency (e.g., Bitcoin, BTC, Ethereum)"
                value={searchQuery}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10 bg-black border-white/10 text-white placeholder:text-white/60 h-12 text-lg"
              />
              {searchLoading && (
                <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-white/60" />
              )}
            </div>

            {/* Search Results */}
            {searchQuery.length >= 2 && (
              <div className="max-h-60 overflow-y-auto border border-white/10 rounded-lg">
                {cryptoList
                  .filter(crypto =>
                    crypto.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                    crypto.symbol.toLowerCase().includes(searchQuery.toLowerCase())
                  )
                  .slice(0, 10)
                  .map((crypto) => (
                    <div
                      key={crypto.symbol}
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
                  ))}
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

              {/* Action Buttons */}
              <div className="flex gap-4 justify-center">
                <Button
                  size="lg"
                  className="bg-green-600 hover:bg-green-700 text-white px-8 py-3 text-lg font-semibold"
                  onClick={handleBuy}
                  disabled={placing}
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
                  disabled={placing}
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
