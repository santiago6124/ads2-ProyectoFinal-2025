// Configuration for the application
export const config = {
  apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001',
  ordersApiUrl: process.env.NEXT_PUBLIC_ORDERS_API_URL || 'http://localhost:8002',
  marketApiUrl: process.env.NEXT_PUBLIC_MARKET_API_URL || 'http://localhost:8004',
  walletApiUrl: process.env.NEXT_PUBLIC_WALLET_API_URL || 'http://localhost:8006',
  appName: 'CryptoTrade',
  version: '1.0.0',
} as const
