import { motion } from 'motion/react'
import { Download, ArrowRight } from 'lucide-react'
import Badge from './Badge'

const WinLogo = ({ className }: { className?: string }) => (
  <svg viewBox="0 0 88 88" className={className} fill="currentColor">
    <path d="M0 12.402l35.687-4.86.016 34.423-35.67.203zm35.67 33.529l.028 34.453L.028 75.48.026 45.7zm4.326-39.025L87.314 0v41.527l-47.318.376zm47.329 39.349-.011 41.34-47.318-6.678-.066-34.739z" />
  </svg>
)

type BadgeVariant = 'latest' | 'stable' | 'eol' | 'legacy'

interface ProductCardProps {
  id: string
  name: string
  version: string
  build: string
  description: string
  badge: BadgeVariant
  archs?: string[]
}

export default function ProductCard({
  name,
  version,
  build,
  description,
  badge,
  archs = ['x64'],
}: ProductCardProps) {
  return (
    <motion.div
      whileHover={{ y: -3, scale: 1.005 }}
      transition={{ type: 'spring', stiffness: 400, damping: 25 }}
      className="group relative cursor-pointer rounded-2xl border border-white/7 bg-[#111113] p-6
        hover:border-white/13 hover:shadow-2xl hover:shadow-black/40
        transition-colors duration-300"
    >
      {/* Header row */}
      <div className="flex items-start justify-between mb-4">
        <div className="w-9 h-9 rounded-lg bg-white/5 border border-white/8 flex items-center justify-center">
          <WinLogo className="w-5 h-5 text-white/60" />
        </div>
        <Badge variant={badge} />
      </div>

      {/* Title */}
      <div className="mb-3">
        <h2 className="text-base font-semibold text-white leading-tight">{name} {version}</h2>
        <p className="text-[11px] font-mono text-zinc-500 mt-0.5 tracking-wide">Build {build}</p>
      </div>

      {/* Description */}
      <p className="text-sm text-zinc-400 leading-relaxed mb-4">{description}</p>

      {/* Footer */}
      <div className="flex items-center justify-between">
        <div className="flex gap-1.5">
          {archs.map(arch => (
            <span
              key={arch}
              className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded bg-white/5 border border-white/8 text-zinc-500"
            >
              {arch}
            </span>
          ))}
        </div>
        <div className="flex items-center gap-1 text-xs text-zinc-600 group-hover:text-zinc-400 transition-colors">
          <Download size={12} />
          <span>Get links</span>
          <ArrowRight size={11} className="group-hover:translate-x-0.5 transition-transform" />
        </div>
      </div>
    </motion.div>
  )
}
