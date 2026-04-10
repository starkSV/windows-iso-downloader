import { Routes, Route } from 'react-router-dom'
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
    <div className="min-h-screen pb-20 sm:pb-24" style={{ overflowX: 'hidden', maxWidth: '100vw' }}>
      <ScrollToTop />
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/products" element={<ProductsPage />} />
        <Route path="/product/:productId" element={<ProductDetailPage />} />
        <Route path="/about" element={<AboutPage />} />
        <Route path="/privacy-policy" element={<PrivacyPolicyPage />} />
        <Route path="/disclaimer" element={<DisclaimerPage />} />
      </Routes>
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
