const PORTFOLIO_API_URL = process.env.NEXT_PUBLIC_PORTFOLIO_API_URL || 'http://localhost:8005'

export interface Holding {
  symbol: string
  name: string
  quantity: string
  average_buy_price: string
  current_price: string
  current_value: string
  total_value: string
  profit_loss: string
  profit_loss_percentage: string
  allocation_percentage: string
}

export interface Portfolio {
  id: string
  user_id: number
  total_value: string
  total_invested: string
  profit_loss: string
  profit_loss_percentage: string
  currency: string
  holdings: Holding[]
  created_at: string
  updated_at: string
}

export interface PortfolioAPIError {
  error: string
}

/**
 * Fetch portfolio data for a user
 * @param userId - The user ID
 * @returns Portfolio data
 */
export async function getPortfolio(userId: number): Promise<Portfolio> {
  try {
    const response = await fetch(`${PORTFOLIO_API_URL}/api/portfolios/${userId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store', // Always fetch fresh data
    })

    if (!response.ok) {
      const errorData: PortfolioAPIError = await response.json().catch(() => ({
        error: 'Failed to fetch portfolio'
      }))
      throw new Error(errorData.error || `HTTP error! status: ${response.status}`)
    }

    const data: Portfolio = await response.json()
    console.log('getPortfolio DEBUG: Full response', data)
    if (data.holdings && data.holdings.length > 0) {
      console.log('getPortfolio DEBUG: First holding', data.holdings[0])
    }
    return data
  } catch (error) {
    console.error('Error fetching portfolio:', error)
    throw error
  }
}

/**
 * Format a number with specific decimal places
 * @param value - The value to format (can be string or number)
 * @param decimals - Number of decimal places (8 for crypto, 2 for USD)
 * @returns Formatted string
 */
export function formatNumber(value: string | number, decimals: number = 2): string {
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '0.' + '0'.repeat(decimals)
  return num.toFixed(decimals)
}

/**
 * Format crypto amount (8 decimal places)
 */
export function formatCrypto(value: string | number): string {
  return formatNumber(value, 8)
}

/**
 * Format USD amount (2 decimal places)
 */
export function formatUSD(value: string | number): string {
  const formatted = formatNumber(value, 2)
  return `$${parseFloat(formatted).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`
}

/**
 * Format percentage (2 decimal places with + or - sign)
 * Note: API already returns percentage values (e.g., "3.28" = 3.28%), no need to multiply by 100
 */
export function formatPercentage(value: string | number): string {
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '0.00%'

  const formatted = num.toFixed(2)
  const sign = num >= 0 ? '+' : ''
  return `${sign}${formatted}%`
}

/**
 * Calculate the trend direction from profit/loss
 */
export function getTrend(profitLoss: string | number): 'up' | 'down' {
  const num = typeof profitLoss === 'string' ? parseFloat(profitLoss) : profitLoss
  return num >= 0 ? 'up' : 'down'
}
