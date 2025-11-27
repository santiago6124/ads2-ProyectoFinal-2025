"use client"

import { useEffect, useState } from "react"
import confetti from "canvas-confetti"

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
      audio.volume = 0.6
      audio.play().catch(e => console.error("Error playing sound:", e))

      // Trigger confetti
      const duration = 3000
      const end = Date.now() + duration

      const colors = type === "buy"
        ? ["#22c55e", "#16a34a", "#ffffff", "#fbbf24", "#facc15"] // Green & Gold
        : ["#ef4444", "#dc2626", "#ffffff", "#fbbf24", "#facc15"] // Red & Gold

      const frame = () => {
        confetti({
          particleCount: 8,
          angle: 60,
          spread: 55,
          origin: { x: 0 },
          colors: colors
        })
        confetti({
          particleCount: 8,
          angle: 120,
          spread: 55,
          origin: { x: 1 },
          colors: colors
        })
        confetti({
          particleCount: 5,
          angle: 90,
          spread: 70,
          origin: { x: 0.5, y: 0 },
          colors: colors
        })

        if (Date.now() < end) {
          requestAnimationFrame(frame)
        } else {
          setTimeout(() => {
            setShowText(false)
            if (onComplete) onComplete()
          }, 1500)
        }
      }

      frame()
    } else {
      setShowText(false)
    }
  }, [isVisible, type, onComplete])

  if (!isVisible || !showText) return null

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center pointer-events-none">
      <div className="relative animate-casino-entrance">
        {/* Background Glow with pulse animation */}
        <div 
          className={`absolute inset-0 blur-3xl opacity-60 animate-pulse ${type === "buy" ? "bg-green-500" : "bg-red-500"}`}
          style={{
            transform: "scale(1.5)",
            animation: "pulse 1s cubic-bezier(0.4, 0, 0.6, 1) infinite"
          }}
        />

        {/* Main Text with casino style */}
        <h1 
          className={`relative text-7xl md:text-9xl font-black tracking-tighter text-center transform -rotate-3 drop-shadow-2xl animate-bounce-subtle ${
            type === "buy"
              ? "text-transparent bg-clip-text bg-gradient-to-b from-yellow-300 via-yellow-500 to-yellow-700"
              : "text-transparent bg-clip-text bg-gradient-to-b from-red-300 via-red-500 to-red-700"
          }`}
          style={{
            textShadow: "0 0 20px rgba(255,255,255,0.8), 0 0 40px rgba(255,255,255,0.5), 0 0 60px rgba(255,255,255,0.3), 0 0 80px rgba(0,0,0,0.5)",
            WebkitTextStroke: "3px white",
            filter: "drop-shadow(0 0 10px rgba(255,215,0,0.8))"
          }}
        >
          {type === "buy" ? "🚀 ¡COMPRA EXITOSA! 🚀" : "💰 SOLD! 💰"}
        </h1>

        {/* Subtitle with fade in */}
        <p 
          className="text-white text-3xl md:text-4xl font-bold text-center mt-6 drop-shadow-lg animate-fade-in-up"
          style={{
            animationDelay: "0.2s",
            textShadow: "0 0 10px rgba(0,0,0,0.8), 0 0 20px rgba(255,255,255,0.5)"
          }}
        >
          {type === "buy" ? "✨ COMPRA REALIZADA ✨" : "✨ SUCCESSFUL SALE ✨"}
        </p>

        {/* Sparkle effects */}
        <div className="absolute inset-0 overflow-hidden pointer-events-none">
          {[...Array(20)].map((_, i) => (
            <div
              key={i}
              className="absolute w-2 h-2 bg-yellow-300 rounded-full animate-sparkle"
              style={{
                left: `${Math.random() * 100}%`,
                top: `${Math.random() * 100}%`,
                animationDelay: `${Math.random() * 2}s`,
                animationDuration: `${1 + Math.random() * 2}s`
              }}
            />
          ))}
        </div>
      </div>

      <style jsx>{`
        @keyframes casino-entrance {
          0% {
            opacity: 0;
            transform: scale(0.3) rotate(-10deg);
          }
          50% {
            transform: scale(1.1) rotate(2deg);
          }
          100% {
            opacity: 1;
            transform: scale(1) rotate(-3deg);
          }
        }

        @keyframes bounce-subtle {
          0%, 100% {
            transform: translateY(0) rotate(-3deg) scale(1);
          }
          50% {
            transform: translateY(-10px) rotate(-3deg) scale(1.05);
          }
        }

        @keyframes fade-in-up {
          0% {
            opacity: 0;
            transform: translateY(20px);
          }
          100% {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes sparkle {
          0%, 100% {
            opacity: 0;
            transform: scale(0);
          }
          50% {
            opacity: 1;
            transform: scale(1);
          }
        }

        .animate-casino-entrance {
          animation: casino-entrance 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
        }

        .animate-bounce-subtle {
          animation: bounce-subtle 2s ease-in-out infinite;
        }

        .animate-fade-in-up {
          animation: fade-in-up 0.8s ease-out forwards;
          opacity: 0;
        }

        .animate-sparkle {
          animation: sparkle 2s ease-in-out infinite;
        }
      `}</style>
    </div>
  )
}
