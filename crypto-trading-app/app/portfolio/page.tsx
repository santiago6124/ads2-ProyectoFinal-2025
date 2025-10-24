"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { PortfolioStats } from "@/components/portfolio-stats"
import { PortfolioChart } from "@/components/portfolio-chart"
import { AssetAllocation } from "@/components/asset-allocation"
import { TransactionHistory } from "@/components/transaction-history"
import { WalletActions } from "@/components/wallet-actions"

export default function PortfolioPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()

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
            <h1 className="text-3xl font-bold tracking-tight">Portfolio</h1>
            <p className="text-muted-foreground mt-1">Track your assets and performance</p>
          </div>
          <WalletActions />
        </div>

        <PortfolioStats />

        <div className="grid lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <PortfolioChart />
          </div>
          <div>
            <AssetAllocation />
          </div>
        </div>

        <TransactionHistory />
      </div>
    </DashboardLayout>
  )
}
