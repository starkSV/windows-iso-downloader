import { Cpu, MemoryStick, HardDrive, Shield } from 'lucide-react'

interface Requirement {
  icon: React.ReactNode
  value: string
  label: string
}

interface SystemRequirementsProps {
  isWin11?: boolean
}

export default function SystemRequirements({ isWin11 = false }: SystemRequirementsProps) {
  const requirements: Requirement[] = [
    { icon: <Cpu size={16} />, value: '1 GHz+', label: 'Processor' },
    { icon: <MemoryStick size={16} />, value: isWin11 ? '4 GB' : '2 GB', label: 'RAM' },
    { icon: <HardDrive size={16} />, value: '64 GB', label: 'Storage' },
  ]

  return (
    <div className="rounded-xl border border-white/7 bg-[#111113] p-4 mt-5">
      <p className="text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-600 mb-3">
        System Requirements
      </p>
      <div className="grid grid-cols-3 gap-3">
        {requirements.map(req => (
          <div key={req.label} className="flex flex-col gap-1.5">
            <span className="text-zinc-500">{req.icon}</span>
            <span className="text-sm font-semibold text-white">{req.value}</span>
            <span className="text-[11px] text-zinc-600">{req.label}</span>
          </div>
        ))}
      </div>
      {isWin11 && (
        <div className="mt-3 pt-3 border-t border-white/5 flex items-center gap-2">
          <Shield size={13} className="text-amber-500 flex-shrink-0" />
          <span className="text-[11px] text-amber-500/80">TPM 2.0 required for Windows 11</span>
        </div>
      )}
    </div>
  )
}
