"use client"

import { TrendingUp, TrendingDown, DollarSign, Activity } from "lucide-react"
import { Card } from "@/components/ui/card"
import { useAuth } from "@/lib/auth-context"

export function QuickStats() {
  const { user } = useAuth()
  
  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(amount)
  }

  const stats = [
    {
      name: "Portfolio Value",
      value: formatCurrency(user?.initial_balance || 0),
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
      value: formatCurrency(user?.initial_balance || 0),
      change: "0%",
      trend: "neutral",
      icon: DollarSign,
    },
  ]

  return (
    <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
      {stats.map((stat, index) => (
        <Card key={stat.name} className="p-6 bg-black border border-white/10 shadow-lg hover:shadow-xl transition-all duration-300 hover:scale-105">
          <div className="flex items-center justify-between mb-4">
            <div className="h-12 w-12 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg border border-white/10">
              <stat.icon className="h-6 w-6 text-white" />
            </div>
            {stat.trend !== "neutral" && (
              <div
                className={`flex items-center gap-1 text-sm font-bold px-2 py-1 rounded-full border ${
                  stat.trend === "up" ? "bg-green-500/20 text-green-400 border-green-500/30" : "bg-red-500/20 text-red-400 border-red-500/30"
                }`}
              >
                {stat.trend === "up" ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
                {stat.change}
              </div>
            )}
          </div>
          <div>
            <p className="text-sm text-white/60 mb-2 font-medium">{stat.name}</p>
            <p className="text-2xl font-bold text-white">{stat.value}</p>
          </div>
        </Card>
      ))}
    </div>
  )
}
