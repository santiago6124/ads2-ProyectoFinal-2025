// Configuration for the application
export const config = {
  apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001',
  appName: 'CryptoTrade',
  version: '1.0.0',
} as const
