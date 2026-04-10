import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'motion/react'
import { Search, ArrowRight } from 'lucide-react'
import type { Product } from '../types'

const quickSearches = [
  { label: 'Windows 11 25H2', query: '25H2' },
  { label: 'Windows 11 24H2', query: '24H2' },
  { label: 'Windows 10', query: 'Windows 10' },
  { label: 'Windows 8.1', query: 'Windows 8.1' },
]

export default function ProductsPage() {
  const [products, setProducts] = useState<Product[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [query, setQuery] = useState('')
  useEffect(() => {
    document.title = "All Windows Releases | Windows ISO Downloader"
    const metaDesc = document.querySelector('meta[name="description"]')
    if (metaDesc) metaDesc.setAttribute('content', "Browse all official Microsoft Windows ISO releases available for direct download.")
    
    fetch('/data/products.json')
      .then(r => r.json())
      .then((data: Record<string, string>) => {
        setProducts(Object.entries(data).map(([id, name]) => ({ id, name })))
      })
      .finally(() => setIsLoading(false))
  }, [])

  const filtered = products.filter(p =>
    p.name.toLowerCase().includes(query.toLowerCase())
  )

  return (
    <div className="max-w-3xl mx-auto px-4 pt-12 pb-10">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <h1 className="text-3xl font-bold text-white mb-1">All Products</h1>
        <p className="text-white/40 text-sm mb-6">
          {products.length} versions available
        </p>

        {/* Search */}
        <div className="relative mb-4">
          <Search size={16} className="absolute left-3.5 top-1/2 -translate-y-1/2 text-white/30" />
          <input
            type="text"
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search Windows versions..."
            className="w-full bg-white/5 border border-white/8 rounded-xl pl-10 pr-4 py-3 text-sm text-white placeholder:text-white/25 focus:outline-none focus:border-white/20 focus:bg-white/8 transition-all"
          />
        </div>

        {/* Quick filters */}
        <div className="flex flex-wrap gap-2 mb-6">
          {quickSearches.map(s => (
            <button
              key={s.label}
              onClick={() => setQuery(s.query)}
              className="px-3 py-1.5 rounded-lg text-xs font-medium border border-white/8 bg-white/4 text-white/50 hover:text-white hover:border-white/15 hover:bg-white/8 transition-all"
            >
              {s.label}
            </button>
          ))}
          {query && (
            <button
              onClick={() => setQuery('')}
              className="px-3 py-1.5 rounded-lg text-xs font-medium border border-white/8 bg-white/4 text-white/40 hover:text-white transition-all"
            >
              Clear ×
            </button>
          )}
        </div>

        {/* List */}
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(8)].map((_, i) => (
              <div key={i} className="h-14 rounded-xl bg-white/4 animate-pulse" />
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <p className="text-white/30 text-sm text-center py-12">No products found.</p>
        ) : (
          <motion.div
            className="space-y-1.5"
            initial="hidden"
            animate="show"
            variants={{ show: { transition: { staggerChildren: 0.03 } } }}
          >
            {filtered.map(product => (
              <motion.div
                key={product.id}
                variants={{
                  hidden: { opacity: 0, x: -8 },
                  show: { opacity: 1, x: 0, transition: { type: 'spring', stiffness: 300, damping: 28 } },
                }}
              >
                <Link
                  to={`/product/${product.id}`}
                  className="group w-full flex items-center justify-between px-4 py-3.5 rounded-xl border border-white/6 bg-white/3 hover:bg-white/6 hover:border-white/12 transition-all text-left"
                >
                  <div className="flex items-center gap-3">
                    <svg viewBox="0 0 88 88" className="w-5 h-5 text-white/30 flex-shrink-0 group-hover:text-white/50 transition-colors" fill="currentColor">
                      <path d="M0 12.402l35.687-4.86.016 34.423-35.67.203zm35.67 33.529l.028 34.453L.028 75.48.026 45.7zm4.326-39.025L87.314 0v41.527l-47.318.376zm47.329 39.349-.011 41.34-47.318-6.678-.066-34.739z" />
                    </svg>
                    <span className="text-sm text-white/70 group-hover:text-white transition-colors font-medium">
                      {product.name}
                    </span>
                  </div>
                  <ArrowRight size={14} className="text-white/20 group-hover:text-white/50 group-hover:translate-x-0.5 transition-all" />
                </Link>
              </motion.div>
            ))}
          </motion.div>
        )}
      </motion.div>
    </div>
  )
}
