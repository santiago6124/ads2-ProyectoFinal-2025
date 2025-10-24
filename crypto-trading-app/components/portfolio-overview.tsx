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
    <Card className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-bold">Your Holdings</h2>
          <p className="text-sm text-muted-foreground mt-1">Current cryptocurrency positions</p>
        </div>
        <Button variant="outline" size="sm">
          View All
        </Button>
      </div>

      <div className="space-y-4">
        {holdings.map((holding) => (
          <div
            key={holding.symbol}
            className="flex items-center justify-between p-4 rounded-lg border border-border hover:bg-accent/50 transition-colors"
          >
            <div className="flex items-center gap-4">
              <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center">
                <span className="text-sm font-bold text-primary">{holding.symbol}</span>
              </div>
              <div>
                <p className="font-semibold">{holding.name}</p>
                <p className="text-sm text-muted-foreground">
                  {holding.amount} {holding.symbol}
                </p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-semibold">{holding.value}</p>
              <div
                className={`flex items-center justify-end gap-1 text-sm font-medium ${
                  holding.trend === "up" ? "text-green-500" : "text-red-500"
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
