"use client"

import { Card } from "@/components/ui/card"
import { TrendingUp, TrendingDown } from "lucide-react"

const trendingCoins = [
  { symbol: "BTC", name: "Bitcoin", price: "$44,823.45", change: "+5.2%", trend: "up" },
  { symbol: "ETH", name: "Ethereum", price: "$2,563.12", change: "+3.8%", trend: "up" },
  { symbol: "BNB", name: "Binance Coin", price: "$312.45", change: "+2.1%", trend: "up" },
  { symbol: "SOL", name: "Solana", price: "$124.32", change: "-1.5%", trend: "down" },
  { symbol: "XRP", name: "Ripple", price: "$0.6234", change: "+4.3%", trend: "up" },
  { symbol: "ADA", name: "Cardano", price: "$1.89", change: "+1.2%", trend: "up" },
]

export function TrendingCoins() {
  return (
    <Card className="p-6">
      <div className="mb-6">
        <h2 className="text-xl font-bold">Trending</h2>
        <p className="text-sm text-muted-foreground mt-1">Top movers in 24h</p>
      </div>

      <div className="space-y-3">
        {trendingCoins.map((coin) => (
          <div
            key={coin.symbol}
            className="flex items-center justify-between p-3 rounded-lg hover:bg-accent/50 transition-colors cursor-pointer"
          >
            <div className="flex items-center gap-3">
              <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                <span className="text-xs font-bold text-primary">{coin.symbol}</span>
              </div>
              <div>
                <p className="font-semibold text-sm">{coin.name}</p>
                <p className="text-xs text-muted-foreground">{coin.symbol}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-semibold text-sm">{coin.price}</p>
              <div
                className={`flex items-center justify-end gap-1 text-xs font-medium ${
                  coin.trend === "up" ? "text-green-500" : "text-red-500"
                }`}
              >
                {coin.trend === "up" ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
                {coin.change}
              </div>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
