"use client"

import { createContext, useContext, useState, useEffect, type ReactNode } from "react"
import { apiService, type User, ApiError } from "./api"

interface AuthContextType {
  user: User | null
  login: (email: string, password: string) => Promise<boolean>
  signup: (username: string, email: string, password: string, firstName?: string, lastName?: string) => Promise<boolean>
  logout: () => void
  isLoading: boolean
  error: string | null
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    // Check for stored user session
    const storedUser = localStorage.getItem("crypto_user")
    const storedToken = localStorage.getItem("crypto_access_token")
    
    if (storedUser && storedToken) {
      try {
        const userData = JSON.parse(storedUser)
        setUser(userData)
        
        // Optionally verify token is still valid by fetching user profile
        // This could be done in a background check
      } catch (err) {
        // Clear invalid stored data
        localStorage.removeItem("crypto_user")
        localStorage.removeItem("crypto_access_token")
        localStorage.removeItem("crypto_refresh_token")
      }
    }
    setIsLoading(false)
  }, [])

  const login = async (email: string, password: string): Promise<boolean> => {
    try {
      setError(null)
      setIsLoading(true)
      
      const response = await apiService.login({ email, password })
      
      if (response.success) {
        const { user, access_token, refresh_token } = response.data
        
        setUser(user)
        localStorage.setItem("crypto_user", JSON.stringify(user))
        localStorage.setItem("crypto_access_token", access_token)
        localStorage.setItem("crypto_refresh_token", refresh_token)
        
        return true
      }
      return false
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError("Login failed. Please try again.")
      }
      return false
    } finally {
      setIsLoading(false)
    }
  }

  const signup = async (
    username: string, 
    email: string, 
    password: string, 
    firstName?: string, 
    lastName?: string
  ): Promise<boolean> => {
    try {
      setError(null)
      setIsLoading(true)
      
      const response = await apiService.register({
        username,
        email,
        password,
        first_name: firstName,
        last_name: lastName,
      })
      
      if (response.success) {
        // After successful registration, automatically log in
        return await login(email, password)
      }
      return false
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError("Registration failed. Please try again.")
      }
      return false
    } finally {
      setIsLoading(false)
    }
  }

  const logout = async () => {
    try {
      const refreshToken = localStorage.getItem("crypto_refresh_token")
      if (refreshToken) {
        await apiService.logout(refreshToken)
      }
    } catch (err) {
      // Ignore logout errors
    } finally {
      setUser(null)
      localStorage.removeItem("crypto_user")
      localStorage.removeItem("crypto_access_token")
      localStorage.removeItem("crypto_refresh_token")
    }
  }

  return (
    <AuthContext.Provider value={{ 
      user, 
      login, 
      signup, 
      logout, 
      isLoading, 
      error 
    }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}
