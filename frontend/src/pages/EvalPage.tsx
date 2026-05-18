import { Link } from 'react-router-dom'
import { motion } from 'motion/react'
import { ArrowRight, AlertTriangle, Server } from 'lucide-react'
import { evalProducts } from '../data/evalProducts'

const SITE_URL = 'https://msdl.tech-latest.com'

const itemListJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'ItemList',
  name: 'Windows Server & Enterprise Evaluation ISOs',
  description: 'Official evaluation ISOs for Windows Server and Enterprise editions, direct from Microsoft.',
  itemListElement: evalProducts.map((p, i) => ({
    '@type': 'ListItem',
    position: i + 1,
    name: p.name,
    url: `${SITE_URL}/product/${p.slug}`,
  })),
}

const typeColors = {
  server: 'text-violet-400 bg-violet-500/10 border-violet-500/20',
  enterprise: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
} as const

const typeLabel = {
  server: 'Server',
  enterprise: 'Enterprise',
} as const

const container = {
  hidden: {},
  show: { transition: { staggerChildren: 0.06 } },
}

const cardVariant = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { type: 'spring' as const, stiffness: 280, damping: 28 } },
}

export default function EvalPage() {
  return (
    <>
      <title>Enterprise & Server ISOs | Windows ISO Downloader</title>
      <meta name="description" content="Download Windows Server 2025, 2022, 2019, 2016 and Windows 11 Enterprise evaluation ISOs directly from Microsoft's CDN." />
      <link rel="canonical" href={`${SITE_URL}/eval`} />
      <meta property="og:title" content="Enterprise & Server ISOs | Windows ISO Downloader" />
      <meta property="og:description" content="Download Windows Server and Enterprise evaluation ISOs directly from Microsoft's CDN." />
      <meta property="og:url" content={`${SITE_URL}/eval`} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />

      <div className="max-w-4xl mx-auto px-5 pt-12 pb-10">
        <motion.div
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
        >
          {/* Header */}
          <div className="mb-6">
            <p className="text-[11px] font-mono font-semibold tracking-[0.2em] uppercase text-violet-400 mb-3">
              Enterprise & Server
            </p>
            <h1 className="text-3xl font-bold text-white mb-2">Evaluation ISOs</h1>
            <p className="text-zinc-400 text-sm leading-relaxed max-w-lg">
              Official evaluation editions from Microsoft's evalcenter. Direct links from Microsoft's CDN — no session, no registration.
            </p>
          </div>

          {/* Eval notice */}
          <div className="flex items-start gap-3 p-4 rounded-xl border border-amber-500/15 bg-amber-500/6 mb-8">
            <AlertTriangle size={15} className="text-amber-400/80 mt-0.5 flex-shrink-0" />
            <p className="text-xs text-amber-400/70 leading-relaxed">
              <strong className="text-amber-400">Evaluation editions only.</strong> These ISOs are time-limited 180-day trials intended for testing and evaluation — not for production use. A valid license is required for activation beyond the trial period.
            </p>
          </div>

          {/* Product grid */}
          <motion.div
            className="grid grid-cols-1 sm:grid-cols-2 gap-3"
            variants={container}
            initial="hidden"
            animate="show"
          >
            {evalProducts.map(product => (
              <motion.div key={product.slug} variants={cardVariant}>
                <Link to={`/product/${product.slug}`} className="block outline-none group">
                  <motion.div
                    whileHover={{ y: -3, scale: 1.005 }}
                    transition={{ type: 'spring', stiffness: 400, damping: 25 }}
                    className="relative rounded-2xl border border-white/7 bg-[#111113] p-6
                      hover:border-white/13 hover:shadow-2xl hover:shadow-black/40
                      transition-colors duration-300"
                  >
                    {/* Header row */}
                    <div className="flex items-start justify-between mb-4">
                      <div className="w-9 h-9 rounded-lg bg-white/5 border border-white/8 flex items-center justify-center">
                        <Server className="w-4 h-4 text-white/50" />
                      </div>
                      <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full border ${typeColors[product.type]}`}>
                        {typeLabel[product.type]}
                      </span>
                    </div>

                    {/* Title */}
                    <h2 className="text-base font-semibold text-white leading-tight">{product.name}</h2>
                    <p className="text-[11px] font-mono text-zinc-500 mt-0.5 mb-3">{product.version}</p>
                    <p className="text-sm text-zinc-400 leading-relaxed mb-4 line-clamp-2">{product.description}</p>

                    {/* Footer */}
                    <div className="flex items-center justify-between">
                      <div className="flex gap-1.5">
                        {product.archs.map(arch => (
                          <span key={arch} className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded bg-white/5 border border-white/8 text-zinc-500">
                            {arch}
                          </span>
                        ))}
                        <span className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded bg-amber-500/8 border border-amber-500/20 text-amber-500/70">
                          EVAL
                        </span>
                      </div>
                      <div className="flex items-center gap-1 text-xs text-zinc-600 group-hover:text-zinc-400 transition-colors">
                        <span>Get links</span>
                        <ArrowRight size={11} className="group-hover:translate-x-0.5 transition-transform" />
                      </div>
                    </div>
                  </motion.div>
                </Link>
              </motion.div>
            ))}
          </motion.div>
        </motion.div>
      </div>
    </>
  )
}
