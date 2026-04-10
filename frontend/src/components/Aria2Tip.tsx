import { useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { Terminal, ChevronRight, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'

interface Aria2TipProps {
  downloadUrl?: string
}

export default function Aria2Tip({ downloadUrl }: Aria2TipProps) {
  const [open, setOpen] = useState(false)
  const [copied, setCopied] = useState(false)

  const urlPlaceholder = downloadUrl || 'PASTE_YOUR_LINK_HERE'
  const command = `aria2c -x 16 -s 16 "${urlPlaceholder}"`

  function handleCopy() {
    navigator.clipboard.writeText(command)
    setCopied(true)
    toast.success('Command copied!')
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="mt-3">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 text-[12px] text-zinc-600 hover:text-zinc-400 transition-colors py-1"
      >
        <Terminal size={13} />
        <span>Download faster with aria2 — 16× parallel connections</span>
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
            key="aria2-content"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ type: 'spring', stiffness: 300, damping: 30 }}
            className="overflow-hidden"
          >
            <div className="mt-2 rounded-xl border border-white/7 bg-[#0d0d0f] overflow-hidden">
              <div className="flex items-center justify-between px-4 py-2 border-b border-white/5">
                <span className="text-[10px] font-mono text-zinc-600 uppercase tracking-widest">Terminal</span>
                <button
                  onClick={handleCopy}
                  className="flex items-center gap-1.5 text-[11px] text-zinc-500 hover:text-white transition-colors"
                >
                  {copied ? <Check size={11} className="text-green-400" /> : <Copy size={11} />}
                  <span>{copied ? 'Copied' : 'Copy'}</span>
                </button>
              </div>
              <div className="px-4 py-3 overflow-x-auto">
                <code className="text-[12px] font-mono text-zinc-300 whitespace-nowrap">
                  <span className="text-zinc-600 select-none">$ </span>
                  {command}
                </code>
              </div>
              {!downloadUrl && (
                <p className="px-4 pb-3 text-[11px] text-zinc-600">
                  Generate a link above, then paste it into this command for 16× speed.
                </p>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
