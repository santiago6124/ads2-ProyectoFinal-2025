"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { MarketTable } from "@/components/market-table"
import { MarketChart } from "@/components/market-chart"
import { Input } from "@/components/ui/input"
import { Search, TrendingUp, TrendingDown, Activity } from "lucide-react"
import { Card } from "@/components/ui/card"

export default function MarketsPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const [searchQuery, setSearchQuery] = useState("")

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
            <h1 className="text-3xl font-bold tracking-tight">Markets</h1>
            <p className="text-muted-foreground mt-1">Real-time cryptocurrency prices and charts</p>
          </div>
          <div className="relative w-full sm:w-80">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search cryptocurrencies..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
        </div>

        <div className="grid sm:grid-cols-3 gap-4">
          <Card className="p-6">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm text-muted-foreground">Market Cap</p>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="text-2xl font-bold">$1.2T</p>
            <p className="text-sm text-green-500 mt-1">+2.4% (24h)</p>
          </Card>
          <Card className="p-6">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm text-muted-foreground">24h Volume</p>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="text-2xl font-bold">$89.5B</p>
            <p className="text-sm text-green-500 mt-1">+5.2% (24h)</p>
          </Card>
          <Card className="p-6">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm text-muted-foreground">BTC Dominance</p>
              <TrendingDown className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="text-2xl font-bold">48.3%</p>
            <p className="text-sm text-red-500 mt-1">-0.8% (24h)</p>
          </Card>
        </div>

        <MarketChart />

        <MarketTable searchQuery={searchQuery} />
      </div>
    </DashboardLayout>
  )
}
