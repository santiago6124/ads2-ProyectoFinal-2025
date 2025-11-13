// Configuration for the application
export const config = {
  apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001',
  ordersApiUrl: process.env.NEXT_PUBLIC_ORDERS_API_URL || 'http://localhost:8002',
  marketApiUrl: process.env.NEXT_PUBLIC_MARKET_API_URL || 'http://localhost:8004',
  portfolioApiUrl: process.env.NEXT_PUBLIC_PORTFOLIO_API_URL || 'http://localhost:8005',
  searchApiUrl: process.env.NEXT_PUBLIC_SEARCH_API_URL || 'http://localhost:8003',
  appName: 'CryptoTrade',
  version: '1.0.0',
} as const
