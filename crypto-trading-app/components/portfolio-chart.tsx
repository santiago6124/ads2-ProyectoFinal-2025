"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis, ResponsiveContainer } from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"

const timeframes = ["24H", "7D", "1M", "3M", "1Y", "ALL"]

const generatePortfolioData = (days: number) => {
  const data = []
  const now = Date.now()
  let baseValue = 25000

  for (let i = days; i >= 0; i--) {
    const change = (Math.random() - 0.4) * 1000
    baseValue += change
    data.push({
      date: new Date(now - i * 86400000).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
      }),
      value: Math.max(baseValue, 20000),
    })
  }
  return data
}

export function PortfolioChart() {
  const [selectedTimeframe, setSelectedTimeframe] = useState("1M")
  const [chartData] = useState(generatePortfolioData(30))

  const currentValue = chartData[chartData.length - 1]?.value || 34563
  const startValue = chartData[0]?.value || 25000
  const totalChange = currentValue - startValue
  const percentChange = ((totalChange / startValue) * 100).toFixed(2)

  return (
    <Card className="p-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h2 className="text-xl font-bold mb-2">Portfolio Value</h2>
          <div className="flex items-baseline gap-3">
            <span className="text-3xl font-bold">${currentValue.toFixed(2)}</span>
            <span className={`font-semibold ${totalChange >= 0 ? "text-green-500" : "text-red-500"}`}>
              {totalChange >= 0 ? "+" : ""}${totalChange.toFixed(2)} ({percentChange}%)
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
          value: {
            label: "Portfolio Value",
            color: "hsl(var(--primary))",
          },
        }}
        className="h-[400px] w-full"
      >
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData}>
            <defs>
              <linearGradient id="portfolioGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3} />
                <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.3} />
            <XAxis
              dataKey="date"
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
              tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`}
            />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area
              type="monotone"
              dataKey="value"
              stroke="hsl(var(--primary))"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#portfolioGradient)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartContainer>
    </Card>
  )
}
