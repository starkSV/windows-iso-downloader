export interface EvalProductConfig {
  slug: string
  name: string
  version: string
  build: string
  description: string
  seoDesc: string   // ≤155 chars, for <meta name="description"> and og:description
  archs: string[]
  type: 'server' | 'enterprise'
  requirements: {
    cpu: string
    ram: string
    disk: string
    note?: string
  }
}

export const evalProducts: EvalProductConfig[] = [
  {
    slug: 'server-2025',
    name: 'Windows Server 2025',
    version: 'Standard / Datacenter',
    build: 'Latest evaluation build',
    seoDesc: 'Download Windows Server 2025 evaluation ISO from Microsoft. 180-day trial. Direct CDN link — no registration required.',
    description:
      'The latest Windows Server release featuring enhanced security, hybrid cloud capabilities, and improved performance. Includes SMB over QUIC, credential guard improvements, and delegated managed service accounts.',
    archs: ['x64'],
    type: 'server',
    requirements: {
      cpu: '1.4 GHz 64-bit',
      ram: '512 MB / 2 GB GUI',
      disk: '32 GB',
    },
  },
  {
    slug: 'server-2022',
    name: 'Windows Server 2022',
    version: 'Standard / Datacenter',
    build: 'Latest evaluation build',
    seoDesc: 'Download Windows Server 2022 evaluation ISO from Microsoft. Long-term servicing channel. 180-day trial. Direct CDN link, no registration.',
    description:
      'Long-term servicing channel release with Secured-core server support, TLS 1.3 by default, and DNS-over-HTTPS. The recommended choice for new server infrastructure evaluation.',
    archs: ['x64'],
    type: 'server',
    requirements: {
      cpu: '1.4 GHz 64-bit',
      ram: '512 MB / 2 GB GUI',
      disk: '32 GB',
    },
  },
  {
    slug: 'server-2019',
    name: 'Windows Server 2019',
    version: 'Standard / Datacenter',
    build: '17763',
    seoDesc: 'Download Windows Server 2019 evaluation ISO from Microsoft. Includes Defender ATP and Storage Migration Service. 180-day trial, direct CDN link.',
    description:
      'Stable and widely deployed. Includes Windows Defender Advanced Threat Protection, Storage Migration Service, and System Insights. Ideal for existing infrastructure evaluation.',
    archs: ['x64'],
    type: 'server',
    requirements: {
      cpu: '1.4 GHz 64-bit',
      ram: '512 MB / 2 GB GUI',
      disk: '32 GB',
    },
  },
  {
    slug: 'server-2016',
    name: 'Windows Server 2016',
    version: 'Standard / Datacenter',
    build: '14393',
    seoDesc: 'Download Windows Server 2016 evaluation ISO from Microsoft. Supports Nano Server, Storage Spaces Direct, and Windows Containers. 180-day trial.',
    description:
      'Legacy server release with Nano Server, Storage Spaces Direct, and Windows Containers support. Suitable for older environment compatibility testing.',
    archs: ['x64'],
    type: 'server',
    requirements: {
      cpu: '1.4 GHz 64-bit',
      ram: '512 MB / 2 GB GUI',
      disk: '32 GB',
    },
  },
  {
    slug: 'win11-ent',
    name: 'Windows 11 Enterprise',
    version: 'Evaluation',
    build: 'Latest evaluation build',
    seoDesc: 'Download Windows 11 Enterprise evaluation ISO from Microsoft. Includes BitLocker, Defender for Endpoint, and Intune management. 180-day trial.',
    description:
      'Full enterprise feature set including Windows Hello for Business, BitLocker, Microsoft Defender for Endpoint integration, and advanced management via Intune and Group Policy.',
    archs: ['x64'],
    type: 'enterprise',
    requirements: {
      cpu: '1 GHz, 2+ cores 64-bit',
      ram: '4 GB',
      disk: '64 GB',
      note: 'TPM 2.0 required',
    },
  },
]

export const evalSlugSet = new Set(evalProducts.map(p => p.slug))
