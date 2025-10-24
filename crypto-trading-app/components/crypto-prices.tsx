"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { marketApiService, type PriceData } from "@/lib/market-api"
import { TrendingUp, TrendingDown, Loader2 } from "lucide-react"

interface CryptoPriceProps {
  symbol: string
  showChange?: boolean
}

export function CryptoPrice({ symbol, showChange = true }: CryptoPriceProps) {
  const [priceData, setPriceData] = useState<PriceData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchPrice = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await marketApiService.getPrice(symbol)
        setPriceData(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch price')
      } finally {
        setLoading(false)
      }
    }

    fetchPrice()
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchPrice, 30000)
    return () => clearInterval(interval)
  }, [symbol])

  if (loading) {
    return (
      <Card className="p-4">
        <div className="flex items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin" />
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="p-4">
        <div className="text-center text-red-600">
          <p className="text-sm">Error loading {symbol}</p>
          <p className="text-xs">{error}</p>
        </div>
      </Card>
    )
  }

  if (!priceData) {
    return (
      <Card className="p-4">
        <div className="text-center text-gray-500">
          <p className="text-sm">No data available</p>
        </div>
      </Card>
    )
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 8,
    }).format(price)
  }

  const formatTimestamp = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleTimeString()
  }

  return (
    <Card className="p-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="font-semibold text-lg">{symbol}</h3>
          <p className="text-2xl font-bold text-slate-900">
            {formatPrice(priceData.price)}
          </p>
          <p className="text-xs text-slate-500">
            Updated: {formatTimestamp(priceData.timestamp)}
          </p>
        </div>
        {showChange && (
          <div className="flex items-center space-x-1">
            <TrendingUp className="h-5 w-5 text-green-600" />
            <span className="text-sm text-green-600">+2.5%</span>
          </div>
        )}
      </div>
      {priceData.source && (
        <p className="text-xs text-slate-400 mt-2">
          Source: {priceData.source}
        </p>
      )}
    </Card>
  )
}

interface CryptoPricesGridProps {
  symbols?: string[]
}

export function CryptoPricesGrid({ symbols = ['BTC', 'ETH', 'ADA'] }: CryptoPricesGridProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {symbols.map((symbol) => (
        <CryptoPrice key={symbol} symbol={symbol} />
      ))}
    </div>
  )
}
