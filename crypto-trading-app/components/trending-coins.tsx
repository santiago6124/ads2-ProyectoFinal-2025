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
    <Card className="p-6 bg-linear-to-br from-slate-900 to-slate-800 border-slate-700 shadow-lg">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-white">Trending</h2>
        <p className="text-sm text-slate-400 mt-1">Top movers in 24h</p>
      </div>

      <div className="space-y-4">
        {trendingCoins.map((coin) => (
          <div
            key={coin.symbol}
            className="flex items-center justify-between p-4 rounded-xl bg-slate-800/50 hover:bg-slate-700/50 transition-all duration-300 cursor-pointer border border-slate-700 hover:border-slate-600"
          >
            <div className="flex items-center gap-4">
              <div className="h-12 w-12 rounded-xl bg-linear-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-lg">
                <span className="text-sm font-bold text-white">{coin.symbol}</span>
              </div>
              <div>
                <p className="font-bold text-sm text-white">{coin.name}</p>
                <p className="text-xs text-slate-400">{coin.symbol}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-bold text-sm text-white">{coin.price}</p>
              <div
                className={`flex items-center justify-end gap-1 text-xs font-bold px-2 py-1 rounded-full ${
                  coin.trend === "up" ? "bg-green-500/20 text-green-400" : "bg-red-500/20 text-red-400"
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
