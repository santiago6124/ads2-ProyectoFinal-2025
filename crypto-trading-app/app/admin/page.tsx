"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Shield, DollarSign, Users, Edit, Trash2, UserPlus, RotateCcw } from "lucide-react"
import { apiService, type User } from "@/lib/api"
import { useToast } from "@/hooks/use-toast"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

export default function AdminPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const { toast } = useToast()
  const [users, setUsers] = useState<User[]>([])
  const [isLoadingUsers, setIsLoadingUsers] = useState(true)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [balanceAmount, setBalanceAmount] = useState("")
  const [balanceDescription, setBalanceDescription] = useState("")
  const [isUpdating, setIsUpdating] = useState(false)
  const [balanceDialogOpen, setBalanceDialogOpen] = useState(false)
  const [createUserDialogOpen, setCreateUserDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [userToDelete, setUserToDelete] = useState<number | null>(null)
  const [newUser, setNewUser] = useState({
    username: "",
    email: "",
    password: "",
    first_name: "",
    last_name: "",
    role: "normal" as "normal" | "admin",
    initial_balance: 100000
  })

  useEffect(() => {
    if (!isLoading && (!user || user.role !== 'admin')) {
      router.push("/dashboard")
    }
  }, [user, isLoading, router])

  useEffect(() => {
    if (user && user.role === 'admin') {
      loadUsers()
    }
  }, [user])

  const loadUsers = async () => {
    try {
      setIsLoadingUsers(true)
      const accessToken = localStorage.getItem('crypto_access_token')
      if (!accessToken) return

      const response = await apiService.getAllUsers(accessToken)
      setUsers(response.users || [])
    } catch (error) {
      console.error('Error loading users:', error)
    } finally {
      setIsLoadingUsers(false)
    }
  }

  const handleUpdateBalance = async () => {
    if (!selectedUser || !balanceAmount) return

    setIsUpdating(true)
    try {
      const accessToken = localStorage.getItem('crypto_access_token')
      if (!accessToken) return

      const amount = parseFloat(balanceAmount)
      await apiService.updateUserBalance(
        selectedUser.id,
        amount,
        balanceDescription || `Admin balance adjustment`,
        accessToken
      )

      toast({
        title: "Success",
        description: "Balance updated successfully!"
      })
      setBalanceDialogOpen(false)
      setBalanceAmount("")
      setBalanceDescription("")
      setSelectedUser(null)
      loadUsers()
    } catch (error) {
      console.error('Error updating balance:', error)
      toast({
        title: "Error",
        description: "Error updating balance. Please try again.",
        variant: "destructive"
      })
    } finally {
      setIsUpdating(false)
    }
  }

  const handleDeleteUser = async () => {
    if (!userToDelete) return

    try {
      const accessToken = localStorage.getItem('crypto_access_token')
      if (!accessToken) return

      await apiService.adminDeleteUser(userToDelete, accessToken)
      toast({
        title: "Success",
        description: "User deleted successfully!"
      })
      setDeleteDialogOpen(false)
      setUserToDelete(null)
      loadUsers()
    } catch (error) {
      console.error('Error deleting user:', error)
      toast({
        title: "Error",
        description: "Error deleting user. Please try again.",
        variant: "destructive"
      })
    }
  }

  const handleCreateUser = async () => {
    if (!newUser.username || !newUser.email || !newUser.password) {
      toast({
        title: "Validation Error",
        description: "Please fill in all required fields",
        variant: "destructive"
      })
      return
    }

    setIsUpdating(true)
    try {
      const accessToken = localStorage.getItem('crypto_access_token')
      if (!accessToken) return

      await apiService.adminCreateUser({
        username: newUser.username,
        email: newUser.email,
        password: newUser.password,
        first_name: newUser.first_name || undefined,
        last_name: newUser.last_name || undefined,
        role: newUser.role,
        initial_balance: newUser.initial_balance
      }, accessToken)

      toast({
        title: "Success",
        description: "User created successfully!"
      })
      setCreateUserDialogOpen(false)
      setNewUser({
        username: "",
        email: "",
        password: "",
        first_name: "",
        last_name: "",
        role: "normal",
        initial_balance: 100000
      })
      loadUsers()
    } catch (error: any) {
      console.error('Error creating user:', error)
      toast({
        title: "Error",
        description: `Error creating user: ${error.message || 'Please try again'}`,
        variant: "destructive"
      })
    } finally {
      setIsUpdating(false)
    }
  }

  const handleReactivateUser = async (userId: number) => {
    try {
      const accessToken = localStorage.getItem('crypto_access_token')
      if (!accessToken) return

      await apiService.adminReactivateUser(userId, accessToken)
      toast({
        title: "Success",
        description: "User reactivated successfully!"
      })
      loadUsers()
    } catch (error) {
      console.error('Error reactivating user:', error)
      toast({
        title: "Error",
        description: "Error reactivating user. Please try again.",
        variant: "destructive"
      })
    }
  }

  if (isLoading || !user || user.role !== 'admin') {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
            <Shield className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Admin Panel</h1>
            <p className="text-muted-foreground mt-1">Manage users and balances</p>
          </div>
        </div>

        <Card className="p-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <Users className="h-5 w-5 text-primary" />
              <h2 className="text-xl font-bold">User Management</h2>
            </div>
            <Dialog open={createUserDialogOpen} onOpenChange={setCreateUserDialogOpen}>
              <DialogTrigger asChild>
                <Button>
                  <UserPlus className="h-4 w-4 mr-2" />
                  Create User
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-md">
                <DialogHeader>
                  <DialogTitle>Create New User</DialogTitle>
                  <DialogDescription>
                    Add a new user to the system
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  <div className="space-y-2">
                    <Label htmlFor="new-username">Username *</Label>
                    <Input
                      id="new-username"
                      value={newUser.username}
                      onChange={(e) => setNewUser({...newUser, username: e.target.value})}
                      placeholder="johndoe"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-email">Email *</Label>
                    <Input
                      id="new-email"
                      type="email"
                      value={newUser.email}
                      onChange={(e) => setNewUser({...newUser, email: e.target.value})}
                      placeholder="john@example.com"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-password">Password *</Label>
                    <Input
                      id="new-password"
                      type="password"
                      value={newUser.password}
                      onChange={(e) => setNewUser({...newUser, password: e.target.value})}
                      placeholder="Min. 8 characters"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-first-name">First Name</Label>
                    <Input
                      id="new-first-name"
                      value={newUser.first_name}
                      onChange={(e) => setNewUser({...newUser, first_name: e.target.value})}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-last-name">Last Name</Label>
                    <Input
                      id="new-last-name"
                      value={newUser.last_name}
                      onChange={(e) => setNewUser({...newUser, last_name: e.target.value})}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-role">Role</Label>
                    <select
                      id="new-role"
                      value={newUser.role}
                      onChange={(e) => setNewUser({...newUser, role: e.target.value as "normal" | "admin"})}
                      className="w-full px-3 py-2 border rounded-md"
                    >
                      <option value="normal">Normal</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-balance">Initial Balance</Label>
                    <Input
                      id="new-balance"
                      type="number"
                      step="0.01"
                      value={newUser.initial_balance}
                      onChange={(e) => setNewUser({...newUser, initial_balance: parseFloat(e.target.value)})}
                    />
                  </div>
                  <Button
                    onClick={handleCreateUser}
                    disabled={isUpdating || !newUser.username || !newUser.email || !newUser.password}
                    className="w-full"
                  >
                    {isUpdating ? 'Creating...' : 'Create User'}
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </div>

          {isLoadingUsers ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Username</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Current Balance</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((u) => (
                    <TableRow key={u.id}>
                      <TableCell>{u.id}</TableCell>
                      <TableCell className="font-medium">{u.username}</TableCell>
                      <TableCell>{u.email}</TableCell>
                      <TableCell>
                        <span className={`px-2 py-1 rounded-full text-xs ${
                          u.role === 'admin'
                            ? 'bg-purple-100 text-purple-700'
                            : 'bg-blue-100 text-blue-700'
                        }`}>
                          {u.role}
                        </span>
                      </TableCell>
                      <TableCell className="font-mono">
                        ${u.current_balance?.toLocaleString() || '0'}
                      </TableCell>
                      <TableCell>
                        <span className={`px-2 py-1 rounded-full text-xs ${
                          u.is_active
                            ? 'bg-green-100 text-green-700'
                            : 'bg-red-100 text-red-700'
                        }`}>
                          {u.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          {!u.is_active && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleReactivateUser(u.id)}
                              title="Reactivate user"
                            >
                              <RotateCcw className="h-4 w-4" />
                            </Button>
                          )}
                          <Dialog open={balanceDialogOpen && selectedUser?.id === u.id} onOpenChange={(open) => {
                            setBalanceDialogOpen(open)
                            if (!open) {
                              setSelectedUser(null)
                              setBalanceAmount("")
                              setBalanceDescription("")
                            }
                          }}>
                            <DialogTrigger asChild>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setSelectedUser(u)}
                              >
                                <DollarSign className="h-4 w-4" />
                              </Button>
                            </DialogTrigger>
                            <DialogContent>
                              <DialogHeader>
                                <DialogTitle>Update Balance for {u.username}</DialogTitle>
                                <DialogDescription>
                                  Current balance: ${u.current_balance?.toLocaleString() || '0'}
                                </DialogDescription>
                              </DialogHeader>
                              <div className="space-y-4 py-4">
                                <div className="space-y-2">
                                  <Label htmlFor="amount">Amount (use negative to deduct)</Label>
                                  <Input
                                    id="amount"
                                    type="number"
                                    step="0.01"
                                    placeholder="1000.00"
                                    value={balanceAmount}
                                    onChange={(e) => setBalanceAmount(e.target.value)}
                                  />
                                </div>
                                <div className="space-y-2">
                                  <Label htmlFor="description">Description (optional)</Label>
                                  <Input
                                    id="description"
                                    placeholder="Balance adjustment reason"
                                    value={balanceDescription}
                                    onChange={(e) => setBalanceDescription(e.target.value)}
                                  />
                                </div>
                                <Button
                                  onClick={handleUpdateBalance}
                                  disabled={isUpdating || !balanceAmount}
                                  className="w-full"
                                >
                                  {isUpdating ? 'Updating...' : 'Update Balance'}
                                </Button>
                              </div>
                            </DialogContent>
                          </Dialog>
                          <Dialog open={deleteDialogOpen && userToDelete === u.id} onOpenChange={(open) => {
                            setDeleteDialogOpen(open)
                            if (!open) setUserToDelete(null)
                          }}>
                            <DialogTrigger asChild>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setUserToDelete(u.id)}
                                disabled={u.id === user.id}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </DialogTrigger>
                            <DialogContent>
                              <DialogHeader>
                                <DialogTitle>Delete User</DialogTitle>
                                <DialogDescription>
                                  Are you sure you want to delete {u.username}? This action cannot be undone.
                                </DialogDescription>
                              </DialogHeader>
                              <div className="flex gap-2 justify-end pt-4">
                                <Button
                                  variant="outline"
                                  onClick={() => {
                                    setDeleteDialogOpen(false)
                                    setUserToDelete(null)
                                  }}
                                >
                                  Cancel
                                </Button>
                                <Button
                                  variant="destructive"
                                  onClick={handleDeleteUser}
                                >
                                  Delete
                                </Button>
                              </div>
                            </DialogContent>
                          </Dialog>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </Card>
      </div>
    </DashboardLayout>
  )
}
