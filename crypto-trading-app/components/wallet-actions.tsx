"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ArrowDownToLine, ArrowUpFromLine } from "lucide-react"

export function WalletActions() {
  const [depositAmount, setDepositAmount] = useState("")
  const [withdrawAmount, setWithdrawAmount] = useState("")
  const [depositOpen, setDepositOpen] = useState(false)
  const [withdrawOpen, setWithdrawOpen] = useState(false)

  const handleDeposit = () => {
    if (!depositAmount || Number.parseFloat(depositAmount) <= 0) return
    alert(`Deposit request: $${depositAmount}`)
    setDepositAmount("")
    setDepositOpen(false)
  }

  const handleWithdraw = () => {
    if (!withdrawAmount || Number.parseFloat(withdrawAmount) <= 0) return
    alert(`Withdrawal request: $${withdrawAmount}`)
    setWithdrawAmount("")
    setWithdrawOpen(false)
  }

  return (
    <div className="flex gap-3">
      <Dialog open={depositOpen} onOpenChange={setDepositOpen}>
        <DialogTrigger asChild>
          <Button className="gap-2">
            <ArrowDownToLine className="h-4 w-4" />
            Deposit
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Deposit Funds</DialogTitle>
            <DialogDescription>Add funds to your trading account</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="deposit-amount">Amount (USD)</Label>
              <Input
                id="deposit-amount"
                type="number"
                placeholder="0.00"
                value={depositAmount}
                onChange={(e) => setDepositAmount(e.target.value)}
                step="0.01"
              />
            </div>
            <div className="p-4 rounded-lg bg-accent/50 border border-border">
              <p className="text-sm text-muted-foreground mb-2">Deposit Methods</p>
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span>Bank Transfer</span>
                  <span className="text-muted-foreground">1-3 business days</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span>Credit/Debit Card</span>
                  <span className="text-muted-foreground">Instant</span>
                </div>
              </div>
            </div>
            <Button className="w-full" onClick={handleDeposit}>
              Confirm Deposit
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={withdrawOpen} onOpenChange={setWithdrawOpen}>
        <DialogTrigger asChild>
          <Button variant="outline" className="gap-2 bg-transparent">
            <ArrowUpFromLine className="h-4 w-4" />
            Withdraw
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Withdraw Funds</DialogTitle>
            <DialogDescription>Transfer funds from your trading account</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="p-4 rounded-lg bg-accent/50 border border-border">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Available Balance</span>
                <span className="text-lg font-bold">$10,000.00</span>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="withdraw-amount">Amount (USD)</Label>
              <Input
                id="withdraw-amount"
                type="number"
                placeholder="0.00"
                value={withdrawAmount}
                onChange={(e) => setWithdrawAmount(e.target.value)}
                step="0.01"
              />
            </div>
            <div className="p-4 rounded-lg bg-accent/50 border border-border">
              <p className="text-sm text-muted-foreground mb-2">Withdrawal Info</p>
              <div className="space-y-2 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Processing Time</span>
                  <span>1-3 business days</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Fee</span>
                  <span>$0.00</span>
                </div>
              </div>
            </div>
            <Button className="w-full" onClick={handleWithdraw}>
              Confirm Withdrawal
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
