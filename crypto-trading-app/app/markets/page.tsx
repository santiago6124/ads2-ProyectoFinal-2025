"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { QuickStats } from "@/components/quick-stats"
import { TrendingUp, TrendingDown, RefreshCw, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import Link from "next/link"
import { marketApiService, PriceData } from "@/lib/market-api"

export default function MarketsPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const [lastUpdated, setLastUpdated] = useState<Date>(new Date())
  const [cryptoData, setCryptoData] = useState<PriceData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await marketApiService.getTop100()
        setCryptoData(data.slice(0, 5)) // Solo las primeras 5 cryptos
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch market data')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
    
    // Update last updated time every 30 seconds
    const interval = setInterval(() => {
      setLastUpdated(new Date())
      fetchData()
    }, 30000)
    return () => clearInterval(interval)
  }, [])

  if (isLoading || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-black">
        <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full" />
      </div>
    )
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

  const formatCurrency = (amount: number) => {
    if (amount >= 1e12) {
      return `$${(amount / 1e12).toFixed(2)}T`
    } else if (amount >= 1e9) {
      return `$${(amount / 1e9).toFixed(2)}B`
    } else if (amount >= 1e6) {
      return `$${(amount / 1e6).toFixed(2)}M`
    } else if (amount >= 1e3) {
      return `$${(amount / 1e3).toFixed(2)}K`
    } else {
      return `$${amount.toFixed(2)}`
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

  return (
    <DashboardLayout>
      <div className="space-y-8 bg-black min-h-screen p-6">
        {/* Header */}
        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <h1 className="text-4xl font-bold tracking-tight text-white">Markets</h1>
            <p className="text-white/60 mt-2 text-lg">
              Top cryptocurrencies and market overview
            </p>
            <p className="text-white/40 text-sm mt-1">
              Last updated: {lastUpdated.toLocaleTimeString()}
            </p>
          </div>
          <Button 
            variant="outline" 
            className="border-white/20 text-white hover:bg-white/10"
            onClick={() => setLastUpdated(new Date())}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>

        {/* Quick Stats - Los 4 cuadrados superiores */}
        <QuickStats />

        {/* Tabla simplificada con 5 cryptos principales */}
        <Card className="overflow-hidden bg-black border border-white/10 shadow-lg">
          <div className="p-6 border-b border-white/10">
            <h2 className="text-2xl font-bold text-white">Top Cryptocurrencies</h2>
            <p className="text-white/60 mt-1">Real-time prices and market data</p>
          </div>
          
          {loading ? (
            <div className="p-8 flex items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-white" />
              <span className="ml-2 text-white">Loading market data...</span>
            </div>
          ) : error ? (
            <div className="p-8 text-center text-red-400">
              <p className="text-lg font-semibold">Error loading market data</p>
              <p className="text-sm mt-2">{error}</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="border-b border-white/10 bg-black">
                  <tr>
                    <th className="text-left p-4 text-sm font-semibold text-white/60">#</th>
                    <th className="text-left p-4 text-sm font-semibold text-white/60">Name</th>
                    <th className="text-right p-4 text-sm font-semibold text-white/60">Price</th>
                    <th className="text-right p-4 text-sm font-semibold text-white/60">24h %</th>
                    <th className="text-right p-4 text-sm font-semibold text-white/60">Market Cap</th>
                    <th className="text-center p-4 text-sm font-semibold text-white/60">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {cryptoData.map((crypto, index) => (
                    <tr key={`${crypto.symbol}-${crypto.name}-${index}`} className="border-b border-white/10 hover:bg-white/5 transition-colors">
                      <td className="p-4">
                        <span className="text-sm text-white/60">{index + 1}</span>
                      </td>
                      <td className="p-4">
                        <div className="flex items-center gap-3">
                          <div className="h-10 w-10 rounded-full bg-white/5 flex items-center justify-center border border-white/10 overflow-hidden">
                            <img 
                              src={getCryptoIcon(crypto.symbol)}
                              alt={crypto.symbol}
                              className="h-8 w-8 rounded-full"
                              onError={(e) => {
                                const target = e.target as HTMLImageElement;
                                target.style.display = 'none';
                                const parent = target.parentElement;
                                if (parent) {
                                  parent.innerHTML = `<span class="text-sm font-bold text-white">${crypto.symbol}</span>`;
                                  parent.className = "h-10 w-10 rounded-full bg-blue-500 flex items-center justify-center border border-white/10";
                                }
                              }}
                            />
                          </div>
                          <div>
                            <p className="font-semibold text-white">{crypto.name}</p>
                            <p className="text-sm text-white/60">{crypto.symbol}</p>
                          </div>
                        </div>
                      </td>
                      <td className="p-4 text-right font-semibold text-white">{formatPrice(crypto.price)}</td>
                      <td className="p-4 text-right">
                        <div
                          className={`inline-flex items-center gap-1 font-semibold ${
                            crypto.change_24h >= 0 ? "text-green-400" : "text-red-400"
                          }`}
                        >
                          {crypto.change_24h >= 0 ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
                          {Math.abs(crypto.change_24h).toFixed(2)}%
                        </div>
                      </td>
                      <td className="p-4 text-right text-white/60">{formatCurrency(crypto.market_cap)}</td>
                      <td className="p-4 text-center">
                        <Button size="sm" asChild className="bg-blue-500 hover:bg-blue-600 text-white border-0">
                          <Link href={`/trade?coin=${crypto.symbol.toLowerCase()}`}>Trade</Link>
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      </div>
    </DashboardLayout>
  )
}