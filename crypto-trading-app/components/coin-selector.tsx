"use client"

import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Input } from "@/components/ui/input"
import { Check, ChevronsUpDown, Search, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { marketApiService, PriceData } from "@/lib/market-api"

interface Coin {
  id: string
  symbol: string
  name: string
  price: string
  priceNum: number
}

interface CoinSelectorProps {
  selectedCoin: string
  onSelectCoin: (coin: string) => void
}

export function CoinSelector({ selectedCoin, onSelectCoin }: CoinSelectorProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState("")
  const [coins, setCoins] = useState<Coin[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchCoins = async () => {
      try {
        setLoading(true)
        const prices = await marketApiService.getAllPrices()

        const formattedCoins: Coin[] = prices.map((price: PriceData) => ({
          id: price.symbol.toLowerCase(),
          symbol: price.symbol,
          name: price.name,
          price: formatPrice(price.price),
          priceNum: price.price
        }))

        // Sort by market cap or name
        formattedCoins.sort((a, b) => b.priceNum - a.priceNum)
        setCoins(formattedCoins)
      } catch (error) {
        console.error('Failed to fetch coins:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchCoins()

    // Refresh prices every 60 seconds
    const interval = setInterval(fetchCoins, 60000)
    return () => clearInterval(interval)
  }, [])

  const formatPrice = (price: number) => {
    if (!price || isNaN(price)) return '$0.00'
    if (price >= 1000) {
      return `$${price.toLocaleString()}`
    } else if (price >= 1) {
      return `$${price.toFixed(2)}`
    } else {
      return `$${price.toFixed(6)}`
    }
  }

  const selected = coins.find((coin) => coin.id === selectedCoin || coin.symbol === selectedCoin.toUpperCase())
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
          disabled={loading && coins.length === 0}
        >
          {loading && coins.length === 0 ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>Loading coins...</span>
            </div>
          ) : (
            <>
              <div className="flex items-center gap-3">
                <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                  <span className="text-xs font-bold text-primary">{selected?.symbol || 'BTC'}</span>
                </div>
                <div className="text-left">
                  <p className="font-semibold">{selected?.name || 'Bitcoin'}</p>
                  <p className="text-xs text-muted-foreground">{selected?.price || '$0.00'}</p>
                </div>
              </div>
              <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
            </>
          )}
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
          {loading && coins.length === 0 ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : filteredCoins.length === 0 ? (
            <div className="text-center p-8 text-muted-foreground">
              <p>No coins found</p>
            </div>
          ) : (
            filteredCoins.map((coin) => (
              <button
                key={coin.id}
                onClick={() => {
                  onSelectCoin(coin.symbol)
                  setOpen(false)
                  setSearch("")
                }}
                className={cn(
                  "w-full flex items-center gap-3 p-2 rounded-md hover:bg-accent transition-colors",
                  (selectedCoin === coin.id || selectedCoin.toUpperCase() === coin.symbol) && "bg-accent",
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
                {(selectedCoin === coin.id || selectedCoin.toUpperCase() === coin.symbol) && <Check className="h-4 w-4 text-primary" />}
              </button>
            ))
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
