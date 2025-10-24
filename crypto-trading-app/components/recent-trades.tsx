"use client"

import { Card } from "@/components/ui/card"

const generateTrades = (count: number) => {
  return Array.from({ length: count }, (_, i) => ({
    price: 44800 + Math.random() * 100,
    amount: (Math.random() * 0.5).toFixed(4),
    time: new Date(Date.now() - i * 60000).toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    }),
    type: Math.random() > 0.5 ? "buy" : "sell",
  }))
}

export function RecentTrades() {
  const trades = generateTrades(15)

  return (
    <Card className="p-6">
      <h3 className="text-lg font-bold mb-4">Recent Trades</h3>

      <div className="space-y-2">
        <div className="grid grid-cols-3 gap-4 text-xs text-muted-foreground px-2">
          <span>Price (USD)</span>
          <span className="text-right">Amount (BTC)</span>
          <span className="text-right">Time</span>
        </div>

        <div className="space-y-1 max-h-[300px] overflow-y-auto">
          {trades.map((trade, i) => (
            <div
              key={i}
              className="grid grid-cols-3 gap-4 text-sm px-2 py-2 rounded hover:bg-accent/50 transition-colors"
            >
              <span className={`font-medium ${trade.type === "buy" ? "text-green-500" : "text-red-500"}`}>
                ${trade.price.toFixed(2)}
              </span>
              <span className="text-right">{trade.amount}</span>
              <span className="text-right text-muted-foreground">{trade.time}</span>
            </div>
          ))}
        </div>
      </div>
    </Card>
  )
}
