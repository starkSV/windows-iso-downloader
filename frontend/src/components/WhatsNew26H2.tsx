const highlights = [
  'AI-powered Settings agent, NPU monitoring in Task Manager, new Taskbar AI tools',
  'Taskbar can move to the top, left, or right of the screen; more Start menu layouts',
  'File Explorer is faster — quicker large-file deletes, quicker Home page load',
  'Search prioritizes your local files before falling back to the web',
]

export default function WhatsNew26H2() {
  return (
    <div className="rounded-xl border border-white/7 bg-[#111113] p-4 mt-5">
      <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600 mb-3">
        What's new in 26H2
      </p>
      <p className="text-sm text-zinc-400 leading-relaxed mb-3">
        Windows 11 26H2 (build 26300.9457), released September 29, 2026 — the first feature update since 25H2.
      </p>
      <ul className="space-y-1.5">
        {highlights.map(item => (
          <li key={item} className="text-[13px] text-zinc-400 leading-relaxed flex gap-2">
            <span className="text-zinc-600 flex-shrink-0">•</span>
            {item}
          </li>
        ))}
      </ul>
    </div>
  )
}
