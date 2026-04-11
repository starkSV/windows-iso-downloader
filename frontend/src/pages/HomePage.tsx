import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'motion/react'
import { ArrowRight } from 'lucide-react'
import ProductCard from '../components/ProductCard'
import StatsBar from '../components/StatsBar'
import HowItWorks from '../components/HowItWorks'
import ComparisonTable from '../components/ComparisonTable'
import FAQAccordion from '../components/FAQAccordion'

const featured = [
  {
    id: '3262',
    name: 'Windows 11',
    version: '25H2',
    build: '26200.6584',
    description: 'The latest Windows 11 release with AI features and improved performance.',
    badge: 'latest' as const,
    archs: ['x64', 'ARM64'],
  },
  {
    id: '3113',
    name: 'Windows 11',
    version: '24H2',
    build: '26100.1742',
    description: 'The widely deployed stable release. Recommended for enterprise environments.',
    badge: 'stable' as const,
    archs: ['x64', 'ARM64'],
  },
  {
    id: '2618',
    name: 'Windows 10',
    version: '22H2',
    build: '19045.2965',
    description: 'The final Windows 10 feature update. Security support until October 2025.',
    badge: 'eol' as const,
    archs: ['x64', 'x86'],
  },
  {
    id: '52',
    name: 'Windows 8.1',
    version: 'RTM',
    build: '9600.17415',
    description: 'Legacy release for older hardware compatibility and historical reference.',
    badge: 'legacy' as const,
    archs: ['x64', 'x86'],
  },
]

const container = {
  hidden: {},
  show: { transition: { staggerChildren: 0.08 } },
}

const cardVariant = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { type: 'spring' as const, stiffness: 280, damping: 28 } },
}

export default function HomePage() {
  const [totalReleases, setTotalReleases] = useState(16)

  useEffect(() => {
    document.title = "Windows ISO Downloader | Official Microsoft Images"
    const metaDesc = document.querySelector('meta[name="description"]')
    if (metaDesc) {
      metaDesc.setAttribute('content', "Download official Microsoft Windows ISO files. Fast, free, no registration required.")
    }
    
    fetch('/data/products.json')
      .then(r => r.json())
      .then(data => setTotalReleases(Object.keys(data).length))
      .catch(() => {})
  }, [])

  return (
    <div className="max-w-4xl mx-auto px-5 pt-20 pb-16 relative" style={{ overflowX: 'hidden' }}>

      {/* Ambient glow — radial-gradient used over blur-3xl to optimize Mobile Lighthouse (Speed Index bypass) */}
      <div 
        className="fixed top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] pointer-events-none -z-10" 
        style={{ background: 'radial-gradient(ellipse at center, rgba(59,130,246,0.10) 0%, transparent 60%)' }}
      />

      {/* Hero */}
      <motion.div
        className="text-center mb-6"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
      >
        <p className="text-[11px] font-mono font-semibold tracking-[0.22em] uppercase text-blue-400 mb-5">
          Official Microsoft ISOs
        </p>
        <h1
          className="font-bold text-white tracking-tight leading-tight mb-4"
          style={{ fontSize: 'clamp(1.6rem, 7.5vw, 3.75rem)' }}
        >
          Windows ISO <span className="text-white/50">Downloader</span>
        </h1>
        <p className="text-zinc-400 text-base max-w-sm mx-auto mb-6 leading-relaxed">
          Direct links from Microsoft's CDN. No ads. No registration.
        </p>
        <div className="flex items-center justify-center gap-4 text-[12px] text-zinc-500">
          <span className="flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500 inline-block" />
            Always official
          </span>
          <span className="text-zinc-700">·</span>
          <span className="flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-500 inline-block" />
            Real-time links
          </span>
          <span className="text-zinc-700">·</span>
          <span className="flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-zinc-600 inline-block" />
            Free forever
          </span>
        </div>
      </motion.div>

      {/* Stats bar */}
      <div className="mb-10">
        <StatsBar />
      </div>

      {/* Featured grid */}
      <motion.div
        className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-8"
        variants={container}
        initial="hidden"
        animate="show"
      >
        {featured.map(product => (
          <motion.div key={product.id} variants={cardVariant}>
            <Link to={`/product/${product.id}`} className="block outline-none">
              <ProductCard
                {...product}
              />
            </Link>
          </motion.div>
        ))}
      </motion.div>

      {/* Browse all CTA */}
      <motion.div
        className="text-center"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.5 }}
      >
        <Link
          to="/products"
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl border border-white/8 bg-white/4 text-sm text-zinc-400 hover:text-white hover:border-white/15 hover:bg-white/7 transition-all duration-200"
        >
          Browse all {totalReleases} releases
          <ArrowRight size={14} />
        </Link>
      </motion.div>

      {/* How it works */}
      <HowItWorks />

      {/* Comparison table */}
      <ComparisonTable />

      {/* FAQ */}
      <FAQAccordion />
    </div>
  )
}
