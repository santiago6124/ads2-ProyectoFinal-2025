"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { Button } from "@/components/ui/button"
import Link from "next/link"
import { TrendingUp, BarChart3, Shield, Zap, ArrowRight } from "lucide-react"

export default function HomePage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && user) {
      router.push("/dashboard")
    }
  }, [user, isLoading, router])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-blue-50">
        <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50">
      {/* Header */}
      <header className="bg-white/90 backdrop-blur-sm sticky top-0 z-50 border-b border-slate-200">
        <div className="container mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-blue-600 to-blue-700 flex items-center justify-center">
              <TrendingUp className="h-5 w-5 text-white" />
            </div>
            <span className="text-xl font-bold text-slate-900">CryptoTrade</span>
          </div>
          <div className="flex items-center gap-3">
            <Button variant="ghost" asChild className="text-slate-700 hover:text-slate-900">
              <Link href="/login">Sign in</Link>
            </Button>
            <Button asChild className="bg-blue-600 hover:bg-blue-700 text-white">
              <Link href="/signup">Get started</Link>
            </Button>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="container mx-auto px-4 py-20 lg:py-32">
        <div className="max-w-4xl mx-auto text-center space-y-8">
          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-blue-100 border border-blue-200">
            <div className="h-2 w-2 rounded-full bg-blue-600 animate-pulse" />
            <span className="text-sm font-medium text-blue-800">Live Trading Platform</span>
          </div>

          <h1 className="text-5xl lg:text-7xl font-bold tracking-tight text-balance">
            <span className="text-slate-900">Trade crypto with </span>
            <span className="bg-gradient-to-r from-blue-600 to-blue-800 bg-clip-text text-transparent">
              confidence
            </span>
          </h1>

          <p className="text-xl lg:text-2xl text-slate-600 text-balance max-w-2xl mx-auto">
            Access real-time market data, advanced charts, and execute trades instantly on the most trusted platform.
          </p>

          <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-4">
            <Button size="lg" className="h-14 px-8 text-lg font-semibold bg-blue-600 hover:bg-blue-700 text-white" asChild>
              <Link href="/signup">
                Start trading now
                <ArrowRight className="ml-2 h-5 w-5" />
              </Link>
            </Button>
            <Button size="lg" variant="outline" className="h-14 px-8 text-lg border-slate-300 text-slate-700 hover:bg-slate-50" asChild>
              <Link href="/login">Sign in</Link>
            </Button>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="container mx-auto px-4 py-20">
        <div className="grid md:grid-cols-3 gap-8">
          {[
            {
              icon: BarChart3,
              title: "Advanced Charts",
              description: "Professional trading tools with real-time data and technical indicators",
            },
            {
              icon: Zap,
              title: "Instant Execution",
              description: "Lightning-fast trade execution with minimal slippage",
            },
            {
              icon: Shield,
              title: "Secure Trading",
              description: "Bank-level security with encrypted transactions and cold storage",
            },
          ].map((feature, i) => (
            <div key={i} className="p-6 rounded-2xl border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow space-y-4">
              <div className="h-12 w-12 rounded-xl bg-blue-100 flex items-center justify-center">
                <feature.icon className="h-6 w-6 text-blue-600" />
              </div>
              <h3 className="text-2xl font-bold text-slate-900">{feature.title}</h3>
              <p className="text-slate-600 text-lg">{feature.description}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
