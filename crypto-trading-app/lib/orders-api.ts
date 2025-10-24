// Orders API service for creating and managing trading orders

export interface OrderRequest {
  type: "buy" | "sell"
  crypto_symbol: string
  quantity: string
  order_kind: "market" | "limit"  // Changed from order_type to order_kind
  order_price?: string  // Added order_price field
}

export interface OrderResponse {
  id: string
  user_id: number
  symbol: string
  order_type: "buy" | "sell"
  quantity: number
  price: number
  status: "pending" | "executed" | "cancelled" | "failed"
  created_at: string
  executed_at?: string
  total_cost?: number
  total_value?: number
}

class OrdersApiService {
  private baseUrl: string

  constructor() {
    this.baseUrl = process.env.NEXT_PUBLIC_ORDERS_API_URL || 'http://localhost:8002'
  }

  async createOrder(orderData: OrderRequest): Promise<OrderResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/orders`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('crypto_access_token')}` // Using correct token key
        },
        body: JSON.stringify(orderData)
      })

      if (!response.ok) {
        let errorMessage = 'Failed to create order'
        try {
          const errorData = await response.json()
          errorMessage = errorData.message || errorData.error || errorMessage
        } catch (parseError) {
          // If response is not JSON, use status text
          errorMessage = response.statusText || errorMessage
        }
        throw new Error(errorMessage)
      }

      return await response.json()
    } catch (error) {
      console.error('Error creating order:', error)
      throw error
    }
  }

  async getOrders(userId: number): Promise<OrderResponse[]> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/users/${userId}/orders`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('crypto_access_token')}`
        }
      })

      if (!response.ok) {
        throw new Error('Failed to fetch orders')
      }

      return await response.json()
    } catch (error) {
      console.error('Error fetching orders:', error)
      throw error
    }
  }

  async getOrder(orderId: string): Promise<OrderResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/orders/${orderId}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('crypto_access_token')}`
        }
      })

      if (!response.ok) {
        throw new Error('Failed to fetch order')
      }

      return await response.json()
    } catch (error) {
      console.error('Error fetching order:', error)
      throw error
    }
  }

  async cancelOrder(orderId: string): Promise<void> {
    try {
      const response = await fetch(`${this.baseUrl}/api/v1/orders/${orderId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('crypto_access_token')}`
        }
      })

      if (!response.ok) {
        throw new Error('Failed to cancel order')
      }
    } catch (error) {
      console.error('Error cancelling order:', error)
      throw error
    }
  }

  // Health check
  async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseUrl}/health`)
      return response.ok
    } catch (error) {
      return false
    }
  }
}

export const ordersApiService = new OrdersApiService()
