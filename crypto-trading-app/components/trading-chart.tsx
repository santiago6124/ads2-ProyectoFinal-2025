"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis, ResponsiveContainer } from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"
import { TrendingUp } from "lucide-react"

const timeframes = ["1m", "5m", "15m", "1h", "4h", "1d"]

const generateChartData = (points: number) => {
  const data = []
  const now = Date.now()
  let basePrice = 44000

  for (let i = points; i >= 0; i--) {
    const change = (Math.random() - 0.5) * 500
    basePrice += change
    data.push({
      time: new Date(now - i * 60000).toLocaleTimeString("en-US", {
        hour: "2-digit",
        minute: "2-digit",
      }),
      price: Math.max(basePrice, 43000),
      volume: Math.random() * 1000000,
    })
  }
  return data
}

interface TradingChartProps {
  coin: string
}

export function TradingChart({ coin }: TradingChartProps) {
  const [selectedTimeframe, setSelectedTimeframe] = useState("15m")
  const [chartData] = useState(generateChartData(50))

  const currentPrice = chartData[chartData.length - 1]?.price || 44000
  const previousPrice = chartData[0]?.price || 43000
  const priceChange = ((currentPrice - previousPrice) / previousPrice) * 100

  return (
    <Card className="p-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <div className="flex items-center gap-2 mb-2">
            <h2 className="text-2xl font-bold">Bitcoin</h2>
            <span className="text-sm text-muted-foreground">BTC/USD</span>
          </div>
          <div className="flex items-baseline gap-3">
            <span className="text-3xl font-bold">${currentPrice.toFixed(2)}</span>
            <span
              className={`flex items-center gap-1 font-semibold ${priceChange >= 0 ? "text-green-500" : "text-red-500"}`}
            >
              <TrendingUp className="h-4 w-4" />
              {priceChange >= 0 ? "+" : ""}
              {priceChange.toFixed(2)}%
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
              className="text-xs"
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
            color: "hsl(var(--primary))",
          },
        }}
        className="h-[500px] w-full"
      >
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData}>
            <defs>
              <linearGradient id="tradingColorPrice" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3} />
                <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.3} />
            <XAxis
              dataKey="time"
              stroke="hsl(var(--muted-foreground))"
              fontSize={12}
              tickLine={false}
              axisLine={false}
            />
            <YAxis
              stroke="hsl(var(--muted-foreground))"
              fontSize={12}
              tickLine={false}
              axisLine={false}
              tickFormatter={(value) => `$${(value / 1000).toFixed(1)}k`}
              domain={["dataMin - 500", "dataMax + 500"]}
            />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area
              type="monotone"
              dataKey="price"
              stroke="hsl(var(--primary))"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#tradingColorPrice)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartContainer>
    </Card>
  )
}
