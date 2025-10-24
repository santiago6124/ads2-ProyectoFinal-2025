"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { TrendingUp, TrendingDown, Star } from "lucide-react"
import Link from "next/link"

interface Crypto {
  id: string
  symbol: string
  name: string
  price: string
  change24h: number
  volume24h: string
  marketCap: string
  sparkline: number[]
}

const cryptoData: Crypto[] = [
  {
    id: "bitcoin",
    symbol: "BTC",
    name: "Bitcoin",
    price: "$44,823.45",
    change24h: 5.2,
    volume24h: "$28.5B",
    marketCap: "$876.2B",
    sparkline: [42000, 42500, 43000, 42800, 43500, 44000, 44823],
  },
  {
    id: "ethereum",
    symbol: "ETH",
    name: "Ethereum",
    price: "$2,563.12",
    change24h: 3.8,
    volume24h: "$15.2B",
    marketCap: "$308.1B",
    sparkline: [2400, 2450, 2500, 2480, 2520, 2550, 2563],
  },
  {
    id: "binancecoin",
    symbol: "BNB",
    name: "Binance Coin",
    price: "$312.45",
    change24h: 2.1,
    volume24h: "$1.8B",
    marketCap: "$48.2B",
    sparkline: [305, 308, 310, 309, 311, 312, 312],
  },
  {
    id: "solana",
    symbol: "SOL",
    name: "Solana",
    price: "$124.32",
    change24h: -1.5,
    volume24h: "$2.4B",
    marketCap: "$52.8B",
    sparkline: [128, 127, 126, 125, 124, 123, 124],
  },
  {
    id: "ripple",
    symbol: "XRP",
    name: "Ripple",
    price: "$0.6234",
    change24h: 4.3,
    volume24h: "$1.2B",
    marketCap: "$33.5B",
    sparkline: [0.59, 0.6, 0.61, 0.62, 0.62, 0.623, 0.623],
  },
  {
    id: "cardano",
    symbol: "ADA",
    name: "Cardano",
    price: "$1.89",
    change24h: 1.2,
    volume24h: "$890M",
    marketCap: "$66.4B",
    sparkline: [1.85, 1.86, 1.87, 1.88, 1.88, 1.89, 1.89],
  },
  {
    id: "dogecoin",
    symbol: "DOGE",
    name: "Dogecoin",
    price: "$0.0823",
    change24h: -2.3,
    volume24h: "$645M",
    marketCap: "$11.7B",
    sparkline: [0.085, 0.084, 0.083, 0.082, 0.082, 0.082, 0.082],
  },
  {
    id: "polkadot",
    symbol: "DOT",
    name: "Polkadot",
    price: "$7.45",
    change24h: 6.8,
    volume24h: "$456M",
    marketCap: "$9.8B",
    sparkline: [6.9, 7.0, 7.1, 7.2, 7.3, 7.4, 7.45],
  },
]

interface MarketTableProps {
  searchQuery: string
}

export function MarketTable({ searchQuery }: MarketTableProps) {
  const [favorites, setFavorites] = useState<Set<string>>(new Set())

  const filteredData = cryptoData.filter(
    (crypto) =>
      crypto.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      crypto.symbol.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  const toggleFavorite = (id: string) => {
    setFavorites((prev) => {
      const newFavorites = new Set(prev)
      if (newFavorites.has(id)) {
        newFavorites.delete(id)
      } else {
        newFavorites.add(id)
      }
      return newFavorites
    })
  }

  return (
    <Card className="overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="border-b border-border bg-muted/50">
            <tr>
              <th className="text-left p-4 text-sm font-semibold text-muted-foreground">#</th>
              <th className="text-left p-4 text-sm font-semibold text-muted-foreground">Name</th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground">Price</th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground">24h %</th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground hidden md:table-cell">
                Volume (24h)
              </th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground hidden lg:table-cell">
                Market Cap
              </th>
              <th className="text-center p-4 text-sm font-semibold text-muted-foreground hidden xl:table-cell">
                Last 7 Days
              </th>
              <th className="text-right p-4 text-sm font-semibold text-muted-foreground">Action</th>
            </tr>
          </thead>
          <tbody>
            {filteredData.map((crypto, index) => (
              <tr key={crypto.id} className="border-b border-border hover:bg-accent/50 transition-colors">
                <td className="p-4">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => toggleFavorite(crypto.id)}
                      className="text-muted-foreground hover:text-yellow-500 transition-colors"
                    >
                      <Star
                        className={`h-4 w-4 ${favorites.has(crypto.id) ? "fill-yellow-500 text-yellow-500" : ""}`}
                      />
                    </button>
                    <span className="text-sm text-muted-foreground">{index + 1}</span>
                  </div>
                </td>
                <td className="p-4">
                  <div className="flex items-center gap-3">
                    <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                      <span className="text-xs font-bold text-primary">{crypto.symbol}</span>
                    </div>
                    <div>
                      <p className="font-semibold">{crypto.name}</p>
                      <p className="text-sm text-muted-foreground">{crypto.symbol}</p>
                    </div>
                  </div>
                </td>
                <td className="p-4 text-right font-semibold">{crypto.price}</td>
                <td className="p-4 text-right">
                  <div
                    className={`inline-flex items-center gap-1 font-semibold ${
                      crypto.change24h >= 0 ? "text-green-500" : "text-red-500"
                    }`}
                  >
                    {crypto.change24h >= 0 ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
                    {Math.abs(crypto.change24h).toFixed(2)}%
                  </div>
                </td>
                <td className="p-4 text-right text-muted-foreground hidden md:table-cell">{crypto.volume24h}</td>
                <td className="p-4 text-right text-muted-foreground hidden lg:table-cell">{crypto.marketCap}</td>
                <td className="p-4 hidden xl:table-cell">
                  <div className="flex items-center justify-center">
                    <svg width="100" height="40" className="overflow-visible">
                      <polyline
                        points={crypto.sparkline
                          .map((value, i) => {
                            const x = (i / (crypto.sparkline.length - 1)) * 100
                            const min = Math.min(...crypto.sparkline)
                            const max = Math.max(...crypto.sparkline)
                            const y = 40 - ((value - min) / (max - min)) * 40
                            return `${x},${y}`
                          })
                          .join(" ")}
                        fill="none"
                        stroke={crypto.change24h >= 0 ? "rgb(34, 197, 94)" : "rgb(239, 68, 68)"}
                        strokeWidth="2"
                      />
                    </svg>
                  </div>
                </td>
                <td className="p-4 text-right">
                  <Button size="sm" asChild>
                    <Link href={`/trade?coin=${crypto.id}`}>Trade</Link>
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}
