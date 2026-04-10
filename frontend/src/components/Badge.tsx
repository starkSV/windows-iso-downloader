interface BadgeProps {
  variant: 'latest' | 'stable' | 'eol' | 'legacy'
  label?: string
}

const config = {
  latest: {
    label: 'LATEST',
    className: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
  },
  stable: {
    label: 'STABLE',
    className: 'bg-green-500/15 text-green-400 border-green-500/25',
  },
  eol: {
    label: 'END OF LIFE',
    className: 'bg-amber-500/15 text-amber-400 border-amber-500/25',
  },
  legacy: {
    label: 'LEGACY',
    className: 'bg-zinc-500/15 text-zinc-400 border-zinc-500/25',
  },
}

export default function Badge({ variant, label }: BadgeProps) {
  const { label: defaultLabel, className } = config[variant]
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded-full border text-[10px] font-semibold tracking-widest font-mono ${className}`}
    >
      {label ?? defaultLabel}
    </span>
  )
}
