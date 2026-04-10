import { Link } from 'react-router-dom'

export default function SiteFooter() {
  return (
    <footer className="max-w-4xl mx-auto px-5 pb-4 pt-8 text-center">
      <p className="text-[11px] text-zinc-600 leading-relaxed">
        This open-source project is not affiliated with Microsoft Corporation.{' '}
        Windows is a registered trademark of Microsoft Corporation.{' '}
        All ISO files are hosted on Microsoft's official CDN.
      </p>
      <div className="flex items-center justify-center gap-4 mt-2 text-[11px] text-zinc-700">
        <Link to="/privacy-policy" className="hover:text-zinc-500 transition-colors">Privacy Policy</Link>
        <span>·</span>
        <Link to="/disclaimer" className="hover:text-zinc-500 transition-colors">Disclaimer</Link>
        <span>·</span>
        <Link to="/about" className="hover:text-zinc-500 transition-colors">About</Link>
        <span>·</span>
        <a
          href="https://github.com/starkSV/windows-iso-downloader"
          target="_blank"
          rel="noopener noreferrer"
          className="hover:text-zinc-500 transition-colors"
        >
          GitHub
        </a>
      </div>
    </footer>
  )
}
