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
  const [chartData, setChartData] = useState<Array<{time: string, price: number}>>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Map timeframe to API interval
  const getIntervalFromTimeframe = (tf: string) => {
    switch(tf) {
      case "1H": return { interval: "1m", limit: 60 }
      case "24H": return { interval: "1h", limit: 24 }
      case "7D": return { interval: "4h", limit: 42 }
      case "30D": return { interval: "1d", limit: 30 }
      case "1Y": return { interval: "1w", limit: 52 }
      case "ALL": return { interval: "1w", limit: 104 }
      default: return { interval: "1h", limit: 24 }
    }
  }

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

    // Refresh every 60 seconds (changed from 30 to reduce updates)
    const interval = setInterval(fetchBtcData, 60000)
    return () => clearInterval(interval)
  }, [])

  // Fetch chart data when timeframe changes
  useEffect(() => {
    const fetchChartData = async () => {
      try {
        const { interval, limit } = getIntervalFromTimeframe(selectedTimeframe)
        const history = await marketApiService.getPriceHistory('BTC', interval)

        // Format the data for the chart
        const formatted = history.history.slice(-limit).map((item: any) => {
          const date = new Date(item.timestamp * 1000)
          let timeFormat = {}

          // Adjust time format based on timeframe
          if (selectedTimeframe === "1H") {
            timeFormat = { hour: "2-digit", minute: "2-digit" }
          } else if (selectedTimeframe === "24H") {
            timeFormat = { hour: "2-digit", minute: "2-digit" }
          } else if (selectedTimeframe === "7D" || selectedTimeframe === "30D") {
            timeFormat = { month: "short", day: "numeric" }
          } else {
            timeFormat = { month: "short", year: "2-digit" }
          }

          return {
            time: date.toLocaleString("en-US", timeFormat as any),
            price: item.price,
          }
        })

        setChartData(formatted)
      } catch (err) {
        console.error('Failed to fetch chart data:', err)
      }
    }

    fetchChartData()
  }, [selectedTimeframe])

  const formatPrice = (price: number) => {
    if (price >= 1000) {
      return `$${price.toLocaleString()}`
    } else {
      return `$${price.toFixed(2)}`
    }
  }

  // Calculate dynamic Y-axis domain based on chart data
  const getYAxisDomain = () => {
    if (chartData.length === 0) return [0, 100000]

    const prices = chartData.map(d => d.price)
    const minPrice = Math.min(...prices)
    const maxPrice = Math.max(...prices)

    // Add padding (5% on each side) for better visualization
    const padding = (maxPrice - minPrice) * 0.05
    const domainMin = Math.max(0, minPrice - padding)
    const domainMax = maxPrice + padding

    return [domainMin, domainMax]
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
              domain={getYAxisDomain()}
              tickFormatter={(value) => {
                if (value >= 1000) {
                  return `$${(value / 1000).toFixed(1)}k`
                } else {
                  return `$${value.toFixed(2)}`
                }
              }}
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
