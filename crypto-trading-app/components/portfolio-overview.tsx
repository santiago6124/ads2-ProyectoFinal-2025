"use client"

import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { ArrowUpRight, ArrowDownRight } from "lucide-react"

const holdings = [
  { symbol: "BTC", name: "Bitcoin", amount: "0.5234", value: "$23,456.78", change: "+5.2%", trend: "up" },
  { symbol: "ETH", name: "Ethereum", amount: "3.2145", value: "$8,234.56", change: "+3.8%", trend: "up" },
  { symbol: "SOL", name: "Solana", amount: "45.678", value: "$5,678.90", change: "-2.1%", trend: "down" },
  { symbol: "ADA", name: "Cardano", amount: "1234.56", value: "$2,345.67", change: "+1.5%", trend: "up" },
]

export function PortfolioOverview() {
  return (
    <Card className="p-6 bg-black border border-white/10 shadow-lg">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-white">Your Holdings</h2>
          <p className="text-sm text-white/60 mt-1">Current cryptocurrency positions</p>
        </div>
        <Button variant="outline" size="sm" className="border-white/20 text-white hover:bg-white/10">
          View All
        </Button>
      </div>

      <div className="space-y-4">
        {holdings.map((holding) => (
          <div
            key={holding.symbol}
            className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300"
          >
            <div className="flex items-center gap-4">
              <div className="h-14 w-14 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg border border-white/10">
                <span className="text-sm font-bold text-white">{holding.symbol}</span>
              </div>
              <div>
                <p className="font-bold text-white">{holding.name}</p>
                <p className="text-sm text-white/60">
                  {holding.amount} {holding.symbol}
                </p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-bold text-white">{holding.value}</p>
              <div
                className={`flex items-center justify-end gap-1 text-sm font-bold px-2 py-1 rounded-full border ${
                  holding.trend === "up" ? "bg-green-500/20 text-green-400 border-green-500/30" : "bg-red-500/20 text-red-400 border-red-500/30"
                }`}
              >
                {holding.trend === "up" ? <ArrowUpRight className="h-4 w-4" /> : <ArrowDownRight className="h-4 w-4" />}
                {holding.change}
              </div>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
