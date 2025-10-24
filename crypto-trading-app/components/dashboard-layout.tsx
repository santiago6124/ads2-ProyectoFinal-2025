"use client"

import { type ReactNode, useState } from "react"
import { useAuth } from "@/lib/auth-context"
import { useRouter, usePathname } from "next/navigation"
import { Button } from "@/components/ui/button"
import {
  TrendingUp,
  LayoutDashboard,
  LineChart,
  Wallet,
  ArrowLeftRight,
  Settings,
  LogOut,
  Menu,
  X,
  User,
} from "lucide-react"
import Link from "next/link"
import { cn } from "@/lib/utils"

interface DashboardLayoutProps {
  children: ReactNode
}

const navigation = [
  { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { name: "Markets", href: "/markets", icon: LineChart },
  { name: "Trade", href: "/trade", icon: ArrowLeftRight },
  { name: "Portfolio", href: "/portfolio", icon: Wallet },
  { name: "Settings", href: "/settings", icon: Settings },
]

export function DashboardLayout({ children }: DashboardLayoutProps) {
  const { user, logout } = useAuth()
  const router = useRouter()
  const pathname = usePathname()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)

  const handleLogout = () => {
    logout()
    router.push("/")
  }

  return (
    <div className="min-h-screen flex">
      {/* Sidebar - Desktop */}
      <aside className="hidden lg:flex lg:flex-col lg:w-64 border-r border-slate-200 bg-white fixed left-0 top-0 h-screen">
        <div className="flex items-center gap-2 h-16 px-6 border-b border-slate-200">
          <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-blue-600 to-blue-700 flex items-center justify-center">
            <TrendingUp className="h-5 w-5 text-white" />
          </div>
          <span className="text-xl font-bold text-slate-900">CryptoTrade</span>
        </div>

        <nav className="flex-1 px-4 py-6 space-y-1 overflow-y-auto">
          {navigation.map((item) => {
            const isActive = pathname === item.href
            return (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors",
                  isActive
                    ? "bg-blue-600 text-white"
                    : "text-slate-700 hover:bg-slate-100 hover:text-slate-900",
                )}
              >
                <item.icon className="h-5 w-5" />
                {item.name}
              </Link>
            )
          })}
        </nav>

        <div className="p-4 border-t border-slate-200">
          <Link href="/settings" className="block">
            <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-slate-100 hover:bg-slate-200 transition-colors cursor-pointer">
              <div className="h-10 w-10 rounded-full bg-gradient-to-br from-blue-600 to-blue-700 flex items-center justify-center">
                <span className="text-sm font-semibold text-white">
                  {user?.first_name && user?.last_name 
                    ? `${user.first_name[0]}${user.last_name[0]}`
                    : user?.username?.[0]?.toUpperCase() || 'U'
                  }
                </span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate text-slate-900">
                  {user?.first_name && user?.last_name 
                    ? `${user.first_name} ${user.last_name}`
                    : user?.username || 'User'
                  }
                </p>
                <p className="text-xs text-slate-600 truncate">{user?.email}</p>
              </div>
            </div>
          </Link>
          <Button variant="ghost" className="w-full justify-start text-slate-700 hover:text-slate-900 hover:bg-slate-100" onClick={handleLogout}>
            <LogOut className="h-4 w-4 mr-2" />
            Logout
          </Button>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col lg:ml-64">
        {/* Mobile Header */}
        <header className="lg:hidden flex items-center justify-between h-16 px-4 border-b border-border bg-white">
          <div className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-blue-600 to-blue-700 flex items-center justify-center">
              <TrendingUp className="h-5 w-5 text-white" />
            </div>
            <span className="text-xl font-bold text-slate-900">CryptoTrade</span>
          </div>
          <Button variant="ghost" size="icon" onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)} className="text-slate-700 hover:text-slate-900">
            {isMobileMenuOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
          </Button>
        </header>

        {/* Mobile Menu */}
        {isMobileMenuOpen && (
          <div className="lg:hidden border-b border-border bg-white p-4">
            <nav className="space-y-1 mb-4">
              {navigation.map((item) => {
                const isActive = pathname === item.href
                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    onClick={() => setIsMobileMenuOpen(false)}
                    className={cn(
                      "flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors",
                      isActive
                        ? "bg-blue-600 text-white"
                        : "text-slate-700 hover:bg-slate-100 hover:text-slate-900",
                    )}
                  >
                    <item.icon className="h-5 w-5" />
                    {item.name}
                  </Link>
                )
              })}
            </nav>
            <Button variant="outline" className="w-full justify-start bg-transparent border-slate-300 text-slate-700 hover:bg-slate-50" onClick={handleLogout}>
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        )}

        {/* Page Content */}
        <main className="flex-1 overflow-auto">
          <div className="container mx-auto p-6 lg:p-8">{children}</div>
        </main>
      </div>
    </div>
  )
}
