import { useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { ChevronRight } from 'lucide-react'

interface FaqItem {
  q: string
  a: string
}

const faqs: FaqItem[] = [
  {
    q: 'Is this legal?',
    a: 'Yes. MSDL does not host, modify, or redistribute any Microsoft files. It automates access to download links that Microsoft already provides for free at microsoft.com/software-download — the same way Rufus and the open-source Fido script do. Microsoft itself recommends Rufus (which uses the identical approach) in its official documentation. You are downloading a file directly from Microsoft\'s own servers. A valid Windows license is still required to activate the OS.',
  },
  {
    q: 'Are these download links safe?',
    a: 'Yes. MSDL never stores or modifies any files. It retrieves signed, time-limited download links directly from Microsoft\'s servers (software.download.prss.microsoft.com). You can verify this by inspecting your browser\'s network requests.',
  },
  {
    q: 'Why do links expire in 24 hours?',
    a: 'Microsoft generates short-lived, IP-tied signed URLs to prevent redistribution and hotlinking of their ISOs. This is the same link you\'d get from the official Microsoft Software Download page — we just make it accessible without a browser.',
  },
  {
    q: 'Can I use this with IDM or aria2?',
    a: 'Absolutely — and we strongly recommend it. Microsoft throttles single-threaded browser downloads. Tools like aria2 (aria2c -x 16 -s 16 "URL") or IDM use 16 parallel connections, which typically give you full line speed on their CDN.',
  },
  {
    q: 'What\'s the difference between the various product IDs?',
    a: 'Each Windows release has a unique product ID on Microsoft\'s servers. For example, 3262 is Windows 11 25H2 (x64) and 3265 is Windows 11 25H2 (ARM64). MSDL exposes all known IDs so you can always find the exact build you need.',
  },
  {
    q: 'Is MSDL open source?',
    a: 'Yes. Both the frontend and the backend proxy are fully open source. The backend implements the same session-based flow as the Fido PowerShell script, ported to Go and Node.js. You can self-host the entire stack.',
  },
]

export default function FAQAccordion() {
  const [open, setOpen] = useState<number | null>(null)

  return (
    <section className="mt-20 mb-8">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: '-60px' }}
        transition={{ duration: 0.5 }}
      >
        <h2 className="text-xl font-semibold text-white mb-1">Frequently asked questions</h2>
        <p className="text-sm text-zinc-500 mb-8">Everything you need to know before downloading.</p>

        <div className="rounded-2xl border border-white/7 overflow-hidden divide-y divide-white/5">
          {faqs.map((faq, i) => (
            <div key={i} className="bg-[#111113]">
              <button
                onClick={() => setOpen(open === i ? null : i)}
                className="w-full flex items-center justify-between px-5 py-4 text-left group"
              >
                <span className={`text-sm font-medium transition-colors ${open === i ? 'text-white' : 'text-zinc-300 group-hover:text-white'}`}>
                  {faq.q}
                </span>
                <motion.span
                  animate={{ rotate: open === i ? 90 : 0 }}
                  transition={{ type: 'spring', stiffness: 400, damping: 30 }}
                  className="flex-shrink-0 ml-4 text-zinc-600 group-hover:text-zinc-400 transition-colors"
                >
                  <ChevronRight size={16} />
                </motion.span>
              </button>

              <AnimatePresence initial={false}>
                {open === i && (
                  <motion.div
                    key="content"
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: 'auto', opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    transition={{ type: 'spring', stiffness: 300, damping: 30 }}
                    className="overflow-hidden"
                  >
                    <p className="px-5 pb-5 text-sm text-zinc-400 leading-relaxed">
                      {faq.a}
                    </p>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          ))}
        </div>
      </motion.div>
    </section>
  )
}
