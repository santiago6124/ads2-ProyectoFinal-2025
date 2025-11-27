"use client"

import { useEffect, useState } from "react"
import confetti from "canvas-confetti"
import { motion, AnimatePresence } from "framer-motion"

interface CongratsAnimationProps {
  isVisible: boolean
  type: "buy" | "sell"
  onComplete?: () => void
}

export function CongratsAnimation({ isVisible, type, onComplete }: CongratsAnimationProps) {
  const [showText, setShowText] = useState(false)

  useEffect(() => {
    if (isVisible) {
      setShowText(true)

      // Play sound
      const audio = new Audio("/sounds/cash-register.mp3")
      audio.volume = 0.5
      audio.play().catch(e => console.error("Error playing sound:", e))

      // Trigger confetti
      const duration = 3000
      const end = Date.now() + duration

      const colors = type === "buy"
        ? ["#22c55e", "#16a34a", "#ffffff", "#fbbf24"] // Green & Gold
        : ["#ef4444", "#dc2626", "#ffffff", "#fbbf24"] // Red & Gold

      const frame = () => {
        confetti({
          particleCount: 5,
          angle: 60,
          spread: 55,
          origin: { x: 0 },
          colors: colors
        })
        confetti({
          particleCount: 5,
          angle: 120,
          spread: 55,
          origin: { x: 1 },
          colors: colors
        })

        if (Date.now() < end) {
          requestAnimationFrame(frame)
        } else {
          setTimeout(() => {
            setShowText(false)
            if (onComplete) onComplete()
          }, 1000)
        }
      }

      frame()
    }
  }, [isVisible, type, onComplete])

  return (
    <AnimatePresence>
      {isVisible && showText && (
        <motion.div
          initial={{ opacity: 0, scale: 0.5 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 1.5 }}
          className="fixed inset-0 z-50 flex items-center justify-center pointer-events-none"
        >
          <div className="relative">
            {/* Background Glow */}
            <div className={`absolute inset-0 blur-3xl opacity-50 ${type === "buy" ? "bg-green-500" : "bg-red-500"
              }`} />

            {/* Main Text */}
            <h1 className={`relative text-6xl md:text-8xl font-black tracking-tighter text-center transform -rotate-6 drop-shadow-2xl ${type === "buy"
                ? "text-transparent bg-clip-text bg-gradient-to-b from-yellow-300 via-yellow-500 to-yellow-700 stroke-white"
                : "text-transparent bg-clip-text bg-gradient-to-b from-red-300 via-red-500 to-red-700"
              }`}
              style={{
                textShadow: "0 0 10px rgba(0,0,0,0.5), 0 0 20px rgba(0,0,0,0.3)",
                WebkitTextStroke: "2px white"
              }}
            >
              {type === "buy" ? "JACKPOT!" : "SOLD!"}
            </h1>

            {/* Subtitle */}
            <motion.p
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.2 }}
              className="text-white text-2xl md:text-3xl font-bold text-center mt-4 drop-shadow-lg"
            >
              {type === "buy" ? "SUCCESSFUL PURCHASE" : "SUCCESSFUL SALE"}
            </motion.p>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
