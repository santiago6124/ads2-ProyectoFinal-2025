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
    <Card className="p-6 bg-black border border-white/10 shadow-lg">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-white">Recent Activity</h2>
        <p className="text-sm text-white/60 mt-1">Your latest transactions</p>
      </div>

      <div className="space-y-4">
        {activities.map((activity, index) => (
          <div key={index} className="flex items-center justify-between p-4 rounded-xl bg-black border border-white/10 hover:border-white/20 transition-all duration-300">
            <div className="flex items-center gap-4">
              <div
                className={`h-12 w-12 rounded-xl flex items-center justify-center shadow-lg border border-white/10 ${
                  activity.type === "buy" ? "bg-green-500" : "bg-red-500"
                }`}
              >
                {activity.type === "buy" ? (
                  <ArrowDownRight className="h-6 w-6 text-white" />
                ) : (
                  <ArrowUpRight className="h-6 w-6 text-white" />
                )}
              </div>
              <div>
                <p className="font-bold text-white">
                  {activity.type === "buy" ? "Bought" : "Sold"} {activity.coin}
                </p>
                <p className="text-sm text-white/60">{activity.time}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="font-bold text-white">{activity.value}</p>
              <p className="text-sm text-white/60">
                {activity.amount} {activity.coin}
              </p>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
