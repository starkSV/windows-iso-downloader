import { useEffect } from 'react'
import { Link } from 'react-router-dom'

export default function NotFoundPage() {
  useEffect(() => { window.scrollTo(0, 0) }, [])

  return (
    <>
      <title>404 Not Found | Windows ISO Downloader</title>
      <meta name="description" content="The page you are looking for does not exist." />
      <meta name="robots" content="noindex" />
      <link rel="canonical" href="https://msdl.tech-latest.com/" />
      <div className="flex flex-col items-center justify-center min-h-[60vh] text-center px-4">
        <h1 className="text-6xl font-bold text-white mb-4">404</h1>
        <p className="text-zinc-400 mb-8">The page you're looking for doesn't exist.</p>
        <Link to="/" className="px-6 py-2 bg-white text-black font-semibold rounded-full hover:bg-zinc-200 transition-colors">
          Go Home
        </Link>
      </div>
    </>
  )
}
