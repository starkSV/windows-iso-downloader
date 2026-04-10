import { motion } from 'motion/react'

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-2.5">
      <h2 className="text-sm font-semibold text-white/80">{title}</h2>
      <div className="text-sm text-zinc-500 leading-relaxed space-y-2">{children}</div>
    </div>
  )
}

export default function AboutPage() {
  return (
    <div className="max-w-2xl mx-auto px-4 pt-12 pb-10">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.35 }}
        className="rounded-2xl border border-white/7 bg-[#111113] p-8 space-y-7"
      >
        <div>
          <h1 className="text-2xl font-bold text-white mb-1">About MSDL</h1>
          <p className="text-xs font-mono text-zinc-600">A TechLatest open-source project</p>
        </div>

        <Section title="What is this?">
          <p>
            MSDL (Microsoft Software Download Links) is a clean, open-source web tool for
            obtaining official Windows ISO files directly from Microsoft's content delivery network
            — without needing a Windows machine, the Media Creation Tool, or a browser lock check.
          </p>
          <p>
            There are no third-party hosts, no redirects, no registration, and no ads.
            Every link you receive is exactly the same signed URL you would get from
            Microsoft's own download page.
          </p>
        </Section>

        <Section title="How it works">
          <p>
            Our backend replicates the session-based authentication flow that Microsoft uses to
            serve download links to end users. The same approach is used by{' '}
            <a
              href="https://github.com/pbatard/Fido"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-400 hover:text-blue-300 transition-colors"
            >
              Fido
            </a>{' '}
            (the PowerShell script bundled with Rufus). The flow is:
          </p>
          <ol className="list-decimal list-inside space-y-1 text-zinc-500 text-sm">
            <li>Register a session with Microsoft's tracking endpoint</li>
            <li>Fetch and parse the MDT fingerprinting script</li>
            <li>Call the SKU info API to retrieve available languages</li>
            <li>Call the download links API using the warmed session to get signed CDN URLs</li>
          </ol>
          <p>
            Links are IP-tied to the server and expire after 24 hours — this is standard
            Microsoft behaviour, not a limitation of MSDL.
          </p>
        </Section>

        <Section title="Data source">
          <p>
            All product data (release names, build numbers, and available architectures) is sourced
            from Microsoft's official software download connector API:
          </p>
          <p>
            <code className="text-[11px] font-mono bg-white/5 border border-white/7 px-2 py-0.5 rounded text-zinc-400 break-all">
              www.microsoft.com/software-download-connector/api/
            </code>
          </p>
          <p>
            Product IDs are maintained manually based on Microsoft's release cadence.
            Windows 11 25H2, 24H2, Windows 10 22H2, and Windows 8.1 are currently listed.
            New releases are added as Microsoft publishes them.
          </p>
        </Section>

        <Section title="Credits">
          <p>
            The underlying session flow is inspired by{' '}
            <a
              href="https://github.com/pbatard/Fido"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-400 hover:text-blue-300 transition-colors"
            >
              Fido
            </a>{' '}
            by Pete Batard, the same mechanism powering Rufus's ISO download feature.
          </p>
          <p>
            Built and maintained by{' '}
            <a
              href="https://tech-latest.com"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-400 hover:text-blue-300 transition-colors"
            >
              TechLatest
            </a>.
          </p>
        </Section>
      </motion.div>
    </div>
  )
}
