"use client"

import { Card } from "@/components/ui/card"
import { ArrowUpRight, ArrowDownRight } from "lucide-react"

const activities = [
  {
    type: "buy",
    coin: "BTC",
    amount: "0.0234",
    value: "$1,234.56",
    time: "2 hours ago",
  },
  {
    type: "sell",
    coin: "ETH",
    amount: "0.5",
    value: "$1,281.56",
    time: "5 hours ago",
  },
  {
    type: "buy",
    coin: "SOL",
    amount: "10",
    value: "$1,243.20",
    time: "1 day ago",
  },
  {
    type: "buy",
    coin: "ADA",
    amount: "500",
    value: "$945.00",
    time: "2 days ago",
  },
]

export function RecentActivity() {
  return (
    <Card className="p-6">
      <div className="mb-6">
        <h2 className="text-xl font-bold">Recent Activity</h2>
        <p className="text-sm text-muted-foreground mt-1">Your latest transactions</p>
      </div>

      <div className="space-y-4">
        {activities.map((activity, index) => (
          <div key={index} className="flex items-center justify-between p-4 rounded-lg border border-border">
            <div className="flex items-center gap-4">
              <div
                className={`h-10 w-10 rounded-full flex items-center justify-center ${
                  activity.type === "buy" ? "bg-green-500/10" : "bg-red-500/10"
                }`}
              >
                {activity.type === "buy" ? (
                  <ArrowDownRight className="h-5 w-5 text-green-500" />
                ) : (
                  <ArrowUpRight className="h-5 w-5 text-red-500" />
                )}
              </div>
              <div>
                <p className="font-semibold">
                  {activity.type === "buy" ? "Bought" : "Sold"} {activity.coin}
                </p>
                <p className="text-sm text-muted-foreground">{activity.time}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-semibold">{activity.value}</p>
              <p className="text-sm text-muted-foreground">
                {activity.amount} {activity.coin}
              </p>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
