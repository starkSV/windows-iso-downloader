import { useState, useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { motion } from 'motion/react'
import { Home, LayoutGrid, Info, ExternalLink, Shield } from 'lucide-react'

interface DockItem {
  label: string
  icon: React.ReactNode
  path?: string
  href?: string
}

const items: DockItem[] = [
  { label: 'Home',       icon: <Home size={19} />,        path: '/' },
  { label: 'Products',   icon: <LayoutGrid size={19} />,  path: '/products' },
  { label: 'About',      icon: <Info size={19} />,        path: '/about' },
  { label: 'Privacy',    icon: <Shield size={19} />,      path: '/privacy-policy' },
  { label: 'TechLatest', icon: <ExternalLink size={19} />, href: 'https://tech-latest.com' },
]

export default function Dock() {
  const location = useLocation()
  const navigate = useNavigate()
  const [activeIndex, setActiveIndex] = useState(0)

  useEffect(() => {
    const idx = items.findIndex(item => item.path && item.path === location.pathname)
    if (idx !== -1) setActiveIndex(idx)
  }, [location.pathname])

  function handleClick(item: DockItem, index: number) {
    setActiveIndex(index)
    if (item.href) {
      window.open(item.href, '_blank', 'noopener noreferrer')
    } else if (item.path) {
      navigate(item.path)
    }
  }

  return (
    <>
      {/* ── DESKTOP DOCK (sm+) — centered floating pill ── */}
      <div className="hidden sm:flex fixed bottom-5 left-0 right-0 justify-center z-50 pointer-events-none">
        <motion.div
          className="pointer-events-auto flex items-center gap-1 px-3 py-2 rounded-2xl border border-white/10 bg-[#111113]/90 backdrop-blur-xl shadow-2xl shadow-black/50"
          initial={{ y: 16, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ type: 'spring', stiffness: 300, damping: 30 }}
        >
          {items.map((item, i) => {
            const isActive = i === activeIndex
            return (
              <motion.button
                key={item.label}
                onClick={() => handleClick(item, i)}
                whileHover={{ scale: 1.06, y: -2 }}
                whileTap={{ scale: 0.93 }}
                transition={{ type: 'spring', stiffness: 400, damping: 22 }}
                className={`
                  relative flex items-center gap-2 rounded-xl cursor-pointer px-3 py-2
                  transition-all duration-200
                  ${isActive
                    ? 'bg-white/10 border border-white/15 text-white'
                    : 'border border-transparent text-zinc-500 hover:text-zinc-300'
                  }
                `}
                aria-label={item.label}
              >
                {item.icon}

                {/* Expanding active label */}
                <motion.span
                  animate={{ width: isActive ? 'auto' : 0, opacity: isActive ? 1 : 0 }}
                  transition={{ type: 'spring', stiffness: 400, damping: 32 }}
                  className="overflow-hidden whitespace-nowrap text-sm font-medium"
                >
                  {item.label}
                </motion.span>

                {/* Active dot */}
                {isActive && (
                  <motion.span
                    layoutId="dock-dot-desktop"
                    className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-1 h-1 rounded-full bg-blue-400"
                  />
                )}
              </motion.button>
            )
          })}
        </motion.div>
      </div>

      {/* ── MOBILE DOCK (<sm) — full-width bottom bar, flush, no gap ── */}
      <motion.div
        className="flex sm:hidden fixed bottom-0 left-0 right-0 z-50
          border-t border-white/8 bg-[#111113]/95 backdrop-blur-xl"
        initial={{ y: 20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ type: 'spring', stiffness: 300, damping: 30 }}
      >
        {items.map((item, i) => {
          const isActive = i === activeIndex
          return (
            <button
              key={item.label}
              onClick={() => handleClick(item, i)}
              className="flex-1 flex flex-col items-center justify-center gap-1 py-3 min-w-0"
            >
              <span className={`transition-colors ${isActive ? 'text-blue-400' : 'text-zinc-500'}`}>
                {item.icon}
              </span>
              <span
                className={`text-[9px] font-medium tracking-wide transition-colors truncate max-w-full px-1 ${
                  isActive ? 'text-blue-400' : 'text-zinc-600'
                }`}
              >
                {item.label}
              </span>
            </button>
          )
        })}
      </motion.div>
    </>
  )
}
