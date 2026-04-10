import { useNavigate } from 'react-router-dom'
import { ArrowRight } from 'lucide-react'

interface RelatedProduct {
  id: string
  label: string
}

interface RelatedReleasesProps {
  current: string
  items: RelatedProduct[]
}

export default function RelatedReleases({ current, items }: RelatedReleasesProps) {
  const navigate = useNavigate()
  const filtered = items.filter(i => i.id !== current)
  if (filtered.length === 0) return null

  return (
    <div className="mt-6 pt-5 border-t border-white/5">
      <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600 mb-3">
        Also available
      </p>
      <div className="flex flex-wrap gap-2">
        {filtered.map(item => (
          <button
            key={item.id}
            onClick={() => navigate(`/product/${item.id}`)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-white/7 bg-white/3 text-xs text-zinc-400 hover:text-white hover:border-white/13 hover:bg-white/6 transition-all"
          >
            {item.label}
            <ArrowRight size={11} />
          </button>
        ))}
      </div>
    </div>
  )
}
