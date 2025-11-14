"use client"

import { useEffect, useState, Suspense } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { searchApiService, SearchOrderRequest, OrderSearchResult, SearchResponse } from "@/lib/search-api"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Search,
  Filter,
  X,
  ChevronLeft,
  ChevronRight,
  Loader2,
  Calendar,
  DollarSign,
  TrendingUp,
  TrendingDown,
  Clock,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Eye
} from "lucide-react"
import { useToast } from "@/hooks/use-toast"

// Filter panel component
function FilterPanel({
  filters,
  onFilterChange,
  onClearFilters,
  facets
}: {
  filters: SearchOrderRequest
  onFilterChange: (filters: SearchOrderRequest) => void
  onClearFilters: () => void
  facets: any
}) {
  const [localFilters, setLocalFilters] = useState(filters)

  const updateFilter = (key: string, value: any) => {
    const newFilters = { ...localFilters, [key]: value }
    setLocalFilters(newFilters)
    onFilterChange(newFilters)
  }

  const toggleArrayFilter = (key: 'status' | 'type' | 'order_kind' | 'crypto_symbol', value: string) => {
    const currentArray = localFilters[key] || []
    const newArray = currentArray.includes(value)
      ? currentArray.filter(v => v !== value)
      : [...currentArray, value]
    updateFilter(key, newArray.length > 0 ? newArray : undefined)
  }

  const statusOptions = [
    { value: 'pending', label: 'Pending', icon: Clock, color: 'yellow' },
    { value: 'executed', label: 'Executed', icon: CheckCircle2, color: 'green' },
    { value: 'cancelled', label: 'Cancelled', icon: XCircle, color: 'gray' },
    { value: 'failed', label: 'Failed', icon: AlertCircle, color: 'red' },
  ]

  const typeOptions = [
    { value: 'buy', label: 'Buy', icon: TrendingUp, color: 'green' },
    { value: 'sell', label: 'Sell', icon: TrendingDown, color: 'red' },
  ]

  const orderKindOptions = [
    { value: 'market', label: 'Market' },
    { value: 'limit', label: 'Limit' },
  ]

  const topCryptos = ['BTC', 'ETH', 'BNB', 'SOL', 'XRP', 'ADA', 'DOGE', 'AVAX']

  const hasActiveFilters = localFilters.status?.length || localFilters.type?.length ||
    localFilters.order_kind?.length || localFilters.crypto_symbol?.length ||
    localFilters.min_total_amount || localFilters.max_total_amount ||
    localFilters.date_from || localFilters.date_to

  return (
    <Card className="p-6 bg-black border border-white/10">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Filter className="h-5 w-5 text-white" />
          <h3 className="text-lg font-semibold text-white">Filters</h3>
        </div>
        {hasActiveFilters && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onClearFilters}
            className="text-white/60 hover:text-white"
          >
            <X className="h-4 w-4 mr-1" />
            Clear All
          </Button>
        )}
      </div>

      <div className="space-y-6">
        {/* Status Filter */}
        <div>
          <label className="text-sm font-medium text-white/80 mb-2 block">Status</label>
          <div className="grid grid-cols-2 gap-2">
            {statusOptions.map(option => {
              const Icon = option.icon
              const isSelected = localFilters.status?.includes(option.value)
              const count = facets?.facet_counts?.facet_fields?.status?.[option.value] || 0

              return (
                <button
                  key={option.value}
                  onClick={() => toggleArrayFilter('status', option.value)}
                  className={`flex items-center justify-between p-3 rounded-lg border transition-colors ${
                    isSelected
                      ? 'bg-blue-500/20 border-blue-500'
                      : 'bg-white/5 border-white/10 hover:bg-white/10'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <Icon className={`h-4 w-4 text-${option.color}-400`} />
                    <span className="text-sm text-white">{option.label}</span>
                  </div>
                  <span className="text-xs text-white/60">{count}</span>
                </button>
              )
            })}
          </div>
        </div>

        {/* Type Filter */}
        <div>
          <label className="text-sm font-medium text-white/80 mb-2 block">Order Type</label>
          <div className="grid grid-cols-2 gap-2">
            {typeOptions.map(option => {
              const Icon = option.icon
              const isSelected = localFilters.type?.includes(option.value)
              const count = facets?.facet_counts?.facet_fields?.type?.[option.value] || 0

              return (
                <button
                  key={option.value}
                  onClick={() => toggleArrayFilter('type', option.value)}
                  className={`flex items-center justify-between p-3 rounded-lg border transition-colors ${
                    isSelected
                      ? 'bg-blue-500/20 border-blue-500'
                      : 'bg-white/5 border-white/10 hover:bg-white/10'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <Icon className={`h-4 w-4 text-${option.color}-400`} />
                    <span className="text-sm text-white">{option.label}</span>
                  </div>
                  <span className="text-xs text-white/60">{count}</span>
                </button>
              )
            })}
          </div>
        </div>

        {/* Order Kind Filter */}
        <div>
          <label className="text-sm font-medium text-white/80 mb-2 block">Order Kind</label>
          <div className="grid grid-cols-2 gap-2">
            {orderKindOptions.map(option => {
              const isSelected = localFilters.order_kind?.includes(option.value)
              const count = facets?.facet_counts?.facet_fields?.order_kind?.[option.value] || 0

              return (
                <button
                  key={option.value}
                  onClick={() => toggleArrayFilter('order_kind', option.value)}
                  className={`flex items-center justify-between p-3 rounded-lg border transition-colors ${
                    isSelected
                      ? 'bg-blue-500/20 border-blue-500'
                      : 'bg-white/5 border-white/10 hover:bg-white/10'
                  }`}
                >
                  <span className="text-sm text-white">{option.label}</span>
                  <span className="text-xs text-white/60">{count}</span>
                </button>
              )
            })}
          </div>
        </div>

        {/* Cryptocurrency Filter */}
        <div>
          <label className="text-sm font-medium text-white/80 mb-2 block">Cryptocurrency</label>
          <div className="flex flex-wrap gap-2">
            {topCryptos.map(symbol => {
              const isSelected = localFilters.crypto_symbol?.includes(symbol)
              const count = facets?.facet_counts?.facet_fields?.crypto_symbol?.[symbol] || 0

              return (
                <button
                  key={symbol}
                  onClick={() => toggleArrayFilter('crypto_symbol', symbol)}
                  className={`px-3 py-2 rounded-lg border text-sm transition-colors ${
                    isSelected
                      ? 'bg-blue-500/20 border-blue-500 text-white'
                      : 'bg-white/5 border-white/10 text-white/80 hover:bg-white/10'
                  }`}
                >
                  {symbol}
                  {count > 0 && <span className="ml-2 text-xs text-white/60">({count})</span>}
                </button>
              )
            })}
          </div>
        </div>

        {/* Amount Range Filter */}
        <div>
          <label className="text-sm font-medium text-white/80 mb-2 block">Amount Range (USD)</label>
          <div className="grid grid-cols-2 gap-2">
            <Input
              type="number"
              placeholder="Min"
              value={localFilters.min_total_amount || ''}
              onChange={(e) => updateFilter('min_total_amount', e.target.value ? parseFloat(e.target.value) : undefined)}
              className="bg-black border-white/10 text-white"
            />
            <Input
              type="number"
              placeholder="Max"
              value={localFilters.max_total_amount || ''}
              onChange={(e) => updateFilter('max_total_amount', e.target.value ? parseFloat(e.target.value) : undefined)}
              className="bg-black border-white/10 text-white"
            />
          </div>
        </div>

        {/* Date Range Filter */}
        <div>
          <label className="text-sm font-medium text-white/80 mb-2 block">Date Range</label>
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-white/60" />
              <Input
                type="date"
                value={localFilters.date_from || ''}
                onChange={(e) => updateFilter('date_from', e.target.value || undefined)}
                className="bg-black border-white/10 text-white"
              />
            </div>
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-white/60" />
              <Input
                type="date"
                value={localFilters.date_to || ''}
                onChange={(e) => updateFilter('date_to', e.target.value || undefined)}
                className="bg-black border-white/10 text-white"
              />
            </div>
          </div>
        </div>
      </div>
    </Card>
  )
}

// Order card component
function OrderCard({ order, onView }: { order: OrderSearchResult, onView: (order: OrderSearchResult) => void }) {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'executed': return 'bg-green-500/20 text-green-400 border-green-500/30'
      case 'pending': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
      case 'cancelled': return 'bg-gray-500/20 text-gray-400 border-gray-500/30'
      case 'failed': return 'bg-red-500/20 text-red-400 border-red-500/30'
      default: return 'bg-white/10 text-white border-white/20'
    }
  }

  const getTypeColor = (type: string) => {
    return type === 'buy' ? 'text-green-400' : 'text-red-400'
  }

  const getTypeIcon = (type: string) => {
    return type === 'buy' ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  return (
    <Card className="p-4 bg-black border border-white/10 hover:border-white/20 transition-colors">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className={`flex items-center gap-1 ${getTypeColor(order.type)}`}>
            {getTypeIcon(order.type)}
            <span className="font-semibold capitalize">{order.type}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xl font-bold text-white">{order.crypto_symbol}</span>
            <span className="text-sm text-white/60">{order.crypto_name}</span>
          </div>
        </div>
        <Badge className={`${getStatusColor(order.status)} border`}>
          {order.status}
        </Badge>
      </div>

      <div className="grid grid-cols-2 gap-3 mb-3">
        <div>
          <p className="text-xs text-white/60 mb-1">Quantity</p>
          <p className="text-sm font-semibold text-white">{parseFloat(order.quantity).toFixed(8)} {order.crypto_symbol}</p>
        </div>
        <div>
          <p className="text-xs text-white/60 mb-1">Price</p>
          <p className="text-sm font-semibold text-white">${parseFloat(order.price).toLocaleString()}</p>
        </div>
        <div>
          <p className="text-xs text-white/60 mb-1">Total Amount</p>
          <p className="text-sm font-bold text-white">${parseFloat(order.total_amount).toLocaleString()}</p>
        </div>
        <div>
          <p className="text-xs text-white/60 mb-1">Fee</p>
          <p className="text-sm text-white">${parseFloat(order.fee).toFixed(2)}</p>
        </div>
      </div>

      <div className="flex items-center justify-between pt-3 border-t border-white/10">
        <div className="flex items-center gap-2 text-xs text-white/60">
          <Clock className="h-3 w-3" />
          <span>{formatDate(order.created_at)}</span>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onView(order)}
          className="text-blue-400 hover:text-blue-300"
        >
          <Eye className="h-4 w-4 mr-1" />
          Details
        </Button>
      </div>
    </Card>
  )
}

function OrdersContent() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const { toast } = useToast()

  const [searchQuery, setSearchQuery] = useState("")
  const [filters, setFilters] = useState<SearchOrderRequest>({
    page: 1,
    limit: 20,
    sort: 'created_at_desc',
  })
  const [searchResults, setSearchResults] = useState<SearchResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [selectedOrder, setSelectedOrder] = useState<OrderSearchResult | null>(null)

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  useEffect(() => {
    if (user) {
      performSearch()
    }
  }, [user, filters])

  const performSearch = async () => {
    if (!user) return

    try {
      setLoading(true)

      // List of known crypto symbols
      const knownCryptos = ['BTC', 'ETH', 'BNB', 'SOL', 'XRP', 'ADA', 'DOGE', 'AVAX', 'MATIC', 'DOT', 'LINK', 'UNI', 'LTC', 'BCH', 'ATOM', 'NEAR', 'ALGO', 'VET', 'ICP', 'FIL']

      // Check if search query is a crypto symbol
      const upperQuery = searchQuery.trim().toUpperCase()
      const isCryptoSymbol = knownCryptos.includes(upperQuery)

      // Convert dates to ISO 8601 format if present
      const formatDateToISO = (dateStr: string | undefined) => {
        if (!dateStr) return undefined
        // Date input gives us YYYY-MM-DD, convert to ISO 8601 with timezone
        return new Date(dateStr + 'T00:00:00Z').toISOString()
      }

      const searchParams = {
        ...filters,
        user_id: user.id,
        // If searching for a crypto symbol, use crypto_symbol filter instead of q
        crypto_symbol: isCryptoSymbol && searchQuery ? [upperQuery] : filters.crypto_symbol,
        q: !isCryptoSymbol && searchQuery ? searchQuery : undefined,
        // Convert dates to ISO 8601 format
        date_from: formatDateToISO(filters.date_from),
        date_to: formatDateToISO(filters.date_to),
      }

      const results = await searchApiService.searchOrders(searchParams)
      setSearchResults(results)
    } catch (error) {
      console.error('Search failed:', error)
      toast({
        title: "Search Failed",
        description: "Failed to search orders. Please try again.",
        variant: "destructive"
      })
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = () => {
    setFilters(prev => ({ ...prev, page: 1 }))
    performSearch()
  }

  const handleFilterChange = (newFilters: SearchOrderRequest) => {
    setFilters({ ...newFilters, page: 1 })
  }

  const handleClearFilters = () => {
    setSearchQuery("")
    setFilters({
      page: 1,
      limit: 20,
      sort: 'created_at_desc',
    })
  }

  const handlePageChange = (newPage: number) => {
    setFilters(prev => ({ ...prev, page: newPage }))
  }

  const handleSortChange = (newSort: string) => {
    setFilters(prev => ({ ...prev, sort: newSort, page: 1 }))
  }

  if (isLoading || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-black">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    )
  }

  const sortOptions = [
    { value: 'created_at_desc', label: 'Newest First' },
    { value: 'created_at_asc', label: 'Oldest First' },
    { value: 'updated_at_desc', label: 'Recently Updated' },
    { value: 'total_amount_desc', label: 'Highest Amount' },
    { value: 'total_amount_asc', label: 'Lowest Amount' },
    { value: 'executed_at_desc', label: 'Recently Executed' },
  ]

  return (
    <DashboardLayout>
      <div className="min-h-screen bg-black p-6">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">Orders</h1>
          <p className="text-white/60 text-lg">Search and manage your trading orders</p>
        </div>

        <div className="grid grid-cols-12 gap-6">
          {/* Filters Sidebar */}
          <div className="col-span-12 lg:col-span-3">
            <FilterPanel
              filters={filters}
              onFilterChange={handleFilterChange}
              onClearFilters={handleClearFilters}
              facets={searchResults}
            />
          </div>

          {/* Main Content */}
          <div className="col-span-12 lg:col-span-9 space-y-6">
            {/* Search Bar */}
            <Card className="p-4 bg-black border border-white/10">
              <div className="flex gap-3">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-white/60" />
                  <Input
                    placeholder="Search by crypto symbol (e.g., BTC, ETH)..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                    className="pl-10 bg-black border-white/10 text-white"
                  />
                </div>
                <Button
                  onClick={handleSearch}
                  disabled={loading}
                  className="bg-blue-600 hover:bg-blue-700"
                >
                  {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Search'}
                </Button>
              </div>
            </Card>

            {/* Results Header */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <p className="text-white/80">
                  {searchResults && (
                    <>
                      <span className="font-semibold text-white">{searchResults.total}</span> orders found
                    </>
                  )}
                </p>
              </div>

              <div className="flex items-center gap-3">
                <span className="text-sm text-white/60">Sort by:</span>
                <Select value={filters.sort} onValueChange={handleSortChange}>
                  <SelectTrigger className="w-48 bg-black border-white/10 text-white">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="bg-black border-white/10">
                    {sortOptions.map(option => (
                      <SelectItem key={option.value} value={option.value} className="text-white">
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Results Grid */}
            {loading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
              </div>
            ) : searchResults && searchResults.results.length > 0 ? (
              <div className="grid grid-cols-1 gap-4">
                {searchResults.results.map(order => (
                  <OrderCard
                    key={order.id}
                    order={order}
                    onView={setSelectedOrder}
                  />
                ))}
              </div>
            ) : (
              <Card className="p-12 bg-black border border-white/10">
                <div className="text-center">
                  <Search className="h-12 w-12 text-white/20 mx-auto mb-4" />
                  <h3 className="text-xl font-semibold text-white mb-2">No orders found</h3>
                  <p className="text-white/60">Try adjusting your filters or search query</p>
                </div>
              </Card>
            )}

            {/* Pagination */}
            {searchResults && searchResults.total_pages > 1 && (
              <div className="flex items-center justify-between">
                <Button
                  variant="outline"
                  onClick={() => handlePageChange(filters.page! - 1)}
                  disabled={filters.page === 1}
                  className="bg-black border-white/10 text-white"
                >
                  <ChevronLeft className="h-4 w-4 mr-1" />
                  Previous
                </Button>

                <div className="flex items-center gap-2">
                  <span className="text-white/60">
                    Page {filters.page} of {searchResults.total_pages}
                  </span>
                </div>

                <Button
                  variant="outline"
                  onClick={() => handlePageChange(filters.page! + 1)}
                  disabled={filters.page === searchResults.total_pages}
                  className="bg-black border-white/10 text-white"
                >
                  Next
                  <ChevronRight className="h-4 w-4 ml-1" />
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Order Detail Modal */}
        {selectedOrder && (
          <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4"
               onClick={() => setSelectedOrder(null)}>
            <Card className="max-w-2xl w-full bg-black border border-white/10 p-6"
                  onClick={(e) => e.stopPropagation()}>
              <div className="flex items-start justify-between mb-6">
                <h2 className="text-2xl font-bold text-white">Order Details</h2>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSelectedOrder(null)}
                  className="text-white/60 hover:text-white"
                >
                  <X className="h-5 w-5" />
                </Button>
              </div>

              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-white/60 mb-1">Order ID</p>
                    <p className="text-white font-mono text-sm">{selectedOrder.id}</p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Status</p>
                    <Badge className={`${
                      selectedOrder.status === 'executed' ? 'bg-green-500/20 text-green-400' :
                      selectedOrder.status === 'pending' ? 'bg-yellow-500/20 text-yellow-400' :
                      selectedOrder.status === 'cancelled' ? 'bg-gray-500/20 text-gray-400' :
                      'bg-red-500/20 text-red-400'
                    }`}>
                      {selectedOrder.status}
                    </Badge>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Type</p>
                    <p className={`font-semibold ${selectedOrder.type === 'buy' ? 'text-green-400' : 'text-red-400'}`}>
                      {selectedOrder.type.toUpperCase()}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Order Kind</p>
                    <p className="text-white">{selectedOrder.order_kind}</p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Cryptocurrency</p>
                    <p className="text-white font-semibold">{selectedOrder.crypto_symbol} - {selectedOrder.crypto_name}</p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Quantity</p>
                    <p className="text-white">{parseFloat(selectedOrder.quantity).toFixed(8)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Price</p>
                    <p className="text-white font-semibold">${parseFloat(selectedOrder.price).toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Total Amount</p>
                    <p className="text-white font-bold text-lg">${parseFloat(selectedOrder.total_amount).toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Fee (0.1%)</p>
                    <p className="text-white">${parseFloat(selectedOrder.fee).toFixed(2)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-white/60 mb-1">Created At</p>
                    <p className="text-white text-sm">{new Date(selectedOrder.created_at).toLocaleString()}</p>
                  </div>
                  {selectedOrder.executed_at && (
                    <div>
                      <p className="text-sm text-white/60 mb-1">Executed At</p>
                      <p className="text-white text-sm">{new Date(selectedOrder.executed_at).toLocaleString()}</p>
                    </div>
                  )}
                  {selectedOrder.cancelled_at && (
                    <div>
                      <p className="text-sm text-white/60 mb-1">Cancelled At</p>
                      <p className="text-white text-sm">{new Date(selectedOrder.cancelled_at).toLocaleString()}</p>
                    </div>
                  )}
                </div>
              </div>
            </Card>
          </div>
        )}
      </div>
    </DashboardLayout>
  )
}

export default function OrdersPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-black">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    }>
      <OrdersContent />
    </Suspense>
  )
}
