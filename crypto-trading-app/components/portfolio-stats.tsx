"use client"

import { Card } from "@/components/ui/card"
import { TrendingUp, TrendingDown, Wallet, DollarSign, Activity } from "lucide-react"

const stats = [
  {
    name: "Total Balance",
    value: "$34,563.89",
    change: "+12.5%",
    trend: "up",
    icon: Wallet,
    description: "Total portfolio value",
  },
  {
    name: "Total Profit/Loss",
    value: "+$8,945.23",
    change: "+34.9%",
    trend: "up",
    icon: TrendingUp,
    description: "All-time P&L",
  },
  {
    name: "24h Change",
    value: "+$1,234.56",
    change: "+3.7%",
    trend: "up",
    icon: Activity,
    description: "Daily performance",
  },
  {
    name: "Available Cash",
    value: "$10,000.00",
    change: "Ready to trade",
    trend: "neutral",
    icon: DollarSign,
    description: "USD balance",
  },
]

export function PortfolioStats() {
  return (
    <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {stats.map((stat) => (
        <Card key={stat.name} className="p-6">
          <div className="flex items-start justify-between mb-4">
            <div className="h-12 w-12 rounded-xl bg-primary/10 flex items-center justify-center">
              <stat.icon className="h-6 w-6 text-primary" />
            </div>
            {stat.trend !== "neutral" && (
              <div
                className={`flex items-center gap-1 text-sm font-medium ${
                  stat.trend === "up" ? "text-green-500" : "text-red-500"
                }`}
              >
                {stat.trend === "up" ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
                {stat.change}
              </div>
            )}
          </div>
          <div>
            <p className="text-sm text-muted-foreground mb-1">{stat.name}</p>
            <p className="text-2xl font-bold mb-1">{stat.value}</p>
            <p className="text-xs text-muted-foreground">{stat.description}</p>
          </div>
        </Card>
      ))}
    </div>
  )
}
