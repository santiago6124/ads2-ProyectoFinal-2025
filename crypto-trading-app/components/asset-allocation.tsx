"use client"

import { Card } from "@/components/ui/card"
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts"

const assets = [
  { name: "Bitcoin", value: 45, amount: "$15,653.75", color: "hsl(var(--chart-1))" },
  { name: "Ethereum", value: 25, amount: "$8,640.97", color: "hsl(var(--chart-2))" },
  { name: "Solana", value: 15, amount: "$5,184.58", color: "hsl(var(--chart-3))" },
  { name: "Others", value: 10, amount: "$3,456.39", color: "hsl(var(--chart-4))" },
  { name: "Cash", value: 5, amount: "$1,628.20", color: "hsl(var(--chart-5))" },
]

export function AssetAllocation() {
  return (
    <Card className="p-6">
      <h2 className="text-xl font-bold mb-6">Asset Allocation</h2>

      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie data={assets} cx="50%" cy="50%" innerRadius={60} outerRadius={90} paddingAngle={2} dataKey="value">
            {assets.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip
            content={({ active, payload }) => {
              if (active && payload && payload.length) {
                return (
                  <div className="rounded-lg border bg-background p-3 shadow-sm">
                    <div className="font-semibold">{payload[0].name}</div>
                    <div className="text-sm text-muted-foreground">
                      {payload[0].value}% - {assets.find((a) => a.name === payload[0].name)?.amount}
                    </div>
                  </div>
                )
              }
              return null
            }}
          />
        </PieChart>
      </ResponsiveContainer>

      <div className="space-y-3 mt-6">
        {assets.map((asset) => (
          <div key={asset.name} className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="h-3 w-3 rounded-full" style={{ backgroundColor: asset.color }} />
              <span className="text-sm font-medium">{asset.name}</span>
            </div>
            <div className="text-right">
              <p className="text-sm font-semibold">{asset.amount}</p>
              <p className="text-xs text-muted-foreground">{asset.value}%</p>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
