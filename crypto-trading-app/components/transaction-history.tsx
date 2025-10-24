"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ArrowUpRight, ArrowDownRight, Search, Download } from "lucide-react"
import { Badge } from "@/components/ui/badge"

const transactions = [
  {
    id: "1",
    type: "buy",
    coin: "BTC",
    coinName: "Bitcoin",
    amount: "0.0234",
    value: "$1,234.56",
    price: "$52,756.41",
    status: "completed",
    date: "2024-01-15",
    time: "14:32:15",
  },
  {
    id: "2",
    type: "sell",
    coin: "ETH",
    coinName: "Ethereum",
    amount: "0.5",
    value: "$1,281.56",
    price: "$2,563.12",
    status: "completed",
    date: "2024-01-14",
    time: "09:15:42",
  },
  {
    id: "3",
    type: "buy",
    coin: "SOL",
    coinName: "Solana",
    amount: "10",
    value: "$1,243.20",
    price: "$124.32",
    status: "completed",
    date: "2024-01-13",
    time: "16:45:23",
  },
  {
    id: "4",
    type: "buy",
    coin: "ADA",
    coinName: "Cardano",
    amount: "500",
    value: "$945.00",
    price: "$1.89",
    status: "completed",
    date: "2024-01-12",
    time: "11:20:08",
  },
  {
    id: "5",
    type: "sell",
    coin: "BTC",
    coinName: "Bitcoin",
    amount: "0.0156",
    value: "$823.45",
    price: "$52,785.90",
    status: "completed",
    date: "2024-01-11",
    time: "13:55:31",
  },
  {
    id: "6",
    type: "buy",
    coin: "ETH",
    coinName: "Ethereum",
    amount: "1.2",
    value: "$3,075.74",
    price: "$2,563.12",
    status: "pending",
    date: "2024-01-10",
    time: "08:12:45",
  },
]

export function TransactionHistory() {
  const [searchQuery, setSearchQuery] = useState("")
  const [filter, setFilter] = useState<"all" | "buy" | "sell">("all")

  const filteredTransactions = transactions.filter((tx) => {
    const matchesSearch =
      tx.coin.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tx.coinName.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesFilter = filter === "all" || tx.type === filter
    return matchesSearch && matchesFilter
  })

  return (
    <Card className="p-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h2 className="text-xl font-bold">Transaction History</h2>
          <p className="text-sm text-muted-foreground mt-1">View all your trading activity</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search transactions..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10 w-full sm:w-64"
            />
          </div>
          <Button variant="outline" size="icon">
            <Download className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="flex gap-2 mb-6">
        <Button
          variant={filter === "all" ? "default" : "outline"}
          size="sm"
          onClick={() => setFilter("all")}
          className="bg-transparent"
        >
          All
        </Button>
        <Button
          variant={filter === "buy" ? "default" : "outline"}
          size="sm"
          onClick={() => setFilter("buy")}
          className="bg-transparent"
        >
          Buy
        </Button>
        <Button
          variant={filter === "sell" ? "default" : "outline"}
          size="sm"
          onClick={() => setFilter("sell")}
          className="bg-transparent"
        >
          Sell
        </Button>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="border-b border-border">
            <tr>
              <th className="text-left p-4 text-sm font-semibold text-muted-foreground">Type</th>
              <th className="text-left p-4 text-sm font-semibold text-muted-foreground">Asset</th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground">Amount</th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground hidden md:table-cell">Price</th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground">Value</th>
              <th className="text-center p-4 text-sm font-semibold text-muted-foreground hidden lg:table-cell">
                Status
              </th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground hidden xl:table-cell">
                Date & Time
              </th>
            </tr>
          </thead>
          <tbody>
            {filteredTransactions.map((tx) => (
              <tr key={tx.id} className="border-b border-border hover:bg-accent/50 transition-colors">
                <td className="p-4">
                  <div
                    className={`inline-flex items-center gap-2 px-3 py-1 rounded-full ${
                      tx.type === "buy" ? "bg-green-500/10 text-green-500" : "bg-red-500/10 text-red-500"
                    }`}
                  >
                    {tx.type === "buy" ? <ArrowDownRight className="h-4 w-4" /> : <ArrowUpRight className="h-4 w-4" />}
                    <span className="text-sm font-semibold capitalize">{tx.type}</span>
                  </div>
                </td>
                <td className="p-4">
                  <div className="flex items-center gap-3">
                    <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                      <span className="text-xs font-bold text-primary">{tx.coin}</span>
                    </div>
                    <div>
                      <p className="font-semibold">{tx.coinName}</p>
                      <p className="text-xs text-muted-foreground">{tx.coin}</p>
                    </div>
                  </div>
                </td>
                <td className="p-4 text-right font-medium">
                  {tx.amount} {tx.coin}
                </td>
                <td className="p-4 text-right text-muted-foreground hidden md:table-cell">{tx.price}</td>
                <td className="p-4 text-right font-semibold">{tx.value}</td>
                <td className="p-4 text-center hidden lg:table-cell">
                  <Badge variant={tx.status === "completed" ? "default" : "secondary"} className="capitalize">
                    {tx.status}
                  </Badge>
                </td>
                <td className="p-4 text-right text-sm text-muted-foreground hidden xl:table-cell">
                  <div>{tx.date}</div>
                  <div className="text-xs">{tx.time}</div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {filteredTransactions.length === 0 && (
        <div className="text-center py-12">
          <p className="text-muted-foreground">No transactions found</p>
        </div>
      )}
    </Card>
  )
}
