import { motion } from 'motion/react'

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <h2 className="text-sm font-semibold text-white/80">{title}</h2>
      <div className="text-sm text-zinc-500 leading-relaxed space-y-2">{children}</div>
    </div>
  )
}

export default function PrivacyPolicyPage() {
  return (
    <div className="max-w-2xl mx-auto px-4 pt-12 pb-10">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.35 }}
        className="rounded-2xl border border-white/7 bg-[#111113] p-8 space-y-7"
      >
        <div>
          <h1 className="text-2xl font-bold text-white mb-1">Privacy Policy</h1>
          <p className="text-xs font-mono text-zinc-600">Last updated: April 9, 2026</p>
        </div>

        <Section title="No data collected">
          <p>
            MSDL does not collect, store, log, or share any personally identifiable information.
            There are no user accounts, no cookies, no analytics trackers, and no third-party
            advertising scripts of any kind.
          </p>
        </Section>

        <Section title="How download links work">
          <p>
            When you request a download link, our backend server contacts Microsoft's software
            download API on your behalf using a non-Windows browser user-agent (the same method
            used by open-source tools like Rufus/Fido). Microsoft's servers return a time-limited,
            IP-tied signed URL.
          </p>
          <p>
            Our backend does not log your IP address, does not store the generated link, and does
            not proxy the actual file download — the ISO is downloaded directly from Microsoft's CDN.
          </p>
        </Section>

        <Section title="Third-party services">
          <p>
            All Windows ISO files are hosted on Microsoft's official content delivery network
            (<code className="text-zinc-400 text-[11px] font-mono bg-white/5 px-1 py-0.5 rounded">software.download.prss.microsoft.com</code>).
            We are not responsible for Microsoft's privacy practices or link availability.
          </p>
        </Section>

        <Section title="Open source">
          <p>
            The full source code for both the frontend and backend is publicly available.
            You can inspect, audit, or self-host the entire stack.
          </p>
        </Section>

        <Section title="Contact">
          <p>
            Questions? Reach us via{' '}
            <a
              href="https://tech-latest.com/contact-us/"
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
