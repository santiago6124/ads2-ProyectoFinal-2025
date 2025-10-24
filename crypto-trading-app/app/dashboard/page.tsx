"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { PortfolioOverview } from "@/components/portfolio-overview"
import { QuickStats } from "@/components/quick-stats"
import { TrendingCoins } from "@/components/trending-coins"
import { RecentActivity } from "@/components/recent-activity"
import { CryptoPricesGrid } from "@/components/crypto-prices"

export default function DashboardPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  if (isLoading || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-blue-50">
        <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            Welcome back, {user?.first_name && user?.last_name 
              ? `${user.first_name} ${user.last_name}`
              : user?.username || 'User'
            }
          </h1>
          <p className="text-muted-foreground mt-1">Here's what's happening with your portfolio today.</p>
        </div>

        <QuickStats />

        <div>
          <h2 className="text-2xl font-bold mb-4">Market Prices</h2>
          <CryptoPricesGrid />
        </div>

        <div className="grid lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <PortfolioOverview />
            <RecentActivity />
          </div>
          <div>
            <TrendingCoins />
          </div>
        </div>
      </div>
    </DashboardLayout>
  )
}
