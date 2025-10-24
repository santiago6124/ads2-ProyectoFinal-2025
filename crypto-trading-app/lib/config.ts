// Configuration for the application
export const config = {
  apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001',
  marketApiUrl: process.env.NEXT_PUBLIC_MARKET_API_URL || 'http://localhost:8004',
  appName: 'CryptoTrade',
  version: '1.0.0',
} as const
