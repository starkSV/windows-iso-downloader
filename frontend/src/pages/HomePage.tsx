import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'motion/react'
import { ArrowRight, Server, Terminal } from 'lucide-react'
import ProductCard from '../components/ProductCard'
import StatsBar from '../components/StatsBar'
import HowItWorks from '../components/HowItWorks'
import ComparisonTable from '../components/ComparisonTable'
import FAQAccordion from '../components/FAQAccordion'
import RecentlyViewed from '../components/RecentlyViewed'

const SITE_URL = 'https://msdl.tech-latest.com'

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

const jsonLd = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'MSDL — Windows ISO Downloader',
  applicationCategory: 'UtilitiesApplication',
  operatingSystem: 'Web',
  url: SITE_URL,
  description: 'Download official Microsoft Windows ISO files for Windows 11, 10, 8.1, and Windows Server. Direct links from Microsoft CDN. Free, no registration.',
  offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
}

export default function HomePage() {
  const [totalReleases, setTotalReleases] = useState(16)

  useEffect(() => {
    fetch('/data/products.json')
      .then(r => r.json())
      .then((data: Record<string, { active?: boolean }>) => {
        const activeCount = Object.values(data).filter(p => p.active !== false).length
        setTotalReleases(activeCount)
      })
      .catch(() => {})
  }, [])

  return (
    <>
      <title>Windows ISO Downloader | Official Microsoft Images</title>
      <meta name="description" content="Download official Microsoft Windows ISO files for Windows 11, 10, and 8.1. Fast, free, no registration required." />
      <link rel="canonical" href={`${SITE_URL}/`} />
      <meta property="og:title" content="Windows ISO Downloader | Official Microsoft Images" />
      <meta property="og:description" content="Download official Microsoft Windows ISO files. Fast, free, no registration." />
      <meta property="og:url" content={`${SITE_URL}/`} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

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
          <div className="flex items-center justify-center flex-wrap gap-x-4 gap-y-1.5 text-[12px] text-zinc-500">
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
            <span className="text-zinc-700">·</span>
            <a
              href="https://github.com/starkSV/windows-iso-downloader"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 hover:text-zinc-300 transition-colors"
            >
              <span className="w-1.5 h-1.5 rounded-full bg-purple-500 inline-block" />
              <span className="underline decoration-dotted decoration-zinc-600 underline-offset-4">Open source</span>
            </a>
          </div>
        </motion.div>

        {/* Stats bar */}
        <div className="mb-10">
          <StatsBar />
        </div>

        {/* Recently viewed */}
        <RecentlyViewed />

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

        {/* Enterprise & Server CTA */}
        <motion.div
          className="mb-4"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.45 }}
        >
          <Link
            to="/eval"
            className="group flex items-center justify-between w-full px-5 py-3.5 rounded-xl border border-violet-500/15 bg-violet-500/5 hover:border-violet-500/25 hover:bg-violet-500/8 transition-all duration-200"
          >
            <div className="flex items-center gap-3">
              <div className="w-7 h-7 rounded-lg bg-violet-500/10 border border-violet-500/20 flex items-center justify-center flex-shrink-0">
                <Server size={14} className="text-violet-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-white/80 group-hover:text-white transition-colors leading-tight">
                  Enterprise & Server ISOs
                </p>
                <p className="text-[11px] text-zinc-600 mt-0.5">
                  Server 2025, 2022, 2019, 2016 · Win 11 Enterprise · Eval editions
                </p>
              </div>
            </div>
            <ArrowRight size={14} className="text-zinc-600 group-hover:text-violet-400 group-hover:translate-x-0.5 transition-all" />
          </Link>
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

        {/* CLI promo */}
        <motion.div
          className="mt-6"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.55 }}
        >
          <Link
            to="/cli"
            className="group block px-4 py-3 rounded-xl border border-white/6 bg-[#0d0d0f] hover:border-emerald-500/20 hover:bg-[#0a0f0b] transition-all duration-200"
          >
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-1.5">
                <Terminal size={11} className="text-emerald-500/50" />
                <span className="text-[10px] font-mono text-zinc-600 uppercase tracking-widest">CLI tool</span>
              </div>
              <span className="flex items-center gap-1 text-[11px] text-zinc-600 group-hover:text-emerald-400 transition-colors">
                Learn more
                <ArrowRight size={10} className="group-hover:translate-x-0.5 transition-transform" />
              </span>
            </div>
            <code className="text-[13px] font-mono text-zinc-300">
              <span className="text-zinc-600">$ </span>msdl --id 3262 --lang &quot;English&quot;
            </code>
          </Link>
        </motion.div>

        {/* How it works */}
        <HowItWorks />

        {/* Comparison table */}
        <ComparisonTable />

        {/* FAQ */}
        <FAQAccordion />
      </div>
    </>
  )
}
