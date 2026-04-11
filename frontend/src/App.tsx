import { Routes, Route, Link } from 'react-router-dom'
import { Toaster } from 'sonner'
import Dock from './components/Dock'
import SiteFooter from './components/SiteFooter'
import HomePage from './pages/HomePage'
import ProductsPage from './pages/ProductsPage'
import ProductDetailPage from './pages/ProductDetailPage'
import AboutPage from './pages/AboutPage'
import PrivacyPolicyPage from './pages/PrivacyPolicyPage'
import DisclaimerPage from './pages/DisclaimerPage'
import ScrollToTop from './components/ScrollToTop'

export default function App() {
  return (
    <div className="min-h-screen pb-20 sm:pb-24 flex flex-col" style={{ overflowX: 'hidden', maxWidth: '100vw' }}>
      <ScrollToTop />
      <main className="flex-1">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/products" element={<ProductsPage />} />
          <Route path="/product/:productId" element={<ProductDetailPage />} />
          <Route path="/about" element={<AboutPage />} />
          <Route path="/privacy-policy" element={<PrivacyPolicyPage />} />
          <Route path="/disclaimer" element={<DisclaimerPage />} />
          <Route path="*" element={
            <div className="flex flex-col items-center justify-center min-h-[60vh] text-center px-4">
              <h1 className="text-6xl font-bold text-white mb-4">404</h1>
              <p className="text-zinc-400 mb-8">The page you're looking for doesn't exist.</p>
              <Link to="/" className="px-6 py-2 bg-white text-black font-semibold rounded-full hover:bg-zinc-200 transition-colors">
                Go Home
              </Link>
            </div>
          } />
        </Routes>
      </main>
      <SiteFooter />
      <Dock />
      <Toaster
        position="bottom-center"
        offset={96}
        theme="dark"
        toastOptions={{
          style: {
            background: '#18181b',
            border: '1px solid rgba(255,255,255,0.09)',
            color: '#fafafa',
            fontFamily: 'var(--font-sans)',
          },
        }}
      />
    </div>
  )
}
