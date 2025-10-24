"use client"

import { TrendingUp, TrendingDown, DollarSign, Activity } from "lucide-react"
import { Card } from "@/components/ui/card"

const stats = [
  {
    name: "Portfolio Value",
    value: "$24,563.89",
    change: "+12.5%",
    trend: "up",
    icon: DollarSign,
  },
  {
    name: "24h Change",
    value: "+$1,234.56",
    change: "+5.3%",
    trend: "up",
    icon: TrendingUp,
  },
  {
    name: "Total Profit",
    value: "$8,945.23",
    change: "+23.1%",
    trend: "up",
    icon: Activity,
  },
  {
    name: "Available Balance",
    value: "$10,000.00",
    change: "0%",
    trend: "neutral",
    icon: DollarSign,
  },
]

export function QuickStats() {
  return (
    <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {stats.map((stat) => (
        <Card key={stat.name} className="p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
              <stat.icon className="h-5 w-5 text-primary" />
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
            <p className="text-2xl font-bold">{stat.value}</p>
          </div>
        </Card>
      ))}
    </div>
  )
}
