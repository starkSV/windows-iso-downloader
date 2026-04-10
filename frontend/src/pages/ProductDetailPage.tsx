import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'motion/react'
import * as Select from '@radix-ui/react-select'
import { ArrowLeft, ChevronDown, Download, AlertTriangle, Check, ExternalLink } from 'lucide-react'
import { toast } from 'sonner'
import type { Sku, DownloadOption } from '../types'
import SystemRequirements from '../components/SystemRequirements'
import Aria2Tip from '../components/Aria2Tip'
import RelatedReleases from '../components/RelatedReleases'

const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:3002'

// Map product ID → metadata
const PRODUCT_META: Record<string, { badge: string; archs: string[] }> = {
  '3262': { badge: 'LATEST', archs: ['x64'] },
  '3265': { badge: 'LATEST', archs: ['ARM64'] },
  '3321': { badge: 'LATEST', archs: ['x64'] },
  '3324': { badge: 'LATEST', archs: ['ARM64'] },
  '3113': { badge: 'STABLE', archs: ['x64'] },
  '3131': { badge: 'STABLE', archs: ['ARM64'] },
  '2618': { badge: 'EOL SOON', archs: ['x64', 'x86'] },
  '2378': { badge: 'EOL SOON', archs: ['x64'] },
  '52': { badge: 'LEGACY', archs: ['x64', 'x86'] },
  '48': { badge: 'LEGACY', archs: ['x64', 'x86'] },
}

// Sibling release groups for "Also available"
const RELATED_GROUPS: Record<string, { id: string; label: string }[]> = {
  '3262': [
    { id: '3265', label: 'Win 11 25H2 ARM64' },
    { id: '3321', label: 'Win 11 25H2 (updated)' },
    { id: '3113', label: 'Win 11 24H2' },
  ],
  '3265': [
    { id: '3262', label: 'Win 11 25H2 x64' },
    { id: '3321', label: 'Win 11 25H2 (updated)' },
    { id: '3131', label: 'Win 11 24H2 ARM64' },
  ],
  '3321': [
    { id: '3262', label: 'Win 11 25H2' },
    { id: '3324', label: 'Win 11 25H2 ARM64' },
    { id: '3113', label: 'Win 11 24H2' },
  ],
  '3324': [
    { id: '3321', label: 'Win 11 25H2 x64' },
    { id: '3262', label: 'Win 11 25H2' },
    { id: '3265', label: 'Win 11 25H2 ARM64 (alt)' },
  ],
  '3113': [
    { id: '3262', label: 'Win 11 25H2' },
    { id: '3131', label: 'Win 11 24H2 ARM64' },
    { id: '2618', label: 'Win 10 22H2' },
  ],
  '3131': [
    { id: '3265', label: 'Win 11 25H2 ARM64' },
    { id: '3113', label: 'Win 11 24H2 x64' },
  ],
  '2618': [
    { id: '3113', label: 'Win 11 24H2' },
    { id: '3262', label: 'Win 11 25H2' },
    { id: '52', label: 'Win 8.1' },
  ],
  '52': [
    { id: '48', label: 'Win 8.1 Single Language' },
    { id: '2618', label: 'Win 10 22H2' },
  ],
}

function WinLogo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 88 88" className={className} fill="currentColor">
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

  const meta = PRODUCT_META[productId!] ?? { badge: '', archs: ['x64'] }
  const related = RELATED_GROUPS[productId!] ?? []

  // Load product name + build string
  useEffect(() => {
    fetch('/data/products.json')
      .then(r => r.json())
      .then((data: Record<string, string>) => {
        const name = data[productId!] ?? `Product ${productId}`
        setProductName(name)
        // Extract build from parentheses e.g. "(26200.6584)"
        const match = name.match(/\(([^)]+)\)/)
        if (match) setBuildStr(match[1])
      })
      .catch(() => setProductName(`Product ${productId}`))
  }, [productId])

  // Fetch languages on mount
  useEffect(() => {
    if (!productId) return
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
        // Default to English if present
        const english = data.Skus.find((s: Sku) =>
          s.LocalizedLanguage.toLowerCase().includes('english') &&
          !s.LocalizedLanguage.toLowerCase().includes('international')
        ) ?? data.Skus[0]
        setSelectedSku(english)
      })
      .catch(e => setError(e.message))
      .finally(() => setIsLoadingLangs(false))
  }, [productId])

  async function handleGetLinks() {
    if (!selectedSku || !productId) return
    setIsFetching(true)
    setError(null)
    setDownloadLinks([])

    fetch(`${API_BASE}/proxy?product_id=${productId}&sku_id=${selectedSku.Id}`)
      .then(r => r.json())
      .then(data => {
        if (data.error) throw new Error(data.error)
        if (!data.ProductDownloadOptions?.length) throw new Error('No download links returned.')
        setDownloadLinks(data.ProductDownloadOptions)
      })
      .catch(e => {
        setError(e.message)
        toast.error(e.message)
      })
      .finally(() => setIsFetching(false))
  }

  function handleCopy(uri: string) {
    navigator.clipboard.writeText(uri)
    setCopiedUri(uri)
    toast.success('Link copied to clipboard!')
    setTimeout(() => setCopiedUri(null), 2000)
  }

  // First download link URI for aria2 tip
  const firstUri = downloadLinks[0]?.Uri

  return (
    <div className="max-w-xl mx-auto px-5 pt-12 pb-10">
      {/* Back */}
      <button
        onClick={() => navigate(-1)}
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
          {isLoadingLangs ? (
            <div className="space-y-3">
              <div className="h-3 w-20 bg-white/6 rounded animate-pulse" />
              <div className="h-11 bg-white/6 rounded-xl animate-pulse" />
              <div className="h-11 bg-white/5 rounded-xl animate-pulse" />
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
                className="flex items-start gap-3 p-4 rounded-xl bg-red-500/8 border border-red-500/15 text-red-400 text-sm"
              >
                <AlertTriangle size={15} className="flex-shrink-0 mt-0.5" />
                <span>{error}</span>
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
                  {/* Warning */}
                  <div className="flex items-center gap-2 p-3 rounded-lg bg-amber-500/6 border border-amber-500/12 text-amber-400/80 text-[11px]">
                    <AlertTriangle size={12} className="flex-shrink-0" />
                    Links expire in 24 hours · IP-tied · Use a download manager for full speed
                  </div>

                  {/* Links */}
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
                              : <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} className="w-3.5 h-3.5">
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
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* System requirements */}
        <SystemRequirements isWin11={isWin11(productId!)} />

        {/* aria2 tip */}
        <Aria2Tip downloadUrl={firstUri} />

        {/* Related releases */}
        {related.length > 0 && (
          <RelatedReleases current={productId!} items={related} />
        )}
      </motion.div>
    </div>
  )
}
