import { useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { ExternalLink, ChevronRight } from 'lucide-react'

interface Props {
  productId: string
  languageName: string
}

function officialUrl(productId: string): string {
  const id = Number(productId)
  if (id >= 3113) return 'https://www.microsoft.com/software-download/windows11'
  if (id >= 2378) return 'https://www.microsoft.com/software-download/windows10ISO'
  return 'https://www.microsoft.com/software-download/windows8ISO'
}

function editionLabel(productId: string): string {
  const id = Number(productId)
  if (id >= 3113) return 'Windows 11'
  if (id >= 2378) return 'Windows 10'
  return 'Windows 8.1'
}

export default function OfficialFallback({ productId, languageName }: Props) {
  const [open, setOpen] = useState(false)
  const url = officialUrl(productId)
  const edition = editionLabel(productId)

  return (
    <div className="mt-3">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 text-[12px] text-zinc-600 hover:text-zinc-400 transition-colors py-1"
      >
        <ExternalLink size={13} />
        <span>Get it directly from Microsoft</span>
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
            key="official-fallback"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ type: 'spring', stiffness: 300, damping: 30 }}
            className="overflow-hidden"
          >
            <div className="mt-2 rounded-xl border border-white/7 bg-[#0d0d0f] p-4 space-y-3">
              <p className="text-[12px] text-zinc-500 leading-relaxed">
                Microsoft's official page runs the same lookup from your browser — it always works.
              </p>
              <ol className="space-y-2.5">
                <li className="flex items-start gap-2.5">
                  <span className="flex-shrink-0 w-5 h-5 rounded-full bg-white/5 border border-white/10 flex items-center justify-center text-[10px] font-mono text-zinc-500 mt-0.5">1</span>
                  <span className="text-[12px] text-zinc-400">
                    Open{' '}
                    <a
                      href={url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline underline-offset-2"
                    >
                      Microsoft's official {edition} download page
                    </a>
                  </span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="flex-shrink-0 w-5 h-5 rounded-full bg-white/5 border border-white/10 flex items-center justify-center text-[10px] font-mono text-zinc-500 mt-0.5">2</span>
                  <span className="text-[12px] text-zinc-400">
                    Select edition: <span className="font-mono text-zinc-300">{edition}</span>, then click Confirm
                  </span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="flex-shrink-0 w-5 h-5 rounded-full bg-white/5 border border-white/10 flex items-center justify-center text-[10px] font-mono text-zinc-500 mt-0.5">3</span>
                  <span className="text-[12px] text-zinc-400">
                    Select language: <span className="font-mono text-zinc-300">{languageName}</span>, then click Confirm
                  </span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="flex-shrink-0 w-5 h-5 rounded-full bg-white/5 border border-white/10 flex items-center justify-center text-[10px] font-mono text-zinc-500 mt-0.5">4</span>
                  <span className="text-[12px] text-zinc-400">Click the download link for your architecture</span>
                </li>
              </ol>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
