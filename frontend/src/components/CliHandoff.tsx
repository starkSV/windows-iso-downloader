import { useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { Terminal, ChevronRight, Copy, Check, ExternalLink } from 'lucide-react'
import { toast } from 'sonner'

interface Props {
  productId: string
  langName?: string
  langDisplay?: string
  defaultOpen?: boolean
  highlight?: boolean
}

const RELEASES_URL = 'https://minxl.ink/msdl-github-release'

function InstallSteps() {
  const [copied, setCopied] = useState(false)
  const [brewCopied, setBrewCopied] = useState(false)
  const [curlCopied, setCurlCopied] = useState(false)

  function handleCopy() {
    navigator.clipboard.writeText('winget install starkSV.msdl')
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  function handleBrewCopy() {
    navigator.clipboard.writeText('brew tap starkSV/msdl && brew install msdl-cli')
    setBrewCopied(true)
    setTimeout(() => setBrewCopied(false), 2000)
  }

  function handleCurlCopy() {
    navigator.clipboard.writeText('curl -fsSL https://api.msdl.tech-latest.com/install.sh | bash')
    setCurlCopied(true)
    setTimeout(() => setCurlCopied(false), 2000)
  }

  return (
    <div className="space-y-2">
      <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600">1. Install</p>
      <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-3 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-[10px] font-mono font-semibold tracking-widest uppercase text-emerald-500">Windows · winget</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded border border-emerald-500/20 text-emerald-600 font-mono">recommended</span>
        </div>
        <div className="flex items-center justify-between gap-2 rounded bg-black/30 px-3 py-2">
          <code className="text-[12px] font-mono text-zinc-300">winget install starkSV.msdl</code>
          <button onClick={handleCopy} className="flex-shrink-0 text-zinc-500 hover:text-white transition-colors">
            {copied ? <Check size={12} className="text-green-400" /> : <Copy size={12} />}
          </button>
        </div>
      </div>
      <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-3 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-[10px] font-mono font-semibold tracking-widest uppercase text-emerald-500">macOS / Linux · Homebrew</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded border border-emerald-500/20 text-emerald-600 font-mono">recommended</span>
        </div>
        <div className="flex items-center justify-between gap-2 rounded bg-black/30 px-3 py-2">
          <code className="text-[12px] font-mono text-zinc-300">brew tap starkSV/msdl && brew install msdl-cli</code>
          <button onClick={handleBrewCopy} className="flex-shrink-0 text-zinc-500 hover:text-white transition-colors">
            {brewCopied ? <Check size={12} className="text-green-400" /> : <Copy size={12} />}
          </button>
        </div>
      </div>
      <div className="flex items-center justify-between gap-2 rounded-lg bg-white/4 border border-white/7 px-3 py-2">
        <code className="text-[11px] font-mono text-zinc-400 truncate">curl -fsSL https://api.msdl.tech-latest.com/install.sh | bash</code>
        <button onClick={handleCurlCopy} className="flex-shrink-0 text-zinc-500 hover:text-white transition-colors">
          {curlCopied ? <Check size={12} className="text-green-400" /> : <Copy size={12} />}
        </button>
      </div>
      <a
        href={RELEASES_URL}
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-white/4 border border-white/7 text-[12px] text-zinc-400 hover:text-zinc-200 hover:border-white/12 transition-colors"
      >
        <ExternalLink size={12} className="flex-shrink-0 text-zinc-600" />
        Other platforms — GitHub Releases
      </a>
    </div>
  )
}

function RunStep({ command, langDisplay, onCopy, copied }: {
  command: string
  langDisplay?: string
  onCopy: () => void
  copied: boolean
}) {
  return (
    <div className="space-y-1.5">
      <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600">2. Run</p>
      <div className="rounded-lg border border-white/7 bg-[#0a0a0c] overflow-hidden">
        <div className="flex items-center justify-between px-3 py-1.5 border-b border-white/5">
          <span className="text-[10px] font-mono text-zinc-600">{langDisplay ?? 'will prompt for language'}</span>
          <button
            onClick={onCopy}
            className="flex items-center gap-1.5 text-[11px] text-zinc-500 hover:text-white transition-colors"
          >
            {copied ? <Check size={11} className="text-green-400" /> : <Copy size={11} />}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>
        </div>
        <div className="px-3 py-2.5 overflow-x-auto">
          <code className="text-[12px] font-mono text-zinc-300 whitespace-nowrap">
            <span className="text-zinc-600 select-none">$ </span>
            {command}
          </code>
        </div>
      </div>
    </div>
  )
}

export default function CliHandoff({ productId, langName, langDisplay, defaultOpen = false, highlight = false }: Props) {
  const [open, setOpen] = useState(defaultOpen)
  const [copied, setCopied] = useState(false)

  const command = langName
    ? `msdl --id ${productId} --lang "${langName}"`
    : `msdl --id ${productId}`

  function handleCopy() {
    navigator.clipboard.writeText(command)
    setCopied(true)
    toast.success('Command copied!')
    setTimeout(() => setCopied(false), 2000)
  }

  if (highlight) {
    return (
      <div className="rounded-xl border border-blue-500/25 bg-blue-500/5 p-4 space-y-3">
        <div className="flex items-center gap-2">
          <Terminal size={14} className="text-blue-400 flex-shrink-0" />
          <p className="text-[12px] font-semibold text-blue-300">Use the MSDL CLI instead</p>
          <span className="ml-auto text-[10px] font-mono font-semibold px-1.5 py-0.5 rounded bg-blue-500/15 border border-blue-500/20 text-blue-400 uppercase tracking-wide">
            Recommended
          </span>
        </div>
        <p className="text-[12px] text-zinc-500 leading-relaxed">
          Our server is currently blocked by Microsoft. The CLI fetches the link directly from your machine — no server involved.
        </p>
        <InstallSteps />
        <RunStep command={command} langDisplay={langDisplay} onCopy={handleCopy} copied={copied} />
      </div>
    )
  }

  return (
    <div className="mt-3">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 text-[12px] text-zinc-600 hover:text-zinc-400 transition-colors py-1"
      >
        <Terminal size={13} />
        <span>Get it from your machine via CLI</span>
        <motion.span
          animate={{ rotate: open ? 90 : 0 }}
          transition={{ type: 'spring', stiffness: 400, damping: 30 }}
        >
          <ChevronRight size={13} />
        </motion.span>
      </button>

      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            key="cli-handoff"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ type: 'spring', stiffness: 300, damping: 30 }}
            className="overflow-hidden"
          >
            <div className="mt-2 rounded-xl border border-white/7 bg-[#0d0d0f] p-4 space-y-3">
              <p className="text-[12px] text-zinc-500 leading-relaxed">
                The MSDL CLI runs the lookup directly from your machine — no server in the middle.
              </p>
              <InstallSteps />
              <RunStep command={command} langDisplay={langDisplay} onCopy={handleCopy} copied={copied} />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
