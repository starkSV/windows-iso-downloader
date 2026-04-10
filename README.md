# windows-iso-downloader

> A clean, open-source tool for obtaining official Windows ISO files directly from Microsoft's CDN — without a Windows machine, the Media Creation Tool, or browser restrictions.

**Live:** [msdl.tech-latest.com](https://msdl.tech-latest.com) · **By:** [TechLatest](https://tech-latest.com)

![License](https://img.shields.io/github/license/starkSV/windows-iso-downloader)
![Stars](https://img.shields.io/github/stars/starkSV/windows-iso-downloader)

---

## What it does

MSDL replicates the session flow that Microsoft uses to serve ISO download links — identical to the approach used by [Rufus/Fido](https://github.com/pbatard/Fido). You get the exact same signed CDN URL Microsoft would give you, just without the browser requirement.

- ✅ Direct Microsoft CDN links — no proxying of actual file data
- ✅ 38 languages per release
- ✅ ARM64 + x64 + x86 support
- ✅ No account, no browser lock, no ads, no tracking
- ✅ Links expire in 24 hours (Microsoft's standard behaviour, not a limitation)

---

## Screenshot

> _Coming soon_

---

## Project Structure

```
windows-iso-downloader/
├── frontend/      # React 19 + TypeScript + Vite + Tailwind v4
├── backend/       # Go proxy server (recommended for production)
└── README.md
```

---

## How It Works

```
Browser → Backend Proxy → Microsoft Download API → Signed CDN URL
                               ↓
               1. Register session (Microsoft tracking endpoint)
               2. Parse MDT fingerprint script
               3. Fetch SKU list (available languages)
               4. Fetch signed CDN download URL
```

The flow mirrors [Fido.ps1](https://github.com/pbatard/Fido) by Pete Batard — the same script bundled with Rufus.

---

## Running Locally

### Backend (Go)

```bash
cd backend
go run main.go
# Runs on http://localhost:3002
```

### Frontend

```bash
cd frontend
npm install
npm run dev
# Runs on http://localhost:5173
```

Create `frontend/.env.local`:

```env
VITE_API_URL=http://localhost:3002
```

---

## API Reference

### `GET /skuinfo?product_id=<id>`

Returns available languages for a product.

```json
{
  "Skus": [
    {
      "Id": "0x0409",
      "Language": "en-US",
      "LocalizedLanguage": "English (United States)"
    }
  ]
}
```

### `GET /proxy?product_id=<id>&sku_id=<sku>`

Returns signed download links from Microsoft's CDN.

```json
{
  "ProductDownloadOptions": [
    {
      "Uri": "https://software.download.prss.microsoft.com/...",
      "Architecture": "x64"
    }
  ]
}
```

---

## Supported Products

| Product | ID | Architecture |
|---|---|---|
| Windows 11 25H2 | 3262 | x64 |
| Windows 11 25H2 | 3265 | ARM64 |
| Windows 11 25H2 (updated) | 3321 | x64 |
| Windows 11 25H2 (updated) | 3324 | ARM64 |
| Windows 11 24H2 | 3113 | x64 |
| Windows 11 24H2 | 3131 | ARM64 |
| Windows 10 22H2 | 2618 | x64 / x86 |
| Windows 10 22H2 Home China | 2378 | x64 |
| Windows 8.1 | 52 | x64 / x86 |
| Windows 8.1 Single Language | 48 | x64 / x86 |

---

## Tech Stack

### Frontend (`frontend/`)

| | |
|---|---|
| Framework | React 19 + TypeScript |
| Build tool | Vite 8 |
| Styling | Tailwind CSS v4 |
| Animation | Motion (Framer Motion v12) |
| UI primitives | Radix UI |
| Toast | Sonner |
| Font | Geist |
| Router | React Router v7 |

### Backend (`backend/`)

| | |
|---|---|
| Language | Go 1.22+ |
| HTTP | `net/http` (stdlib, no framework) |
| Session cache | `sync.RWMutex` in-memory · 15 min TTL |
| UUID | `github.com/google/uuid` |
| MDT parsing | `regexp` (stdlib) |

---

## Deployment

> ⚠️ **Deploy the backend to a standard VPS** (Hetzner, DigitalOcean, Linode, etc.) — **not** serverless platforms like Vercel, Cloudflare Workers, or AWS Lambda. Microsoft rate-limits known datacenter IP ranges.

Recommended setup:

| Component | Platform |
|---|---|
| Frontend | Cloudflare Pages / Vercel (static) |
| Backend | VPS with a non-datacenter IP |

---

## Contributing

Pull requests welcome. To add a new Windows release:

1. Find the product ID on `www.microsoft.com/software-download-connector/api/`
2. Add it to `frontend/public/data/products.json`
3. Add related metadata in `ProductDetailPage.tsx` (`PRODUCT_META`, `RELATED_GROUPS`)
4. Update the product table in this README

---

## Disclaimer

This project is **not affiliated with, endorsed by, or sponsored by Microsoft Corporation**.
Windows is a registered trademark of Microsoft Corporation.
All ISO files are served directly from Microsoft's official CDN — this project does not host any files.

---

## License

[MIT](./LICENSE) © [TechLatest](https://tech-latest.com)
