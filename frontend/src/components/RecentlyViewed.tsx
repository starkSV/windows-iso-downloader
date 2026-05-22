import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Clock, ArrowRight } from 'lucide-react'

export interface RecentEntry {
  id: string
  name: string
  badge?: string
  visitedAt: number
  linkExpiresAt?: number
}

const STORAGE_KEY = 'msdl-recent'
const MAX_ENTRIES = 4

export function addRecentEntry(entry: Omit<RecentEntry, 'visitedAt'>) {
  try {
    const existing = getRecentEntries()
    const filtered = existing.filter(e => e.id !== entry.id)
    const updated: RecentEntry[] = [
      { ...entry, visitedAt: Date.now() },
      ...filtered,
    ].slice(0, MAX_ENTRIES)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
  } catch {}
}

export function updateRecentExpiry(id: string, linkExpiresAt: number) {
  try {
    const existing = getRecentEntries()
    const updated = existing.map(e => e.id === id ? { ...e, linkExpiresAt } : e)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
  } catch {}
}

export function getRecentEntries(): RecentEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    return JSON.parse(raw) as RecentEntry[]
  } catch {
    return []
  }
}

function useCountdown(expiresAt?: number) {
  const [remaining, setRemaining] = useState<number | null>(null)

  useEffect(() => {
    if (!expiresAt) { setRemaining(null); return }
    const tick = () => setRemaining(expiresAt - Date.now())
    tick()
    const id = setInterval(tick, 60_000)
    return () => clearInterval(id)
  }, [expiresAt])

  return remaining
}

function formatRemaining(ms: number): string {
  if (ms <= 0) return 'Expired'
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function EntryCard({ entry }: { entry: RecentEntry }) {
  const remaining = useCountdown(entry.linkExpiresAt)
  const isExpired = remaining !== null && remaining <= 0
  const hasExpiry = remaining !== null

  return (
    <Link
      to={`/product/${entry.id}`}
      className="group flex items-center justify-between px-3.5 py-3 rounded-xl border border-white/6 bg-white/2 hover:border-white/10 hover:bg-white/4 transition-all"
    >
      <div className="min-w-0">
        <p className="text-sm text-white/80 group-hover:text-white transition-colors truncate">{entry.name}</p>
        {hasExpiry && (
          <p className={`text-[11px] mt-0.5 ${isExpired ? 'text-red-400/70' : 'text-zinc-600'}`}>
            {isExpired ? 'Link expired' : `Link expires in ${formatRemaining(remaining!)}`}
          </p>
        )}
      </div>
      <ArrowRight size={13} className="text-zinc-700 group-hover:text-zinc-400 flex-shrink-0 ml-2 transition-colors" />
    </Link>
  )
}

export default function RecentlyViewed() {
  const [entries, setEntries] = useState<RecentEntry[]>([])

  useEffect(() => {
    setEntries(getRecentEntries())
  }, [])

  if (entries.length === 0) return null

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <Clock size={12} className="text-zinc-600" />
        <span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600">
          Recently viewed
        </span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {entries.map(entry => (
          <EntryCard key={entry.id} entry={entry} />
        ))}
      </div>
    </div>
  )
}
