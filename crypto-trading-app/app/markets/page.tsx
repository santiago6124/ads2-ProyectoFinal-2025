"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { MarketTable } from "@/components/market-table"
import { MarketChart } from "@/components/market-chart"
import { ExtendedMarketStats } from "@/components/extended-market-stats"
import { MarketCategories } from "@/components/market-categories"
import { TrendingCoins } from "@/components/trending-coins-market"
import { WinnersLosers } from "@/components/winners-losers"
import { Input } from "@/components/ui/input"
import { Search, RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"

export default function MarketsPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const [searchQuery, setSearchQuery] = useState("")
  const [activeTab, setActiveTab] = useState("overview")
  const [lastUpdated, setLastUpdated] = useState<Date>(new Date())

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  useEffect(() => {
    // Update last updated time every 30 seconds
    const interval = setInterval(() => {
      setLastUpdated(new Date())
    }, 30000)
    return () => clearInterval(interval)
  }, [])

  if (isLoading || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-black">
        <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full" />
      </div>
    )
  }

  const tabs = [
    { id: "overview", name: "Overview" },
    { id: "trending", name: "Trending" },
    { id: "categories", name: "Categories" },
    { id: "winners-losers", name: "Gainers & Losers" },
    { id: "all-coins", name: "All Coins" }
  ]

  return (
    <DashboardLayout>
      <div className="space-y-8 bg-black min-h-screen p-6">
        {/* Header */}
        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <h1 className="text-4xl font-bold tracking-tight text-white">Markets</h1>
            <p className="text-white/60 mt-2 text-lg">
              Real-time cryptocurrency prices and market analysis
            </p>
            <p className="text-white/40 text-sm mt-1">
              Last updated: {lastUpdated.toLocaleTimeString()}
            </p>
          </div>
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="relative w-full sm:w-80">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-white/60" />
              <Input
                placeholder="Search cryptocurrencies..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10 bg-black border-white/10 text-white placeholder:text-white/60"
              />
            </div>
            <Button 
              variant="outline" 
              className="border-white/20 text-white hover:bg-white/10"
              onClick={() => setLastUpdated(new Date())}
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh
            </Button>
          </div>
        </div>

        {/* Extended Market Stats */}
        <ExtendedMarketStats />

        {/* Tabs */}
        <Card className="p-6 bg-black border border-white/10 shadow-lg">
          <div className="flex flex-wrap gap-2 mb-6">
            {tabs.map((tab) => (
              <Button
                key={tab.id}
                variant={activeTab === tab.id ? "default" : "outline"}
                onClick={() => setActiveTab(tab.id)}
                className={`${
                  activeTab === tab.id 
                    ? "bg-blue-500 text-white border-blue-500" 
                    : "bg-transparent text-white/70 border-white/20 hover:bg-white/10"
                }`}
              >
                {tab.name}
              </Button>
            ))}
          </div>

          {/* Tab Content */}
          {activeTab === "overview" && (
            <div className="space-y-8">
              <MarketChart />
              <div className="grid lg:grid-cols-2 gap-6">
                <TrendingCoins />
                <WinnersLosers />
              </div>
            </div>
          )}

          {activeTab === "trending" && (
            <div className="space-y-6">
              <TrendingCoins />
              <WinnersLosers />
            </div>
          )}

          {activeTab === "categories" && (
            <MarketCategories />
          )}

          {activeTab === "winners-losers" && (
            <WinnersLosers />
          )}

          {activeTab === "all-coins" && (
            <MarketTable searchQuery={searchQuery} />
          )}
        </Card>
      </div>
    </DashboardLayout>
  )
}