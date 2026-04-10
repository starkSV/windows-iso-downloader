import { motion } from 'motion/react'

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <h2 className="text-sm font-semibold text-white/80">{title}</h2>
      <div className="text-sm text-zinc-500 leading-relaxed space-y-2">{children}</div>
    </div>
  )
}

export default function DisclaimerPage() {
  return (
    <div className="max-w-2xl mx-auto px-4 pt-12 pb-10">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.35 }}
        className="rounded-2xl border border-white/7 bg-[#111113] p-8 space-y-7"
      >
        <div>
          <h1 className="text-2xl font-bold text-white mb-1">Disclaimer</h1>
          <p className="text-xs font-mono text-zinc-600">Last updated: April 9, 2026</p>
        </div>

        <div className="p-4 rounded-xl border border-amber-500/15 bg-amber-500/6 text-amber-400/80 text-[12px] leading-relaxed">
          This open-source project is <strong className="text-amber-400">not affiliated with, endorsed by, or sponsored by
          Microsoft Corporation</strong> in any way. Windows is a registered trademark of Microsoft Corporation.
        </div>

        <Section title="No affiliation with Microsoft">
          <p>
            MSDL is an independent, open-source project. It is not produced, approved, or supported
            by Microsoft Corporation. The name "Windows" and the Windows logo are registered
            trademarks of Microsoft Corporation.
          </p>
        </Section>

        <Section title="Purpose">
          <p>
            MSDL exists solely to make it easier for users to access official, unmodified Microsoft
            Windows ISO files that Microsoft already makes freely available for download. All files
            are served directly from Microsoft's own servers — we do not host, mirror, or modify
            any content.
          </p>
        </Section>

        <Section title="No warranty">
          <p>
            This tool is provided "as is" without warranty of any kind. We make no guarantees
            about the availability of Microsoft's download links, the uptime of our backend, or
            the compatibility of any downloaded ISO with your hardware configuration.
          </p>
        </Section>

        <Section title="Use at your own risk">
          <p>
            By using this tool, you agree that you are solely responsible for complying with
            Microsoft's End User License Agreement (EULA) and any applicable laws in your jurisdiction.
            Downloading Windows does not grant you a license to use it — a valid product key or
            digital license is required for activation.
          </p>
        </Section>

        <Section title="Open source">
          <p>
            The source code is publicly available. You are free to inspect, fork, and self-host
            this project under the terms of its open-source license. The project credits the
            open-source Fido script by Pete Batard for the underlying session flow.
          </p>
        </Section>
      </motion.div>
    </div>
  )
}
