"use client"

import { useEffect, useState, Suspense } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { TradingChart } from "@/components/trading-chart"
import { OrderBook } from "@/components/order-book"
import { TradeForm } from "@/components/trade-form"
import { RecentTrades } from "@/components/recent-trades"
import { CoinSelector } from "@/components/coin-selector"

function TradeContent() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const searchParams = useSearchParams()
  const [selectedCoin, setSelectedCoin] = useState(searchParams.get("coin") || "bitcoin")

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  if (isLoading || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Trade</h1>
            <p className="text-muted-foreground mt-1">Buy and sell cryptocurrencies instantly</p>
          </div>
          <CoinSelector selectedCoin={selectedCoin} onSelectCoin={setSelectedCoin} />
        </div>

        <div className="grid lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <TradingChart coin={selectedCoin} />
            <RecentTrades />
          </div>
          <div className="space-y-6">
            <TradeForm coin={selectedCoin} />
            <OrderBook />
          </div>
        </div>
      </div>
    </DashboardLayout>
  )
}

export default function TradePage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center">
          <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
        </div>
      }
    >
      <TradeContent />
    </Suspense>
  )
}
