"use client"

import { type ReactNode, useState } from "react"
import { useAuth } from "@/lib/auth-context"
import { useRouter, usePathname } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
  FileText,
  ChevronLeft,
  ChevronRight,
  Shield,
  Plus,
} from "lucide-react"
import Link from "next/link"
import { cn } from "@/lib/utils"
import { apiService } from "@/lib/api"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

interface DashboardLayoutProps {
  children: ReactNode
}

const navigation = [
  { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { name: "Markets", href: "/markets", icon: LineChart },
  { name: "Trade", href: "/trade", icon: ArrowLeftRight },
  { name: "Portfolio", href: "/portfolio", icon: Wallet },
  { name: "Orders", href: "/orders", icon: FileText },
  { name: "Settings", href: "/settings", icon: Settings },
]

export function DashboardLayout({ children }: DashboardLayoutProps) {
  const { user, logout, updateBalance } = useAuth()
  const router = useRouter()
  const pathname = usePathname()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false)
  const [addFundsDialogOpen, setAddFundsDialogOpen] = useState(false)
  const [fundsAmount, setFundsAmount] = useState("")
  const [fundsDescription, setFundsDescription] = useState("")
  const [isAddingFunds, setIsAddingFunds] = useState(false)
  const [transactionType, setTransactionType] = useState<'add' | 'withdraw'>('add')

  const handleLogout = () => {
    logout()
    router.push("/")
  }

  const handleAddFunds = async () => {
    if (!fundsAmount) return

    setIsAddingFunds(true)
    try {
      const accessToken = localStorage.getItem('crypto_access_token')
      if (!accessToken) return

      const amount = parseFloat(fundsAmount)
      const finalAmount = transactionType === 'withdraw' ? -amount : amount
      const description = transactionType === 'withdraw'
        ? (fundsDescription || "Withdrawal")
        : (fundsDescription || "Deposit")

      const response = await apiService.addFunds(finalAmount, description, accessToken)

      // Write-through cache: update balance immediately
      if (response.data?.new_balance !== undefined) {
        updateBalance(response.data.new_balance)
      } else if (user?.current_balance !== undefined) {
        updateBalance(user.current_balance + finalAmount)
      }

      const actionText = transactionType === 'withdraw' ? 'withdrawn' : 'added'
      alert(`Successfully ${actionText} $${amount} ${transactionType === 'withdraw' ? 'from' : 'to'} your balance`)
      setAddFundsDialogOpen(false)
      setFundsAmount("")
      setFundsDescription("")
      setTransactionType('add')
    } catch (error: any) {
      console.error('Error processing transaction:', error)
      alert(`Error processing transaction: ${error.message || 'Please try again'}`)
    } finally {
      setIsAddingFunds(false)
    }
  }

  return (
    <div className="min-h-screen flex bg-black">
      {/* Sidebar - Desktop */}
      <aside className={cn(
        "hidden lg:flex lg:flex-col border-r border-white/10 bg-black fixed left-0 top-0 h-screen transition-all duration-300",
        isSidebarCollapsed ? "lg:w-20" : "lg:w-64"
      )}>
        <div className="flex items-center gap-2 h-16 px-6 border-b border-white/10 justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <div className="h-8 w-8 rounded-lg bg-blue-500 flex items-center justify-center border border-white/10 flex-shrink-0">
              <TrendingUp className="h-5 w-5 text-white" />
            </div>
            {!isSidebarCollapsed && <span className="text-xl font-bold text-white truncate">CryptoTrade</span>}
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
            className="text-white/70 hover:text-white hover:bg-white/10 flex-shrink-0"
          >
            {isSidebarCollapsed ? <ChevronRight className="h-5 w-5" /> : <ChevronLeft className="h-5 w-5" />}
          </Button>
        </div>

        <nav className="flex-1 px-4 py-6 space-y-1 overflow-y-auto">
          {navigation.map((item) => {
            const isActive = pathname === item.href
            return (
              <Link
                key={item.name}
                href={item.href}
                title={isSidebarCollapsed ? item.name : undefined}
                className={cn(
                  "flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors",
                  isActive
                    ? "bg-blue-500 text-white"
                    : "text-white/70 hover:bg-white/10 hover:text-white",
                  isSidebarCollapsed && "justify-center"
                )}
              >
                <item.icon className="h-5 w-5 flex-shrink-0" />
                {!isSidebarCollapsed && <span className="truncate">{item.name}</span>}
              </Link>
            )
          })}
          {user?.role === 'admin' && (
            <Link
              href="/admin"
              title={isSidebarCollapsed ? "Admin Panel" : undefined}
              className={cn(
                "flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors border border-purple-500/30",
                pathname === '/admin'
                  ? "bg-purple-500 text-white"
                  : "text-purple-400 hover:bg-purple-500/10 hover:text-purple-300",
                isSidebarCollapsed && "justify-center"
              )}
            >
              <Shield className="h-5 w-5 flex-shrink-0" />
              {!isSidebarCollapsed && <span className="truncate">Admin Panel</span>}
            </Link>
          )}
        </nav>

        <div className="p-4 border-t border-white/10">
          {!isSidebarCollapsed ? (
            <>
              <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-white/5 border border-white/10">
                <Link href="/settings" className="flex items-center gap-3 flex-1 min-w-0 hover:opacity-80 transition-opacity">
                  <div className="h-10 w-10 rounded-full bg-blue-500 flex items-center justify-center border border-white/10 flex-shrink-0">
                    <span className="text-sm font-semibold text-white">
                      {user?.first_name && user?.last_name
                        ? `${user.first_name[0]}${user.last_name[0]}`
                        : user?.username?.[0]?.toUpperCase() || 'U'
                      }
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate text-white">
                      {user?.first_name && user?.last_name
                        ? `${user.first_name} ${user.last_name}`
                        : user?.username || 'User'
                      }
                    </p>
                    <p className="text-xs text-white/60 truncate">{user?.email}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <p className="text-xs text-green-400 font-semibold">
                        ${user?.current_balance?.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) || '0.00'}
                      </p>
                    </div>
                  </div>
                </Link>
                <Dialog open={addFundsDialogOpen} onOpenChange={setAddFundsDialogOpen}>
                  <DialogTrigger asChild>
                    <button className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1 flex-shrink-0">
                      <Plus className="h-3 w-3" />
                      Add
                    </button>
                  </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>Manage Balance</DialogTitle>
                            <DialogDescription>
                              Add or withdraw funds from your trading account
                            </DialogDescription>
                          </DialogHeader>
                          <div className="space-y-4 py-4">
                            <div className="flex gap-2">
                              <Button
                                variant={transactionType === 'add' ? 'default' : 'outline'}
                                onClick={() => setTransactionType('add')}
                                className="flex-1"
                              >
                                Add Funds
                              </Button>
                              <Button
                                variant={transactionType === 'withdraw' ? 'default' : 'outline'}
                                onClick={() => setTransactionType('withdraw')}
                                className="flex-1"
                              >
                                Withdraw
                              </Button>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="funds-amount">Amount (USD)</Label>
                              <Input
                                id="funds-amount"
                                type="number"
                                step="0.01"
                                min="0"
                                placeholder="1000.00"
                                value={fundsAmount}
                                onChange={(e) => setFundsAmount(e.target.value)}
                              />
                              {transactionType === 'withdraw' && user?.current_balance && (
                                <p className="text-xs text-muted-foreground">
                                  Available: ${user.current_balance.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                                </p>
                              )}
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="funds-description">Description (optional)</Label>
                              <Input
                                id="funds-description"
                                placeholder={transactionType === 'add' ? 'e.g., Bank transfer' : 'e.g., Bank withdrawal'}
                                value={fundsDescription}
                                onChange={(e) => setFundsDescription(e.target.value)}
                              />
                            </div>
                            <Button
                              onClick={handleAddFunds}
                              disabled={isAddingFunds || !fundsAmount || (transactionType === 'withdraw' && parseFloat(fundsAmount) > (user?.current_balance || 0))}
                              className="w-full"
                            >
                              {isAddingFunds ? 'Processing...' : (transactionType === 'add' ? 'Add Funds' : 'Withdraw')}
                            </Button>
                          </div>
                        </DialogContent>
                </Dialog>
              </div>
              <Button variant="ghost" className="w-full justify-start text-white/70 hover:text-white hover:bg-white/10" onClick={handleLogout}>
                <LogOut className="h-4 w-4 mr-2" />
                Logout
              </Button>
            </>
          ) : (
            <>
              <Link href="/settings" className="block mb-2">
                <div className="flex items-center justify-center p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors cursor-pointer border border-white/10">
                  <div className="h-10 w-10 rounded-full bg-blue-500 flex items-center justify-center border border-white/10">
                    <span className="text-sm font-semibold text-white">
                      {user?.first_name && user?.last_name
                        ? `${user.first_name[0]}${user.last_name[0]}`
                        : user?.username?.[0]?.toUpperCase() || 'U'
                      }
                    </span>
                  </div>
                </div>
              </Link>
              <Button
                variant="ghost"
                size="icon"
                className="w-full text-white/70 hover:text-white hover:bg-white/10"
                onClick={handleLogout}
                title="Logout"
              >
                <LogOut className="h-5 w-5" />
              </Button>
            </>
          )}
        </div>
      </aside>

      {/* Main Content */}
      <div className={cn(
        "flex-1 flex flex-col transition-all duration-300",
        isSidebarCollapsed ? "lg:ml-20" : "lg:ml-64"
      )}>
        {/* Mobile Header */}
        <header className="lg:hidden flex items-center justify-between h-16 px-4 border-b border-white/10 bg-black">
          <div className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-lg bg-blue-500 flex items-center justify-center border border-white/10">
              <TrendingUp className="h-5 w-5 text-white" />
            </div>
            <span className="text-xl font-bold text-white">CryptoTrade</span>
          </div>
          <Button variant="ghost" size="icon" onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)} className="text-white/70 hover:text-white">
            {isMobileMenuOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
          </Button>
        </header>

        {/* Mobile Menu */}
        {isMobileMenuOpen && (
          <div className="lg:hidden border-b border-white/10 bg-black p-4">
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
                        ? "bg-blue-500 text-white"
                        : "text-white/70 hover:bg-white/10 hover:text-white",
                    )}
                  >
                    <item.icon className="h-5 w-5" />
                    {item.name}
                  </Link>
                )
              })}
              {user?.role === 'admin' && (
                <Link
                  href="/admin"
                  onClick={() => setIsMobileMenuOpen(false)}
                  className={cn(
                    "flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors border border-purple-500/30",
                    pathname === '/admin'
                      ? "bg-purple-500 text-white"
                      : "text-purple-400 hover:bg-purple-500/10 hover:text-purple-300",
                  )}
                >
                  <Shield className="h-5 w-5" />
                  Admin Panel
                </Link>
              )}
            </nav>
            <Button variant="outline" className="w-full justify-start bg-transparent border-white/20 text-white hover:bg-white/10" onClick={handleLogout}>
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        )}

        {/* Page Content */}
        <main className="flex-1 overflow-auto bg-black">
          <div className="container mx-auto p-6 lg:p-8">{children}</div>
        </main>
      </div>
    </div>
  )
}
