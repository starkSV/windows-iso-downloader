import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'motion/react'
import { ArrowLeft, Download, ExternalLink, AlertTriangle, Copy, Check, Server, ChevronDown } from 'lucide-react'
import { toast } from 'sonner'
import { evalProducts } from '../data/evalProducts'
import Aria2Tip from '../components/Aria2Tip'
import RelatedReleases from '../components/RelatedReleases'
import NotFoundPage from './NotFoundPage'

const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:3002'
const SITE_URL = 'https://msdl.tech-latest.com'

interface EvalLink {
  arch: string
  lang: string
  url: string
}

const langNames: Record<string, string> = {
  'en-us': 'English',
  'zh-cn': 'Chinese (Simplified)',
  'zh-tw': 'Chinese (Traditional)',
  'fr-fr': 'French',
  'de-de': 'German',
  'it-it': 'Italian',
  'ja-jp': 'Japanese',
  'ko-kr': 'Korean',
  'pt-br': 'Portuguese (Brazil)',
  'ru-ru': 'Russian',
  'es-es': 'Spanish',
  'pl-pl': 'Polish',
  'nl-nl': 'Dutch',
  'sv-se': 'Swedish',
  'tr-tr': 'Turkish',
}

function WinLogo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 88 88" className={className} fill="currentColor" aria-hidden="true">
      <path d="M0 12.402l35.687-4.86.016 34.423-35.67.203zm35.67 33.529l.028 34.453L.028 75.48.026 45.7zm4.326-39.025L87.314 0v41.527l-47.318.376zm47.329 39.349-.011 41.34-47.318-6.678-.066-34.739z" />
    </svg>
  )
}

export default function EvalDetailPage() {
  const { productId } = useParams<{ productId: string }>()
  const navigate = useNavigate()

  const product = evalProducts.find(p => p.slug === productId)

  const [links, setLinks] = useState<EvalLink[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null)
  const [moreOpen, setMoreOpen] = useState(false)

  useEffect(() => {
    if (!product) return
    setIsLoading(true)
    setError(null)
    fetch(`${API_BASE}/evallinks?product=${product.slug}`)
      .then(r => r.json())
      .then(data => {
        if (data.error) throw new Error(data.error)
        setLinks(data.links ?? [])
      })
      .catch(err => setError(err.message))
      .finally(() => setIsLoading(false))
  }, [product?.slug])

  async function handleCopy(url: string) {
    try {
      await navigator.clipboard.writeText(url)
      setCopiedUrl(url)
      toast.success('Link copied to clipboard')
      setTimeout(() => setCopiedUrl(null), 2000)
    } catch {
      toast.error('Failed to copy link')
    }
  }

  if (!product) return <NotFoundPage />

  const primaryLink = links[0] ?? null
  const otherLinks = links.slice(1)

  const pageTitle = `${product.name} Evaluation ISO Download | Windows ISO Downloader`
  const pageDesc = product.seoDesc
  const canonical = `${SITE_URL}/product/${product.slug}`

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: SITE_URL },
      { '@type': 'ListItem', position: 2, name: 'Enterprise & Server', item: `${SITE_URL}/eval` },
      { '@type': 'ListItem', position: 3, name: product.name, item: canonical },
    ],
  }

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: `${product.name} Evaluation`,
    operatingSystem: 'Windows',
    applicationCategory: 'OperatingSystem',
    url: canonical,
    description: product.seoDesc,
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  }

  return (
    <>
      <title>{pageTitle}</title>
      <meta name="description" content={pageDesc} />
      <link rel="canonical" href={canonical} />
      <meta property="og:title" content={pageTitle} />
      <meta property="og:description" content={pageDesc} />
      <meta property="og:url" content={canonical} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />

      <div className="max-w-xl mx-auto px-5 pt-12 pb-10">
        {/* Back */}
        <button
          onClick={() => window.history.state?.idx ? navigate(-1) : navigate('/eval')}
          className="flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-300 mb-8 transition-colors"
        >
          <ArrowLeft size={15} />
          Enterprise & Server
        </button>

        <motion.div
          initial={{ opacity: 0, y: 14 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35 }}
        >
          {/* Header */}
          <div className="flex items-start gap-4 mb-6">
            <div className="w-12 h-12 rounded-xl bg-white/5 border border-white/8 flex items-center justify-center flex-shrink-0">
              {product.type === 'server'
                ? <Server className="w-6 h-6 text-white/50" />
                : <WinLogo className="w-6 h-6 text-white/50" />
              }
            </div>
            <div className="min-w-0">
              <h1 className="text-2xl font-bold text-white leading-tight tracking-tight">
                {product.name}
              </h1>
              <p className="text-[12px] font-mono text-zinc-500 mt-0.5">{product.version}</p>
              <div className="flex gap-1.5 mt-2">
                {product.archs.map(arch => (
                  <span key={arch} className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded bg-white/5 border border-white/8 text-zinc-500">
                    {arch}
                  </span>
                ))}
                <span className="text-[10px] font-mono font-semibold px-2 py-0.5 rounded-full bg-amber-500/10 border border-amber-500/20 text-amber-400">
                  EVAL
                </span>
              </div>
            </div>
          </div>

          {/* Eval notice */}
          <div className="flex items-start gap-2.5 p-3.5 rounded-xl border border-amber-500/15 bg-amber-500/6 mb-5">
            <AlertTriangle size={14} className="text-amber-400/80 mt-0.5 flex-shrink-0" />
            <p className="text-[11px] text-amber-400/70 leading-relaxed">
              <strong className="text-amber-400">180-day evaluation only.</strong> Not for production use. A valid license is required for activation beyond the trial period.
            </p>
          </div>

          {/* Download card */}
          <div className="rounded-2xl border border-white/7 bg-[#111113] p-6 mb-4">
            <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600 mb-4">
              Download
            </p>

            {isLoading ? (
              <div className="space-y-2.5">
                <div className="h-11 bg-white/5 rounded-xl animate-pulse" />
                <div className="h-8 w-36 bg-white/4 rounded-lg animate-pulse" />
              </div>
            ) : error ? (
              <div className="py-4 text-center">
                <p className="text-sm text-red-400 mb-1">Failed to load download links</p>
                <p className="text-xs text-zinc-600">{error}</p>
              </div>
            ) : primaryLink ? (
              <div className="space-y-3">
                {/* Primary link */}
                <div className="flex items-center gap-2">
                  <a
                    href={primaryLink.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex-1 flex items-center gap-2 px-4 py-3 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-400 hover:bg-blue-500/15 hover:border-blue-500/30 transition-all text-sm font-medium"
                  >
                    <Download size={14} />
                    Download {primaryLink.arch}
                    {primaryLink.lang && (
                      <span className="text-blue-400/60 font-normal">
                        · {langNames[primaryLink.lang] ?? primaryLink.lang}
                      </span>
                    )}
                    <ExternalLink size={12} className="ml-auto opacity-60" />
                  </a>
                  <button
                    onClick={() => handleCopy(primaryLink.url)}
                    className={`p-3 rounded-xl border transition-all flex-shrink-0 ${
                      copiedUrl === primaryLink.url
                        ? 'border-green-500/30 bg-green-500/10 text-green-400'
                        : 'border-white/8 bg-white/4 text-zinc-500 hover:text-white hover:border-white/15'
                    }`}
                    title="Copy download link"
                  >
                    {copiedUrl === primaryLink.url ? <Check size={14} /> : <Copy size={14} />}
                  </button>
                </div>

                {/* More languages */}
                {otherLinks.length > 0 && (
                  <div>
                    <button
                      onClick={() => setMoreOpen(o => !o)}
                      className="flex items-center gap-1.5 text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
                    >
                      <ChevronDown
                        size={12}
                        className={`transition-transform duration-200 ${moreOpen ? 'rotate-180' : ''}`}
                      />
                      {otherLinks.length} more language{otherLinks.length > 1 ? 's' : ''}
                    </button>

                    <AnimatePresence>
                      {moreOpen && (
                        <motion.div
                          initial={{ height: 0, opacity: 0 }}
                          animate={{ height: 'auto', opacity: 1 }}
                          exit={{ height: 0, opacity: 0 }}
                          transition={{ duration: 0.2 }}
                          className="overflow-hidden"
                        >
                          <div className="space-y-1.5 mt-2.5">
                            {otherLinks.map(link => (
                              <div key={link.url} className="flex items-center gap-2">
                                <a
                                  href={link.url}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="flex-1 flex items-center gap-2 px-3 py-2 rounded-lg bg-white/4 border border-white/6 text-zinc-400 hover:bg-white/7 hover:border-white/10 hover:text-white transition-all text-xs"
                                >
                                  <Download size={11} />
                                  {link.arch}
                                  {link.lang && (
                                    <span className="text-zinc-500">
                                      · {langNames[link.lang] ?? link.lang}
                                    </span>
                                  )}
                                  <ExternalLink size={10} className="ml-auto opacity-50" />
                                </a>
                                <button
                                  onClick={() => handleCopy(link.url)}
                                  className={`px-2.5 py-2 rounded-lg border text-[10px] font-mono transition-all flex-shrink-0 ${
                                    copiedUrl === link.url
                                      ? 'border-green-500/30 bg-green-500/10 text-green-400'
                                      : 'border-white/8 bg-white/4 text-zinc-500 hover:text-white hover:border-white/15'
                                  }`}
                                >
                                  {copiedUrl === link.url ? '✓' : 'Copy'}
                                </button>
                              </div>
                            ))}
                          </div>
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>
                )}

                <p className="text-[10px] text-zinc-600 pt-1">
                  Direct from Microsoft's CDN · Evaluation only — requires activation after 180 days
                </p>
              </div>
            ) : (
              <p className="text-sm text-zinc-500 text-center py-4">No download links available.</p>
            )}
          </div>

          {/* System requirements */}
          <div className="rounded-xl border border-white/7 bg-[#111113] p-4 mb-4">
            <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600 mb-3">
              System Requirements
            </p>
            <div className="grid grid-cols-3 gap-3">
              <div className="flex flex-col gap-1">
                <span className="text-[11px] text-zinc-600">Processor</span>
                <span className="text-sm font-semibold text-white">{product.requirements.cpu}</span>
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-[11px] text-zinc-600">RAM</span>
                <span className="text-sm font-semibold text-white">{product.requirements.ram}</span>
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-[11px] text-zinc-600">Storage</span>
                <span className="text-sm font-semibold text-white">{product.requirements.disk}</span>
              </div>
            </div>
            {product.requirements.note && (
              <div className="mt-3 pt-3 border-t border-white/5 flex items-center gap-2">
                <AlertTriangle size={12} className="text-amber-500 flex-shrink-0" />
                <span className="text-[11px] text-amber-500/80">{product.requirements.note}</span>
              </div>
            )}
          </div>

          {/* About */}
          <div className="rounded-xl border border-white/7 bg-[#111113] p-4">
            <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600 mb-2">
              About this release
            </p>
            <p className="text-sm text-zinc-400 leading-relaxed">{product.description}</p>
          </div>

          {/* Aria2 tip — outside box, same as consumer page */}
          <Aria2Tip />

          {/* Also available — outside box, same as consumer page */}
          <RelatedReleases
            current={product.slug}
            items={evalProducts.map(p => ({ id: p.slug, label: p.name }))}
          />
        </motion.div>
      </div>
    </>
  )
}
