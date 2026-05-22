import { motion } from 'motion/react'
import { useState, useEffect } from 'react'

interface Stat {
  value: string
  label: string
}

export default function StatsBar() {
  const [totalReleases, setTotalReleases] = useState('17')

  useEffect(() => {
    fetch('/data/products.json')
      .then(r => r.json())
      .then((data: Record<string, { active?: boolean }>) => {
        const activeCount = Object.values(data).filter(p => p.active !== false).length
        setTotalReleases(activeCount.toString())
      })
      .catch(() => {})
  }, [])

  const stats: Stat[] = [
    { value: totalReleases, label: 'releases available' },
    { value: '38', label: 'languages' },
    { value: '0', label: 'intermediaries' },
  ]

  return (
    <motion.div
      className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1.5 text-[11px] font-mono text-zinc-600"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ delay: 0.35, duration: 0.5 }}
    >
      {stats.map((s, i) => (
        <span key={s.label} className="flex items-center gap-4">
          <span>
            <span className="text-zinc-400 font-semibold">{s.value}</span>
            {' '}{s.label}
          </span>
          {i < stats.length - 1 && <span className="text-zinc-700">·</span>}
        </span>
      ))}
    </motion.div>
  )
}
