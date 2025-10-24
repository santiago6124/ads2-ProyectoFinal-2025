"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAuth } from "@/lib/auth-context"
import { ArrowDownUp } from "lucide-react"

interface TradeFormProps {
  coin: string
}

export function TradeForm({ coin }: TradeFormProps) {
  const { user } = useAuth()
  const [buyAmount, setBuyAmount] = useState("")
  const [sellAmount, setSellAmount] = useState("")
  const [buyTotal, setBuyTotal] = useState("")
  const [sellTotal, setSellTotal] = useState("")

  const currentPrice = 44823.45

  const handleBuyAmountChange = (value: string) => {
    setBuyAmount(value)
    const total = Number.parseFloat(value) * currentPrice
    setBuyTotal(isNaN(total) ? "" : total.toFixed(2))
  }

  const handleBuyTotalChange = (value: string) => {
    setBuyTotal(value)
    const amount = Number.parseFloat(value) / currentPrice
    setBuyAmount(isNaN(amount) ? "" : amount.toFixed(8))
  }

  const handleSellAmountChange = (value: string) => {
    setSellAmount(value)
    const total = Number.parseFloat(value) * currentPrice
    setSellTotal(isNaN(total) ? "" : total.toFixed(2))
  }

  const handleBuy = () => {
    if (!buyAmount || Number.parseFloat(buyAmount) <= 0) return
    alert(`Buy order placed: ${buyAmount} BTC for $${buyTotal}`)
    setBuyAmount("")
    setBuyTotal("")
  }

  const handleSell = () => {
    if (!sellAmount || Number.parseFloat(sellAmount) <= 0) return
    alert(`Sell order placed: ${sellAmount} BTC for $${sellTotal}`)
    setSellAmount("")
    setSellTotal("")
  }

  return (
    <Card className="p-6">
      <Tabs defaultValue="buy" className="w-full">
        <TabsList className="grid w-full grid-cols-2 mb-6">
          <TabsTrigger value="buy">Buy</TabsTrigger>
          <TabsTrigger value="sell">Sell</TabsTrigger>
        </TabsList>

        <TabsContent value="buy" className="space-y-4">
          <div className="p-4 rounded-lg bg-accent/50 border border-border">
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-muted-foreground">Available Balance</span>
              <span className="text-sm font-semibold">${user?.balance.toLocaleString()}</span>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="buy-amount">Amount (BTC)</Label>
            <Input
              id="buy-amount"
              type="number"
              placeholder="0.00"
              value={buyAmount}
              onChange={(e) => handleBuyAmountChange(e.target.value)}
              step="0.00000001"
            />
          </div>

          <div className="flex justify-center">
            <div className="h-8 w-8 rounded-full bg-accent flex items-center justify-center">
              <ArrowDownUp className="h-4 w-4 text-muted-foreground" />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="buy-total">Total (USD)</Label>
            <Input
              id="buy-total"
              type="number"
              placeholder="0.00"
              value={buyTotal}
              onChange={(e) => handleBuyTotalChange(e.target.value)}
              step="0.01"
            />
          </div>

          <div className="flex gap-2">
            {[25, 50, 75, 100].map((percent) => (
              <Button
                key={percent}
                variant="outline"
                size="sm"
                className="flex-1 bg-transparent"
                onClick={() => {
                  const total = (user?.balance || 0) * (percent / 100)
                  handleBuyTotalChange(total.toString())
                }}
              >
                {percent}%
              </Button>
            ))}
          </div>

          <div className="p-4 rounded-lg bg-accent/50 border border-border space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Price</span>
              <span className="font-medium">${currentPrice.toLocaleString()}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Fee (0.1%)</span>
              <span className="font-medium">${(Number.parseFloat(buyTotal || "0") * 0.001).toFixed(2)}</span>
            </div>
          </div>

          <Button className="w-full h-12 text-base font-semibold bg-green-600 hover:bg-green-700" onClick={handleBuy}>
            Buy BTC
          </Button>
        </TabsContent>

        <TabsContent value="sell" className="space-y-4">
          <div className="p-4 rounded-lg bg-accent/50 border border-border">
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-muted-foreground">Available BTC</span>
              <span className="text-sm font-semibold">0.5234 BTC</span>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="sell-amount">Amount (BTC)</Label>
            <Input
              id="sell-amount"
              type="number"
              placeholder="0.00"
              value={sellAmount}
              onChange={(e) => handleSellAmountChange(e.target.value)}
              step="0.00000001"
            />
          </div>

          <div className="flex justify-center">
            <div className="h-8 w-8 rounded-full bg-accent flex items-center justify-center">
              <ArrowDownUp className="h-4 w-4 text-muted-foreground" />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="sell-total">Total (USD)</Label>
            <Input id="sell-total" type="number" placeholder="0.00" value={sellTotal} readOnly />
          </div>

          <div className="flex gap-2">
            {[25, 50, 75, 100].map((percent) => (
              <Button
                key={percent}
                variant="outline"
                size="sm"
                className="flex-1 bg-transparent"
                onClick={() => {
                  const amount = 0.5234 * (percent / 100)
                  handleSellAmountChange(amount.toString())
                }}
              >
                {percent}%
              </Button>
            ))}
          </div>

          <div className="p-4 rounded-lg bg-accent/50 border border-border space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Price</span>
              <span className="font-medium">${currentPrice.toLocaleString()}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Fee (0.1%)</span>
              <span className="font-medium">${(Number.parseFloat(sellTotal || "0") * 0.001).toFixed(2)}</span>
            </div>
          </div>

          <Button className="w-full h-12 text-base font-semibold bg-red-600 hover:bg-red-700" onClick={handleSell}>
            Sell BTC
          </Button>
        </TabsContent>
      </Tabs>
    </Card>
  )
}
