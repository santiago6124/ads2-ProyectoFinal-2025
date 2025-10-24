"use client"

import { Card } from "@/components/ui/card"

const generateOrders = (count: number, isBuy: boolean) => {
  return Array.from({ length: count }, (_, i) => ({
    price: isBuy ? 44800 - i * 10 : 44850 + i * 10,
    amount: (Math.random() * 2).toFixed(4),
    total: ((44800 - i * 10) * Math.random() * 2).toFixed(2),
  }))
}

export function OrderBook() {
  const buyOrders = generateOrders(8, true)
  const sellOrders = generateOrders(8, false)

  return (
    <Card className="p-6">
      <h3 className="text-lg font-bold mb-4">Order Book</h3>

      <div className="space-y-4">
        <div>
          <div className="grid grid-cols-3 gap-2 text-xs text-muted-foreground mb-2 px-2">
            <span>Price (USD)</span>
            <span className="text-right">Amount (BTC)</span>
            <span className="text-right">Total</span>
          </div>

          <div className="space-y-1">
            {sellOrders.reverse().map((order, i) => (
              <div
                key={`sell-${i}`}
                className="grid grid-cols-3 gap-2 text-sm px-2 py-1 rounded hover:bg-accent/50 transition-colors"
              >
                <span className="text-red-500 font-medium">${order.price.toLocaleString()}</span>
                <span className="text-right">{order.amount}</span>
                <span className="text-right text-muted-foreground">${order.total}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="py-3 px-2 bg-accent/50 rounded-lg">
          <div className="text-center">
            <div className="text-2xl font-bold">$44,823.45</div>
            <div className="text-xs text-muted-foreground">Current Price</div>
          </div>
        </div>

        <div>
          <div className="space-y-1">
            {buyOrders.map((order, i) => (
              <div
                key={`buy-${i}`}
                className="grid grid-cols-3 gap-2 text-sm px-2 py-1 rounded hover:bg-accent/50 transition-colors"
              >
                <span className="text-green-500 font-medium">${order.price.toLocaleString()}</span>
                <span className="text-right">{order.amount}</span>
                <span className="text-right text-muted-foreground">${order.total}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Card>
  )
}
