import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { motion, AnimatePresence } from 'motion/react'
import * as Select from '@radix-ui/react-select'
import { ArrowLeft, ChevronDown, Download, AlertTriangle, Check, ExternalLink, WifiOff, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import type { Sku, DownloadOption } from '../types'
import SystemRequirements from '../components/SystemRequirements'
import CliCommand from '../components/CliCommand'
import RelatedReleases from '../components/RelatedReleases'
import OfficialFallback from '../components/OfficialFallback'
import CliHandoff from '../components/CliHandoff'
import { addRecentEntry, updateRecentExpiry } from '../components/RecentlyViewed'

const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:3002'
const SITE_URL = 'https://msdl.tech-latest.com'

interface ProductCatalogEntry {
  name: string
  badge: string
  archs: string[]
  related: string[]
  active?: boolean
}

function WinLogo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 88 88" className={className} fill="currentColor" aria-hidden="true">
      <path d="M0 12.402l35.687-4.86.016 34.423-35.67.203zm35.67 33.529l.028 34.453L.028 75.48.026 45.7zm4.326-39.025L87.314 0v41.527l-47.318.376zm47.329 39.349-.011 41.34-47.318-6.678-.066-34.739z" />
    </svg>
  )
}

function archFromUri(uri: string): string {
  const name = uri.split('/').pop()?.split('?')[0]?.toLowerCase() ?? ''
  if (name.includes('arm')) return 'ARM64'
  if (name.includes('x64') || name.includes('64')) return 'x64'
  if (name.includes('x32') || name.includes('32') || name.includes('x86')) return 'x86'
  return 'ISO'
}

function isWin11(productId: string): boolean {
  const id = Number(productId)
  return id >= 3113
}

export default function ProductDetailPage() {
  const { productId } = useParams<{ productId: string }>()
  const navigate = useNavigate()

  const [productName, setProductName] = useState('')
  const [buildStr, setBuildStr] = useState('')
  const [languages, setLanguages] = useState<Sku[]>([])
  const [selectedSku, setSelectedSku] = useState<Sku | null>(null)
  const [downloadLinks, setDownloadLinks] = useState<DownloadOption[]>([])
  const [isLoadingLangs, setIsLoadingLangs] = useState(true)
  const [isFetching, setIsFetching] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copiedUri, setCopiedUri] = useState<string | null>(null)
  const [isNotFound, setIsNotFound] = useState(false)
  const [isValidated, setIsValidated] = useState(false)
  const [hasCatalogError, setHasCatalogError] = useState(false)
  const [, setLinkExpiry] = useState<Date | null>(null)
  const [expiryRemaining, setExpiryRemaining] = useState<number | null>(null)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [linkStatus, setLinkStatus] = useState<'fresh' | 'cached' | 'stale' | null>(null)
  const [isLockdown, setIsLockdown] = useState(false)
  const expiryTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const [meta, setMeta] = useState<{ badge: string; archs: string[]; active: boolean }>({ badge: '', archs: [], active: false })
  const [related, setRelated] = useState<{ id: string; label: string }[]>([])

  // Derived SEO values
  const pageTitle = isNotFound
    ? 'Product Not Found | Windows ISO Downloader'
    : hasCatalogError
    ? 'Error | Windows ISO Downloader'
    : productName
    ? `${productName}${!meta.active ? ' (Discontinued)' : ''} ISO Download | Windows ISO Downloader`
    : 'Loading... | Windows ISO Downloader'

  const pageDescription = isNotFound
    ? 'The requested product could not be found.'
    : hasCatalogError
    ? 'Failed to load product catalog.'
    : productName
    ? `Download the official ${productName} ISO directly from Microsoft's CDN. No registration, no proxy — real-time signed links.`
    : ''

  const canonicalUrl = `${SITE_URL}/product/${productId}`

  const breadcrumbJsonLd = productName && !isNotFound && !hasCatalogError ? {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: SITE_URL },
      { '@type': 'ListItem', position: 2, name: 'All Releases', item: `${SITE_URL}/products` },
      { '@type': 'ListItem', position: 3, name: productName, item: canonicalUrl },
    ],
  } : null

  const softwareJsonLd = productName && !isNotFound && !hasCatalogError ? {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: productName,
    operatingSystem: 'Windows',
    applicationCategory: 'OperatingSystem',
    url: canonicalUrl,
    description: pageDescription,
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  } : null

  // Load product name + build string
  useEffect(() => {
    setBuildStr('')
    setIsNotFound(false)
    setIsValidated(false)
    setHasCatalogError(false)
    setProductName('')
    setMeta({ badge: '', archs: [], active: false })
    setRelated([])

    if (productId && !/^\d+$/.test(productId)) {
      setIsNotFound(true)
      setProductName('Invalid Product ID')
      return
    }

    fetch('/data/products.json')
      .then(r => {
        if (!r.ok) throw new Error('Failed to fetch catalog')
        return r.json()
      })
      .then((data: Record<string, ProductCatalogEntry>) => {
        const product = data[productId!]

        if (!product || !product.name) {
          setIsNotFound(true)
          setProductName('Product Not Found')
          return
        }

        const name = product.name
        setProductName(name)
        const isActive = product.active !== false
        setMeta({ badge: product.badge || '', archs: product.archs || [], active: isActive })

        if (isActive) {
          addRecentEntry({ id: productId!, name, badge: product.badge })
        }

        setRelated((product.related || []).map(rId => ({
          id: rId,
          label: data[rId]?.name || `Product ${rId}`
        })))

        if (isActive) {
          setIsValidated(true)
        }

        // Extract build from parentheses e.g. "(26200.6584)"
        const match = name.match(/\(([^)]+)\)/)
        if (match) setBuildStr(match[1])
      })
      .catch(() => {
        setHasCatalogError(true)
        setProductName('Error Loading Catalog')
      })
  }, [productId])

  // Fetch languages
  useEffect(() => {
    if (!productId || !isValidated || !meta.active) return
    setIsLoadingLangs(true)
    setError(null)
    setDownloadLinks([])
    setSelectedSku(null)

    fetch(`${API_BASE}/skuinfo?product_id=${productId}`)
      .then(r => r.json())
      .then(data => {
        if (data.error) throw new Error(data.error)
        if (!data.Skus?.length) throw new Error('No languages found for this product.')
        setLanguages(data.Skus)
        const english = data.Skus.find((s: Sku) =>
          s.LocalizedLanguage.toLowerCase().includes('english') &&
          !s.LocalizedLanguage.toLowerCase().includes('international')
        ) ?? data.Skus[0]
        setSelectedSku(english)
      })
      .catch(e => setError(e.message))
      .finally(() => setIsLoadingLangs(false))
  }, [productId, isValidated, meta.active])

  async function handleGetLinks() {
    if (!selectedSku || !productId) return
    setIsFetching(true)
    setError(null)
    setDownloadLinks([])
    setLinkExpiry(null)
    setExpiryRemaining(null)
    setLinkStatus(null)
    setIsLockdown(false)
    if (expiryTimerRef.current) clearInterval(expiryTimerRef.current)

    try {
      const r = await fetch(`${API_BASE}/proxy?product_id=${productId}&sku_id=${selectedSku.Id}`)
      const status = r.headers.get('X-MSDL-Link-Status') as 'fresh' | 'cached' | 'stale' | null
      const expiresHeader = r.headers.get('X-MSDL-Link-Expires')
      const lockdownHeader = r.headers.get('X-MSDL-Lockdown')
      const data = await r.json()
      if (data.error) {
        if (data.lockdown) setIsLockdown(true)
        throw new Error(data.error)
      }
      if (!data.ProductDownloadOptions?.length) throw new Error('No download links returned.')
      setDownloadLinks(data.ProductDownloadOptions)
      setLinkStatus(status)
      if (lockdownHeader === '1') setIsLockdown(true)
      let expiry: Date | null = null
      if (expiresHeader) {
        const d = new Date(expiresHeader)
        if (!isNaN(d.getTime())) expiry = d
      }
      if (!expiry) expiry = parseSeExpiry(data.ProductDownloadOptions[0].Uri)
      setLinkExpiry(expiry)
      if (expiry) {
        startExpiryCountdown(expiry)
        if (productId) updateRecentExpiry(productId, expiry.getTime())
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch links'
      setError(msg)
      toast.error(msg)
    } finally {
      setIsFetching(false)
    }
  }

  function handleCopy(uri: string) {
    navigator.clipboard.writeText(uri)
    setCopiedUri(uri)
    toast.success('Link copied to clipboard!')
    setTimeout(() => setCopiedUri(null), 2000)
  }

  function parseSeExpiry(uri: string): Date | null {
    try {
      const se = new URL(uri).searchParams.get('se')
      if (!se) return null
      const d = new Date(se)
      return isNaN(d.getTime()) ? null : d
    } catch {
      return null
    }
  }

  function formatRemaining(ms: number): string {
    if (ms <= 0) return 'Expired'
    const h = Math.floor(ms / 3_600_000)
    const m = Math.floor((ms % 3_600_000) / 60_000)
    if (h > 0) return `${h}h ${m}m`
    return `${m}m`
  }

  function startExpiryCountdown(expiry: Date) {
    if (expiryTimerRef.current) clearInterval(expiryTimerRef.current)
    const tick = () => setExpiryRemaining(expiry.getTime() - Date.now())
    tick()
    expiryTimerRef.current = setInterval(tick, 10_000)
  }

  async function handleRefreshLinks() {
    if (!selectedSku || !productId) return
    setIsRefreshing(true)
    setError(null)
    try {
      const r = await fetch(`${API_BASE}/proxy?product_id=${productId}&sku_id=${selectedSku.Id}&force=true`)
      const status = r.headers.get('X-MSDL-Link-Status') as 'fresh' | 'cached' | 'stale' | null
      const expiresHeader = r.headers.get('X-MSDL-Link-Expires')
      const lockdownHeader = r.headers.get('X-MSDL-Lockdown')
      const data = await r.json()
      if (data.error) {
        if (data.lockdown) setIsLockdown(true)
        throw new Error(data.error)
      }
      if (!data.ProductDownloadOptions?.length) throw new Error('No download links returned.')
      setDownloadLinks(data.ProductDownloadOptions)
      setLinkStatus(status)
      if (lockdownHeader === '1') setIsLockdown(true)
      else setIsLockdown(false)
      let expiry: Date | null = null
      if (expiresHeader) {
        const d = new Date(expiresHeader)
        if (!isNaN(d.getTime())) expiry = d
      }
      if (!expiry) expiry = parseSeExpiry(data.ProductDownloadOptions[0].Uri)
      setLinkExpiry(expiry)
      if (expiry) {
        startExpiryCountdown(expiry)
        if (productId) updateRecentExpiry(productId, expiry.getTime())
      }
      toast.success('Links refreshed!')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Refresh failed'
      setError(msg)
      toast.error(msg)
    } finally {
      setIsRefreshing(false)
    }
  }

  // Cleanup expiry timer on unmount
  useEffect(() => {
    return () => { if (expiryTimerRef.current) clearInterval(expiryTimerRef.current) }
  }, [])

  const firstUri = downloadLinks[0]?.Uri
  const firstFilename = firstUri ? (firstUri.split('/').pop()?.split('?')[0] ?? 'windows.iso') : undefined
  const showRefresh = expiryRemaining !== null && expiryRemaining < 6 * 3_600_000

  return (
    <>
      <title>{pageTitle}</title>
      {pageDescription && <meta name="description" content={pageDescription} />}
      <link rel="canonical" href={canonicalUrl} />
      <meta property="og:title" content={pageTitle} />
      {pageDescription && <meta property="og:description" content={pageDescription} />}
      <meta property="og:url" content={canonicalUrl} />
      {isNotFound && <meta name="robots" content="noindex" />}
      {breadcrumbJsonLd && (
        <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      )}
      {softwareJsonLd && (
        <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      )}

      <div className="max-w-2xl mx-auto px-5 pt-12 pb-10">
        {/* Back */}
        <button
          onClick={() => window.history.state?.idx ? navigate(-1) : navigate('/')}
          className="flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-300 mb-8 transition-colors"
        >
          <ArrowLeft size={15} />
          Back
        </button>

        <motion.div
          initial={{ opacity: 0, y: 14 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35 }}
        >
          {/* Product header */}
          <div className="flex items-start gap-4 mb-8">
            <div className="w-12 h-12 rounded-xl bg-white/5 border border-white/8 flex items-center justify-center flex-shrink-0">
              <WinLogo className="w-6 h-6 text-white/50" />
            </div>
            <div className="min-w-0">
              <h1 className="text-2xl font-bold text-white leading-tight tracking-tight">
                {productName || `Product ${productId}`}
              </h1>
              {buildStr && (
                <p className="text-[12px] font-mono text-zinc-500 mt-0.5">Build {buildStr}</p>
              )}
              {meta.archs.length > 0 && (
                <div className="flex gap-1.5 mt-2">
                  {meta.archs.map(arch => (
                    <span key={arch} className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded bg-white/5 border border-white/8 text-zinc-500">
                      {arch}
                    </span>
                  ))}
                  {meta.badge && (
                    <span className="text-[10px] font-mono font-semibold px-2 py-0.5 rounded-full bg-blue-500/10 border border-blue-500/20 text-blue-400">
                      {meta.badge}
                    </span>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Main card */}
          <div className="rounded-2xl border border-white/7 bg-[#111113] p-6 space-y-5">
            {!productName && !isNotFound && !hasCatalogError ? (
               <div className="space-y-4">
                 <div className="flex items-center gap-2.5 text-xs text-zinc-400 font-medium tracking-wide">
                   <motion.span
                     animate={{ rotate: 360 }}
                     transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                     className="inline-block w-4 h-4 border-2 border-zinc-500/30 border-t-zinc-500 rounded-full"
                   />
                   Loading catalog...
                 </div>
                 <div className="space-y-3">
                   <div className="h-3 w-20 bg-white/6 rounded animate-pulse" />
                   <div className="h-11 bg-white/6 rounded-xl animate-pulse" />
                   <div className="h-11 bg-white/5 rounded-xl animate-pulse" />
                 </div>
               </div>
            ) : hasCatalogError ? (
              <div className="flex flex-col items-center justify-center py-6 text-center">
                <WifiOff size={32} className="text-zinc-600 mb-4" />
                <h2 className="text-lg font-semibold text-white mb-2">Network Error</h2>
                <p className="text-zinc-500 text-sm max-w-xs px-4 leading-relaxed mb-6">
                  Failed to load the product catalog. Please check your internet connection or try refreshing the page.
                </p>
                <button onClick={() => window.location.reload()} className="px-4 py-2 rounded-xl bg-white/5 border border-white/8 text-sm font-medium text-white hover:bg-white/10 transition-colors">
                  Retry
                </button>
              </div>
            ) : isNotFound ? (
              <div className="flex flex-col items-center justify-center py-6 text-center">
                <AlertTriangle size={32} className="text-zinc-600 mb-4" />
                <h2 className="text-lg font-semibold text-white mb-2">Product Not Found</h2>
                <p className="text-zinc-500 text-sm max-w-xs px-4 leading-relaxed mb-6">
                  The product ID you requested ({productId}) does not exist or has been removed from the catalog.
                </p>
                <div className="flex items-center justify-center gap-3 w-full">
                  <Link to="/" className="flex-1 max-w-[140px] px-4 py-2.5 rounded-xl bg-white/5 border border-white/8 text-sm font-medium text-white hover:bg-white/10 transition-colors">
                    Home
                  </Link>
                  <Link to="/products" className="flex-1 max-w-[140px] px-4 py-2.5 rounded-xl bg-blue-500 text-sm font-medium text-white hover:bg-blue-600 transition-colors shadow-lg">
                    Browse all
                  </Link>
                </div>
              </div>
            ) : !meta.active ? (
              <div className="flex flex-col items-center justify-center py-6 text-center">
                <AlertTriangle size={32} className="text-amber-500/50 mb-4" />
                <h2 className="text-lg font-semibold text-white mb-2">Product Discontinued</h2>
                <p className="text-zinc-500 text-sm max-w-sm px-4 leading-relaxed mb-6">
                  This release is no longer hosted on Microsoft's official CDN and cannot be downloaded.
                </p>
                <Link to="/products" className="px-4 py-2.5 rounded-xl bg-white/5 border border-white/8 text-sm font-medium text-white hover:bg-white/10 transition-colors">
                  Browse other releases
                </Link>
              </div>
            ) : isLoadingLangs ? (
              <div className="space-y-4">
                <div className="flex items-center gap-2.5 text-xs text-zinc-400 font-medium tracking-wide">
                  <motion.span
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                    className="inline-block w-4 h-4 border-2 border-zinc-500/30 border-t-zinc-500 rounded-full"
                  />
                  Connecting to Microsoft CDN...
                </div>
                <div className="space-y-3">
                  <div className="h-3 w-20 bg-white/6 rounded animate-pulse" />
                  <div className="h-11 bg-white/6 rounded-xl animate-pulse" />
                  <div className="h-11 bg-white/5 rounded-xl animate-pulse" />
                </div>
              </div>
            ) : error && !languages.length ? (
              <div className="flex items-start gap-3 p-4 rounded-xl bg-red-500/8 border border-red-500/15 text-red-400 text-sm">
                <AlertTriangle size={15} className="flex-shrink-0 mt-0.5" />
                <div>
                  <p className="font-medium mb-0.5">Failed to load languages</p>
                  <p className="text-red-400/70 text-xs">{error}</p>
                </div>
              </div>
            ) : (
              <>
                {/* Language selector */}
                <div>
                  <label className="block text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600 mb-2">
                    Language
                  </label>
                  <Select.Root
                    value={selectedSku?.Id}
                    onValueChange={val => {
                      const sku = languages.find(l => l.Id === val)
                      if (sku) setSelectedSku(sku)
                      setDownloadLinks([])
                      setError(null)
                      setLinkStatus(null)
                    }}
                  >
                    <Select.Trigger className="w-full flex items-center justify-between px-4 py-3 rounded-xl bg-white/5 border border-white/7 text-sm text-white hover:bg-white/7 hover:border-white/12 focus:outline-none transition-all cursor-pointer">
                      <Select.Value placeholder="Select language..." />
                      <Select.Icon>
                        <ChevronDown size={14} className="text-zinc-500" />
                      </Select.Icon>
                    </Select.Trigger>

                    <Select.Portal>
                      <Select.Content
                        className="z-[100] w-[var(--radix-select-trigger-width)] max-h-72 overflow-auto rounded-xl border border-white/9 bg-[#131316] backdrop-blur-xl shadow-2xl text-sm"
                        position="popper"
                        sideOffset={6}
                      >
                        <Select.Viewport className="p-1">
                          {languages.map(lang => (
                            <Select.Item
                              key={lang.Id}
                              value={lang.Id}
                              className="relative flex items-center justify-between px-3 py-2.5 rounded-lg text-zinc-400 data-[highlighted]:text-white data-[highlighted]:bg-white/6 cursor-pointer outline-none transition-colors"
                            >
                              <Select.ItemText>{lang.LocalizedLanguage}</Select.ItemText>
                              <Select.ItemIndicator>
                                <Check size={13} className="text-blue-400" />
                              </Select.ItemIndicator>
                            </Select.Item>
                          ))}
                        </Select.Viewport>
                      </Select.Content>
                    </Select.Portal>
                  </Select.Root>
                </div>

                {/* CTA */}
                <button
                  onClick={handleGetLinks}
                  disabled={!selectedSku || isFetching}
                  className="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-blue-500 text-white text-sm font-semibold hover:bg-blue-600 disabled:opacity-40 disabled:cursor-not-allowed transition-all shadow-[0_0_20px_rgba(59,130,246,0.2)] hover:shadow-[0_0_24px_rgba(59,130,246,0.3)]"
                >
                  {isFetching ? (
                    <>
                      <motion.span
                        animate={{ rotate: 360 }}
                        transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                        className="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full"
                      />
                      Fetching links…
                    </>
                  ) : (
                    <>
                      <Download size={15} />
                      Get Download Links
                    </>
                  )}
                </button>
              </>
            )}

            {/* Error after fetch */}
            <AnimatePresence>
              {error && languages.length > 0 && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: 'auto' }}
                  exit={{ opacity: 0, height: 0 }}
                  className="space-y-3"
                >
                  <div className="flex items-start gap-3 p-4 rounded-xl bg-red-500/8 border border-red-500/15 text-red-400 text-sm">
                    <AlertTriangle size={15} className="flex-shrink-0 mt-0.5" />
                    <span>{isLockdown
                      ? 'Our server has been temporarily blocked by Microsoft and is in a 90-minute cooldown. Use the CLI to get your link instantly — it bypasses our server entirely.'
                      : error
                    }</span>
                  </div>
                  {selectedSku && (
                    <CliHandoff
                      productId={productId!}
                      langName={selectedSku.Language}
                      langDisplay={selectedSku.LocalizedLanguage}
                      highlight
                      defaultOpen
                    />
                  )}
                </motion.div>
              )}
            </AnimatePresence>

            {/* Download links */}
            <AnimatePresence>
              {downloadLinks.length > 0 && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: 'auto' }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ type: 'spring', stiffness: 300, damping: 28 }}
                >
                  <div className="pt-4 border-t border-white/6 space-y-3">
                    {linkStatus === 'stale' ? (
                      /* Stale state: hide links, show banner + CLI recommendation */
                      <>
                        {isLockdown ? (
                          /* Sentinel lockdown — no Refresh, no links, CLI only */
                          <>
                            <div className="flex items-start gap-3 p-3.5 rounded-xl bg-red-500/8 border border-red-500/20 text-red-400">
                              <AlertTriangle size={14} className="flex-shrink-0 mt-0.5" />
                              <p className="text-[12px] leading-relaxed">
                                Our server has been temporarily blocked by Microsoft and is in a 90-minute cooldown. Use the CLI to get your link instantly — it bypasses our server entirely.
                              </p>
                            </div>
                            {selectedSku && (
                              <CliHandoff
                                productId={productId!}
                                langName={selectedSku.Language}
                                langDisplay={selectedSku.LocalizedLanguage}
                                highlight
                                defaultOpen
                              />
                            )}
                          </>
                        ) : (
                          /* Normal stale — show warning, CLI, and Refresh */
                          <>
                            <div className="flex items-start gap-3 p-3.5 rounded-xl bg-amber-500/8 border border-amber-500/20 text-amber-400">
                              <AlertTriangle size={14} className="flex-shrink-0 mt-0.5" />
                              <p className="text-[12px] leading-relaxed">
                                These links are cached and have expired — Microsoft is currently limiting automated requests.
                              </p>
                            </div>
                            {selectedSku && (
                              <CliHandoff
                                productId={productId!}
                                langName={selectedSku.Language}
                                langDisplay={selectedSku.LocalizedLanguage}
                                highlight
                                defaultOpen
                              />
                            )}
                            <button
                              onClick={handleRefreshLinks}
                              disabled={isRefreshing}
                              className="w-full flex items-center justify-center gap-1.5 px-3 py-2.5 rounded-xl bg-white/4 border border-white/8 text-zinc-400 text-[12px] font-medium hover:bg-white/7 disabled:opacity-40 transition-all"
                            >
                              <motion.span
                                animate={{ rotate: isRefreshing ? 360 : 0 }}
                                transition={isRefreshing ? { duration: 1, repeat: Infinity, ease: 'linear' } : {}}
                                className="inline-block"
                              >
                                <RefreshCw size={12} />
                              </motion.span>
                              Try refreshing links
                            </button>
                          </>
                        )}
                      </>
                    ) : (
                      /* Fresh / cached state: show expiry row + links */
                      <>
                        <div className="flex items-center justify-between gap-2">
                          <div className={`flex items-center gap-2 p-3 rounded-lg border text-[11px] flex-1 ${
                            showRefresh
                              ? 'bg-red-500/6 border-red-500/15 text-red-400/80'
                              : 'bg-amber-500/6 border-amber-500/12 text-amber-400/80'
                          }`}>
                            <AlertTriangle size={12} className="flex-shrink-0" />
                            {expiryRemaining !== null
                              ? expiryRemaining <= 0
                                ? 'Links expired — refresh to get new ones'
                                : `Expires in ${formatRemaining(expiryRemaining)} · Direct from Microsoft CDN`
                              : 'Links expire in 24 hours · Direct from Microsoft CDN'
                            }
                          </div>
                          {showRefresh && (
                            <button
                              onClick={handleRefreshLinks}
                              disabled={isRefreshing}
                              className="flex items-center gap-1.5 px-3 py-2.5 rounded-lg bg-blue-500/12 border border-blue-500/20 text-blue-400 text-[11px] font-medium hover:bg-blue-500/20 disabled:opacity-40 transition-all flex-shrink-0"
                            >
                              <motion.span
                                animate={{ rotate: isRefreshing ? 360 : 0 }}
                                transition={isRefreshing ? { duration: 1, repeat: Infinity, ease: 'linear' } : {}}
                                className="inline-block"
                              >
                                <RefreshCw size={11} />
                              </motion.span>
                              Refresh
                            </button>
                          )}
                        </div>

                        {downloadLinks.map(link => {
                          const filename = link.Uri.split('/').pop()?.split('?')[0] ?? 'download.iso'
                          const arch = link.Architecture || archFromUri(link.Uri)
                          const isCopied = copiedUri === link.Uri
                          return (
                            <div
                              key={link.Uri}
                              className="flex items-center justify-between gap-3 p-3.5 rounded-xl bg-white/3 border border-white/5 hover:border-white/9 transition-colors"
                            >
                              <div className="min-w-0">
                                <p className="text-[12px] font-mono text-white truncate">{filename}</p>
                                <p className="text-[11px] text-zinc-600 mt-0.5">{arch}</p>
                              </div>
                              <div className="flex items-center gap-2 flex-shrink-0">
                                <button
                                  onClick={() => handleCopy(link.Uri)}
                                  className="p-2 rounded-lg text-zinc-600 hover:text-white hover:bg-white/6 transition-all"
                                  title="Copy link"
                                >
                                  {isCopied
                                    ? <Check size={13} className="text-green-400" />
                                    : <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} className="w-3.5 h-3.5" aria-hidden="true">
                                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                                        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                                      </svg>
                                  }
                                </button>
                                <a
                                  href={link.Uri}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-500/12 border border-blue-500/20 text-blue-400 text-[11px] font-medium hover:bg-blue-500/20 transition-all"
                                >
                                  <Download size={11} />
                                  {arch}
                                  <ExternalLink size={10} className="opacity-50" />
                                </a>
                              </div>
                            </div>
                          )
                        })}
                      </>
                    )}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>

          {(!isNotFound && !hasCatalogError) && (
            <>
              {/* Fallbacks — only on stale or error */}
              {meta.active && (linkStatus === 'stale' || error !== null) && (
                <>
                  <OfficialFallback
                    productId={productId!}
                    languageName={selectedSku?.LocalizedLanguage ?? ''}
                  />
                  {/* Skip outer CLI card when already shown inside the main card (any stale or error) */}
                  {linkStatus !== 'stale' && !(error !== null && languages.length > 0) && (
                    selectedSku ? (
                      <CliHandoff
                        productId={productId!}
                        langName={selectedSku.Language}
                        langDisplay={selectedSku.LocalizedLanguage}
                      />
                    ) : (
                      <CliHandoff productId={productId!} />
                    )
                  )}
                </>
              )}

              {/* System requirements */}
              <SystemRequirements isWin11={isWin11(productId!)} />

              {/* CLI command */}
              {meta.active && <CliCommand downloadUrl={firstUri} filename={firstFilename} />}

              {/* Related releases */}
              {related.length > 0 && (
                <RelatedReleases current={productId!} items={related} />
              )}
            </>
          )}
        </motion.div>
      </div>
    </>
  )
}
