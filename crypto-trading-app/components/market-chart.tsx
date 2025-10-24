"use client"

import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis, ResponsiveContainer } from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"
import { marketApiService, PriceData } from "@/lib/market-api"
import { Loader2 } from "lucide-react"

const timeframes = ["1H", "24H", "7D", "30D", "1Y", "ALL"]

export function MarketChart() {
  const [selectedTimeframe, setSelectedTimeframe] = useState("24H")
  const [btcData, setBtcData] = useState<PriceData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchBtcData = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await marketApiService.getPrice('BTC')
        setBtcData(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch BTC data')
      } finally {
        setLoading(false)
      }
    }

    fetchBtcData()
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchBtcData, 30000)
    return () => clearInterval(interval)
  }, [])

  // Generate mock chart data for now (in a real app, this would come from historical data API)
  const generateChartData = (points: number) => {
    const data = []
    const now = Date.now()
    let basePrice = btcData?.price || 50000

    for (let i = points; i >= 0; i--) {
      const change = (Math.random() - 0.5) * 2000
      basePrice += change
      data.push({
        time: new Date(now - i * 3600000).toLocaleTimeString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        price: Math.max(basePrice, 40000),
      })
    }
    return data
  }

  const chartData = generateChartData(24)

  const formatPrice = (price: number) => {
    if (price >= 1000) {
      return `$${price.toLocaleString()}`
    } else {
      return `$${price.toFixed(2)}`
    }
  }

  if (loading) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="flex items-center justify-center h-[400px]">
          <Loader2 className="h-8 w-8 animate-spin text-white" />
          <span className="ml-2 text-white">Loading chart data...</span>
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="p-6 bg-black border border-white/10 shadow-lg">
        <div className="flex items-center justify-center h-[400px] text-red-400">
          <div className="text-center">
            <p className="text-lg font-semibold">Error loading chart data</p>
            <p className="text-sm mt-2">{error}</p>
          </div>
        </div>
      </Card>
    )
  }

  return (
    <Card className="p-6 bg-black border border-white/10 shadow-lg">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h2 className="text-2xl font-bold text-white">Bitcoin (BTC)</h2>
          <div className="flex items-baseline gap-3 mt-2">
            <span className="text-3xl font-bold text-white">
              {btcData ? formatPrice(btcData.price) : '$0.00'}
            </span>
            <span className={`font-semibold ${btcData && btcData.change_24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
              {btcData ? `${btcData.change_24h >= 0 ? '+' : ''}${btcData.change_24h.toFixed(2)}%` : '0.00%'}
            </span>
          </div>
        </div>
        <div className="flex gap-2">
          {timeframes.map((tf) => (
            <Button
              key={tf}
              variant={selectedTimeframe === tf ? "default" : "outline"}
              size="sm"
              onClick={() => setSelectedTimeframe(tf)}
              className={`text-xs ${
                selectedTimeframe === tf 
                  ? "bg-blue-500 text-white border-blue-500" 
                  : "bg-transparent text-white/70 border-white/20 hover:bg-white/10"
              }`}
            >
              {tf}
            </Button>
          ))}
        </div>
      </div>

      <ChartContainer
        config={{
          price: {
            label: "Price",
            color: "hsl(217, 91%, 60%)",
          },
        }}
        className="h-[400px] w-full"
      >
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData}>
            <defs>
              <linearGradient id="colorPrice" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="rgb(59, 130, 246)" stopOpacity={0.3} />
                <stop offset="95%" stopColor="rgb(59, 130, 246)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="rgba(255, 255, 255, 0.1)" opacity={0.3} />
            <XAxis
              dataKey="time"
              stroke="rgba(255, 255, 255, 0.6)"
              fontSize={12}
              tickLine={false}
              axisLine={false}
            />
            <YAxis
              stroke="rgba(255, 255, 255, 0.6)"
              fontSize={12}
              tickLine={false}
              axisLine={false}
              tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`}
            />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area
              type="monotone"
              dataKey="price"
              stroke="rgb(59, 130, 246)"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorPrice)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartContainer>
    </Card>
  )
}
