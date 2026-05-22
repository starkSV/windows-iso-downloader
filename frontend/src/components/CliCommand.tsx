import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { Terminal, ChevronRight, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'

type Tool = 'wget' | 'curl' | 'aria2'

const PREF_KEY = 'msdl-cli-tool'

const TOOLS: { id: Tool; label: string }[] = [
  { id: 'wget', label: 'wget' },
  { id: 'curl', label: 'curl' },
  { id: 'aria2', label: 'aria2' },
]

function buildCommand(tool: Tool, url: string, filename: string): string {
  switch (tool) {
    case 'wget':
      return `wget -O "${filename}" "${url}"`
    case 'curl':
      return `curl -L -o "${filename}" "${url}"`
    case 'aria2':
      return `aria2c -x 16 -s 16 --disable-ipv6=true -o "${filename}" "${url}"`
  }
}

interface CliCommandProps {
  downloadUrl?: string
  filename?: string
}

export default function CliCommand({ downloadUrl, filename }: CliCommandProps) {
  const [open, setOpen] = useState(false)
  const [copied, setCopied] = useState(false)
  const [tool, setTool] = useState<Tool>('wget')

  useEffect(() => {
    const saved = localStorage.getItem(PREF_KEY) as Tool | null
    if (saved && TOOLS.some(t => t.id === saved)) setTool(saved)
  }, [])

  function selectTool(t: Tool) {
    setTool(t)
    localStorage.setItem(PREF_KEY, t)
  }

  const url = downloadUrl || 'PASTE_YOUR_LINK_HERE'
  const file = filename || 'windows.iso'
  const command = buildCommand(tool, url, file)

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
        <span>Download via terminal</span>
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
            key="cli-content"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ type: 'spring', stiffness: 300, damping: 30 }}
            className="overflow-hidden"
          >
            <div className="mt-2 rounded-xl border border-white/7 bg-[#0d0d0f] overflow-hidden">
              {/* Tab bar */}
              <div className="flex items-center justify-between px-4 py-2 border-b border-white/5">
                <div className="flex items-center gap-1">
                  {TOOLS.map(t => (
                    <button
                      key={t.id}
                      onClick={() => selectTool(t.id)}
                      className={`px-2.5 py-1 rounded-md text-[10px] font-mono font-medium transition-colors ${
                        tool === t.id
                          ? 'bg-white/8 text-white'
                          : 'text-zinc-600 hover:text-zinc-400'
                      }`}
                    >
                      {t.label}
                    </button>
                  ))}
                </div>
                <button
                  onClick={handleCopy}
                  className="flex items-center gap-1.5 text-[11px] text-zinc-500 hover:text-white transition-colors"
                >
                  {copied ? <Check size={11} className="text-green-400" /> : <Copy size={11} />}
                  <span>{copied ? 'Copied' : 'Copy'}</span>
                </button>
              </div>

              {/* Command */}
              <div className="px-4 py-3 overflow-x-auto">
                <code className="text-[12px] font-mono text-zinc-300 whitespace-nowrap">
                  <span className="text-zinc-600 select-none">$ </span>
                  {command}
                </code>
              </div>

              {!downloadUrl && (
                <p className="px-4 pb-3 text-[11px] text-zinc-600">
                  Generate a link above, then the command will update automatically.
                </p>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
