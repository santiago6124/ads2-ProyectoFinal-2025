"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Input } from "@/components/ui/input"
import { Check, ChevronsUpDown, Search } from "lucide-react"
import { cn } from "@/lib/utils"

const coins = [
  { id: "bitcoin", symbol: "BTC", name: "Bitcoin", price: "$44,823.45" },
  { id: "ethereum", symbol: "ETH", name: "Ethereum", price: "$2,563.12" },
  { id: "binancecoin", symbol: "BNB", name: "Binance Coin", price: "$312.45" },
  { id: "solana", symbol: "SOL", name: "Solana", price: "$124.32" },
  { id: "ripple", symbol: "XRP", name: "Ripple", price: "$0.6234" },
  { id: "cardano", symbol: "ADA", name: "Cardano", price: "$1.89" },
]

interface CoinSelectorProps {
  selectedCoin: string
  onSelectCoin: (coin: string) => void
}

export function CoinSelector({ selectedCoin, onSelectCoin }: CoinSelectorProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState("")

  const selected = coins.find((coin) => coin.id === selectedCoin)
  const filteredCoins = coins.filter(
    (coin) =>
      coin.name.toLowerCase().includes(search.toLowerCase()) ||
      coin.symbol.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full sm:w-[280px] justify-between bg-transparent"
        >
          <div className="flex items-center gap-3">
            <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
              <span className="text-xs font-bold text-primary">{selected?.symbol}</span>
            </div>
            <div className="text-left">
              <p className="font-semibold">{selected?.name}</p>
              <p className="text-xs text-muted-foreground">{selected?.price}</p>
            </div>
          </div>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[280px] p-0">
        <div className="p-2 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search coin..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8 h-9"
            />
          </div>
        </div>
        <div className="max-h-[300px] overflow-auto p-1">
          {filteredCoins.map((coin) => (
            <button
              key={coin.id}
              onClick={() => {
                onSelectCoin(coin.id)
                setOpen(false)
                setSearch("")
              }}
              className={cn(
                "w-full flex items-center gap-3 p-2 rounded-md hover:bg-accent transition-colors",
                selectedCoin === coin.id && "bg-accent",
              )}
            >
              <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                <span className="text-xs font-bold text-primary">{coin.symbol}</span>
              </div>
              <div className="flex-1 text-left">
                <p className="font-semibold text-sm">{coin.name}</p>
                <p className="text-xs text-muted-foreground">{coin.symbol}</p>
              </div>
              <div className="text-right">
                <p className="text-sm font-medium">{coin.price}</p>
              </div>
              {selectedCoin === coin.id && <Check className="h-4 w-4 text-primary" />}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
