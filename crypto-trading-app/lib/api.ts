// API service for communicating with backend microservices
import { config } from './config'

const API_BASE_URL = config.apiUrl

export interface User {
  id: number
  username: string
  email: string
  first_name: string | null
  last_name: string | null
  role: 'normal' | 'admin'
  initial_balance: number
  created_at: string
  last_login?: string
  is_active: boolean
  preferences: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  first_name?: string
  last_name?: string
}

export interface AuthResponse {
  success: boolean
  message: string
  data: {
    user: User
    access_token: string
    refresh_token: string
    expires_in: number
  }
}

export interface RegisterResponse {
  success: boolean
  message: string
  data: User
}

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

class ApiService {
  private baseURL: string

  constructor() {
    this.baseURL = API_BASE_URL
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`
    
    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    }

    try {
      const response = await fetch(url, config)
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new ApiError(response.status, errorData.error || 'Request failed')
      }

      return await response.json()
    } catch (error) {
      if (error instanceof ApiError) {
        throw error
      }
      throw new ApiError(0, 'Network error')
    }
  }

  // Authentication methods
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    return this.request<AuthResponse>('/api/users/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    })
  }

  async register(userData: RegisterRequest): Promise<RegisterResponse> {
    return this.request<RegisterResponse>('/api/users/register', {
      method: 'POST',
      body: JSON.stringify(userData),
    })
  }

  async refreshToken(refreshToken: string): Promise<AuthResponse> {
    return this.request<AuthResponse>('/api/users/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
  }

  async logout(refreshToken: string): Promise<void> {
    await this.request('/api/users/logout', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
  }

  // User methods
  async getUserProfile(userId: number, accessToken: string): Promise<User> {
    const response = await this.request<{ success: boolean; message: string; data: User }>(`/api/users/${userId}`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    })
    return response.data
  }

  async updateUserProfile(
    userId: number,
    userData: { first_name?: string; last_name?: string; preferences?: string },
    accessToken: string
  ): Promise<User> {
    const response = await this.request<{ success: boolean; message: string; data: User }>(`/api/users/${userId}`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify(userData),
    })
    return response.data
  }

  async changePassword(
    userId: number,
    currentPassword: string,
    newPassword: string,
    accessToken: string
  ): Promise<void> {
    await this.request(`/api/users/${userId}/password`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({
        current_password: currentPassword,
        new_password: newPassword,
      }),
    })
  }
}

export const apiService = new ApiService()
export { ApiError }
