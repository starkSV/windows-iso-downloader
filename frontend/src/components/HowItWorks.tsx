import { motion } from 'motion/react'
import { LayoutGrid, Globe, Link2 } from 'lucide-react'

const steps = [
  {
    num: '1',
    icon: LayoutGrid,
    title: 'Pick a release',
    desc: 'Choose from Windows 11, 10, and 8.1 across all feature updates.',
  },
  {
    num: '2',
    icon: Globe,
    title: 'Choose language',
    desc: '38 localised languages available for every release.',
  },
  {
    num: '3',
    icon: Link2,
    title: 'Get CDN link',
    desc: 'Direct from Microsoft\'s servers. No proxy, no middleman.',
  },
]

export default function HowItWorks() {
  return (
    <section className="mt-20">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: '-60px' }}
        transition={{ duration: 0.5 }}
      >
        <h2 className="text-xl font-semibold text-white mb-1">How it works</h2>
        <p className="text-sm text-zinc-500 mb-8">Three steps to your official Windows ISO.</p>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-px bg-white/7 rounded-2xl overflow-hidden border border-white/7">
          {steps.map((step, i) => {
            const Icon = step.icon
            return (
              <div
                key={step.num}
                className="bg-[#111113] p-6 flex flex-col gap-4"
              >
                <div className="flex items-center gap-3">
                  <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-[11px] font-mono font-semibold text-blue-400">
                    {i + 1}
                  </span>
                  <Icon size={16} className="text-zinc-500" />
                </div>
                <div>
                  <p className="text-sm font-semibold text-white mb-1">{step.title}</p>
                  <p className="text-sm text-zinc-500 leading-relaxed">{step.desc}</p>
                </div>
              </div>
            )
          })}
        </div>
      </motion.div>
    </section>
  )
}
