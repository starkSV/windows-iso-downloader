import { motion } from 'motion/react'
import { Check, X, Minus } from 'lucide-react'

type CellValue = true | false | 'varies'

interface Row {
  feature: string
  msdl: CellValue
  official: CellValue
  others: CellValue
}

const rows: Row[] = [
  { feature: 'Direct CDN link',      msdl: true,     official: true,     others: false },
  { feature: 'No browser required',  msdl: true,     official: false,    others: 'varies' },
  { feature: 'ARM64 support',        msdl: true,     official: false,    others: false },
  { feature: 'No account needed',    msdl: true,     official: true,     others: 'varies' },
  { feature: 'No ads or tracking',   msdl: true,     official: true,     others: false },
]

function Cell({ value, highlight = false }: { value: CellValue; highlight?: boolean }) {
  if (value === true)
    return <Check size={15} className={highlight ? 'text-blue-400' : 'text-green-500'} />
  if (value === false)
    return <X size={15} className="text-zinc-600" />
  return <Minus size={15} className="text-zinc-500" />
}

export default function ComparisonTable() {
  return (
    <section className="mt-20">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: '-60px' }}
        transition={{ duration: 0.5 }}
      >
        <h2 className="text-xl font-semibold text-white mb-1">Why MSDL?</h2>
        <p className="text-sm text-zinc-500 mb-8">Compared to your other options.</p>

        {/* Scrollable wrapper on mobile to prevent page-level overflow */}
        <div className="overflow-x-auto rounded-2xl border border-white/7" style={{ WebkitOverflowScrolling: 'touch' }}>
          <div style={{ minWidth: '480px' }}>
            {/* Header */}
            <div className="grid grid-cols-4 bg-[#111113] border-b border-white/7 text-[11px] font-mono font-semibold uppercase tracking-widest">
              <div className="px-4 py-3 text-zinc-600">Feature</div>
              <div className="px-4 py-3 text-blue-400 bg-blue-500/5 border-x border-blue-500/10 text-center">MSDL</div>
              <div className="px-4 py-3 text-zinc-500 text-center">Official</div>
              <div className="px-4 py-3 text-zinc-500 text-center">Others</div>
            </div>

            {/* Rows */}
            {rows.map((row, i) => (
              <div
                key={row.feature}
                className={`grid grid-cols-4 border-b border-white/5 last:border-0 ${i % 2 === 0 ? 'bg-[#0d0d0f]' : 'bg-[#111113]'}`}
              >
                <div className="px-4 py-3.5 text-sm text-zinc-400">{row.feature}</div>
                <div className="px-4 py-3.5 flex justify-center bg-blue-500/3 border-x border-blue-500/8">
                  <Cell value={row.msdl} highlight />
                </div>
                <div className="px-4 py-3.5 flex justify-center">
                  <Cell value={row.official} />
                </div>
                <div className="px-4 py-3.5 flex justify-center">
                  <Cell value={row.others} />
                </div>
              </div>
            ))}
          </div>
        </div>

        <p className="mt-2 text-[11px] text-zinc-700 text-right pr-1 sm:hidden">← scroll →</p>
      </motion.div>
    </section>
  )
}
