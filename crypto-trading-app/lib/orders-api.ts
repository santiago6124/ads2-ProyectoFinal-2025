// Orders API service for creating and managing trading orders

export interface OrderRequest {
  user_id: number
  symbol: string
  order_type: "buy" | "sell"
  quantity: number
  price: number
  total_cost?: number
  total_value?: number
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
      const response = await fetch(`${this.baseUrl}/api/orders`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}` // Assuming JWT token is stored
        },
        body: JSON.stringify(orderData)
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to create order')
      }

      return await response.json()
    } catch (error) {
      console.error('Error creating order:', error)
      throw error
    }
  }

  async getOrders(userId: number): Promise<OrderResponse[]> {
    try {
      const response = await fetch(`${this.baseUrl}/api/orders/user/${userId}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
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
      const response = await fetch(`${this.baseUrl}/api/orders/${orderId}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
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
      const response = await fetch(`${this.baseUrl}/api/orders/${orderId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
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
