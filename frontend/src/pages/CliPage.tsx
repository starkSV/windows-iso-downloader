import { useState } from 'react'
import { motion } from 'motion/react'
import { Terminal, Download, Copy, Check, ExternalLink, Zap, WifiOff, Code } from 'lucide-react'
import { toast } from 'sonner'

const RELEASES_URL = 'https://github.com/starkSV/windows-iso-downloader/releases/latest'
const SITE_URL = 'https://msdl.tech-latest.com'

function CodeBlock({ code, label }: { code: string; label?: string }) {
  const [copied, setCopied] = useState(false)

  function handleCopy() {
    navigator.clipboard.writeText(code)
    setCopied(true)
    toast.success('Copied!')
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="rounded-xl border border-white/7 bg-[#0a0a0c] overflow-hidden">
      {label && (
        <div className="flex items-center justify-between px-4 py-2 border-b border-white/5">
          <span className="text-[10px] font-mono text-zinc-600">{label}</span>
          <button
            onClick={handleCopy}
            className="flex items-center gap-1.5 text-[11px] text-zinc-500 hover:text-white transition-colors"
          >
            {copied ? <Check size={11} className="text-green-400" /> : <Copy size={11} />}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>
        </div>
      )}
      <div className="px-4 py-3 overflow-x-auto flex items-center justify-between gap-3">
        <code className="text-[13px] font-mono text-zinc-300 whitespace-nowrap">
          <span className="text-zinc-600 select-none">$ </span>
          {code}
        </code>
        {!label && (
          <button
            onClick={handleCopy}
            className="flex-shrink-0 text-zinc-600 hover:text-white transition-colors"
          >
            {copied ? <Check size={13} className="text-green-400" /> : <Copy size={13} />}
          </button>
        )}
      </div>
    </div>
  )
}

export default function CliPage() {
  return (
    <>
      <title>msdl CLI — Windows ISO downloader for the command line</title>
      <meta name="description" content="Download Windows ISO links directly from your machine using the msdl command-line tool. No browser, no server — works even when the web service is blocked." />
      <link rel="canonical" href={`${SITE_URL}/cli`} />

      <div className="max-w-2xl mx-auto px-5 pt-12 pb-16">
        <motion.div
          initial={{ opacity: 0, y: 14 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35 }}
          className="space-y-10"
        >

          {/* Hero */}
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center flex-shrink-0">
                <Terminal size={20} className="text-emerald-400" />
              </div>
              <div>
                <p className="text-[10px] font-mono font-semibold tracking-widest uppercase text-emerald-500 mb-0.5">Command-line tool</p>
                <h1 className="text-2xl font-bold text-white leading-tight">msdl CLI</h1>
              </div>
            </div>
            <p className="text-zinc-400 leading-relaxed">
              Get official Windows ISO download links directly from your machine — no browser, no server in the middle. The same links as the web app, fetched straight from Microsoft.
            </p>
          </div>

          {/* Why */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {[
              { icon: <WifiOff size={15} />, title: 'Server-independent', desc: "Works even when msdl.tech is blocked by Microsoft's WAF." },
              { icon: <Code size={15} />, title: 'Scriptable', desc: 'Pipe the URL directly into wget, curl, or aria2 for automation.' },
              { icon: <Zap size={15} />, title: 'Fast', desc: 'No page load, no UI. Just the link.' },
            ].map(item => (
              <div key={item.title} className="p-4 rounded-xl border border-white/7 bg-white/2 space-y-1.5">
                <div className="text-zinc-500">{item.icon}</div>
                <p className="text-sm font-medium text-white">{item.title}</p>
                <p className="text-[12px] text-zinc-500 leading-relaxed">{item.desc}</p>
              </div>
            ))}
          </div>

          {/* Install */}
          <div className="space-y-4">
            <h2 className="text-base font-semibold text-white">Install</h2>

            {/* winget — Windows primary */}
            <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4 space-y-2.5">
              <div className="flex items-center justify-between">
                <span className="text-[11px] font-mono font-semibold tracking-widest uppercase text-emerald-500">Windows · winget</span>
                <span className="text-[10px] px-1.5 py-0.5 rounded border border-emerald-500/20 text-emerald-600 font-mono">recommended</span>
              </div>
              <CodeBlock code="winget install starkSV.msdl" />
            </div>

            {/* Manual / other platforms */}
            <div className="rounded-xl border border-white/7 bg-[#111113] p-4 space-y-3">
              <div className="flex items-center justify-between">
                <p className="text-[12px] text-zinc-400">Other platforms — download binary manually:</p>
                <a
                  href={RELEASES_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1.5 text-[12px] text-emerald-400 hover:text-emerald-300 transition-colors flex-shrink-0"
                >
                  <Download size={13} />
                  GitHub Releases
                  <ExternalLink size={11} className="opacity-60" />
                </a>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[12px]">
                {[
                  { os: 'Windows (manual)', file: 'msdl-windows-amd64.exe', rename: 'msdl.exe' },
                  { os: 'macOS (Apple Silicon)', file: 'msdl-darwin-arm64', rename: 'msdl' },
                  { os: 'macOS (Intel)', file: 'msdl-darwin-amd64', rename: 'msdl' },
                  { os: 'Linux', file: 'msdl-linux-amd64', rename: 'msdl' },
                ].map(p => (
                  <div key={p.os} className="px-3 py-2.5 rounded-lg bg-white/3 border border-white/5 space-y-1">
                    <p className="text-zinc-500">{p.os}</p>
                    <p className="font-mono text-zinc-300">{p.file}</p>
                    <p className="text-zinc-600">Rename to <code className="text-zinc-400">{p.rename}</code> · add to PATH</p>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Usage */}
          <div className="space-y-4">
            <h2 className="text-base font-semibold text-white">Usage</h2>

            <div className="space-y-3">
              <p className="text-[12px] text-zinc-500 font-mono uppercase tracking-widest">Interactive — pick product and language</p>
              <CodeBlock code="msdl" />
            </div>

            <div className="space-y-3">
              <p className="text-[12px] text-zinc-500 font-mono uppercase tracking-widest">Skip the picker with flags</p>
              <CodeBlock code='msdl --id 3262 --lang "English"' label="Windows 11 25H2 · English" />
              <CodeBlock code='msdl --id 3262' label="Windows 11 25H2 · prompts for language" />
            </div>

            <div className="space-y-3">
              <p className="text-[12px] text-zinc-500 font-mono uppercase tracking-widest">Evaluation / Server ISOs</p>
              <CodeBlock code="msdl --eval server-2025" label="Windows Server 2025" />
              <CodeBlock code="msdl --eval win11-ent" label="Windows 11 Enterprise" />
            </div>

            <div className="space-y-3">
              <p className="text-[12px] text-zinc-500 font-mono uppercase tracking-widest">Pipe directly into a download tool</p>
              <CodeBlock code='msdl --id 3262 --lang "English" | wget -i -' label="wget" />
              <CodeBlock code='msdl --id 3262 --lang "English" | xargs curl -L -O' label="curl" />
              <CodeBlock code='msdl --id 3262 --lang "English" | xargs aria2c -x 16 -s 16' label="aria2c" />
            </div>

            <div className="space-y-3">
              <p className="text-[12px] text-zinc-500 font-mono uppercase tracking-widest">List all available products</p>
              <CodeBlock code="msdl --list" />
            </div>
          </div>

          {/* Contribute */}
          <div className="space-y-3">
            <h2 className="text-base font-semibold text-white">Sharing links with the web app</h2>
            <div className="rounded-xl border border-white/7 bg-[#111113] p-4 space-y-2">
              <p className="text-[12px] text-zinc-400 leading-relaxed">
                By default, each successful fetch is shared with the <a href={SITE_URL} className="text-blue-400 hover:text-blue-300 underline underline-offset-2">msdl web app</a> to warm its cache — so the next person who visits gets a cached link instead of hitting Microsoft's API cold.
              </p>
              <p className="text-[12px] text-zinc-500 leading-relaxed">
                This is a background POST to our backend. No personal data is sent — only the product ID, SKU ID, and the raw Microsoft JSON response.
              </p>
              <div className="pt-2 border-t border-white/5">
                <p className="text-[12px] text-zinc-600 mb-1.5">To opt out:</p>
                <CodeBlock code="msdl --id 3262 --lang English --no-contribute" />
              </div>
            </div>
          </div>

          {/* Open source */}
          <div className="rounded-xl border border-white/7 bg-white/2 p-4 flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium text-white mb-0.5">Open source</p>
              <p className="text-[12px] text-zinc-500">Built in Go. MIT licensed. Contributions welcome.</p>
            </div>
            <a
              href="https://github.com/starkSV/windows-iso-downloader"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-white/5 border border-white/8 text-[12px] text-zinc-400 hover:text-white hover:border-white/15 transition-all flex-shrink-0"
            >
              GitHub
              <ExternalLink size={11} />
            </a>
          </div>

        </motion.div>
      </div>
    </>
  )
}
