# windows-iso-downloader

> A clean, open-source tool for obtaining official Windows ISO files directly from Microsoft's CDN — without a Windows machine, the Media Creation Tool, or browser restrictions.

**Live:** [msdl.tech-latest.com](https://msdl.tech-latest.com) · **By:** [TechLatest](https://tech-latest.com)

![License](https://img.shields.io/github/license/starkSV/windows-iso-downloader)
![Stars](https://img.shields.io/github/stars/starkSV/windows-iso-downloader)

---

## What it does

MSDL replicates the session flow that Microsoft uses to serve ISO download links — identical to the approach used by [Rufus/Fido](https://github.com/pbatard/Fido). You get the exact same signed CDN URL Microsoft would give you, just without the browser requirement.

- ✅ Direct Microsoft CDN links — no proxying of actual file data
- ✅ 38 languages per consumer release
- ✅ ARM64 + x64 + x86 support
- ✅ Windows Server 2016–2025 and Windows 11 Enterprise evaluation ISOs
- ✅ No account, no browser lock, no ads, no tracking
- ✅ Consumer links expire in 24 hours (Microsoft's standard behaviour, not a limitation)

---

## Screenshots

| Home | All Releases |
|---|---|
| ![Home page showing featured Windows releases](frontend/public/screenshots/msdl-1.jpg) | ![Products listing page with search](frontend/public/screenshots/msdl-2.jpg) |

| Product detail — language selector | Product detail — download links |
|---|---|
| ![Product detail page with language dropdown](frontend/public/screenshots/msdl-3.jpg) | ![Download links with expiry warning](frontend/public/screenshots/msdl-4.jpg) |

---

## Project Structure

```
windows-iso-downloader/
├── frontend/            # React 19 + TypeScript + Vite + Tailwind v4
├── backend/             # Go proxy server (recommended for production)
├── cloudflare-worker/   # Optional CF Worker for distributed IP routing
└── README.md
```

---

## How It Works

```
Browser → Backend → (CF Worker) → Microsoft API → Signed CDN URL
                          ↓
          1. Register session (Microsoft tracking endpoint)
          2. Parse MDT fingerprint script
          3. Fetch SKU list (available languages)
          4. Fetch signed CDN download URL
```

The flow mirrors [Fido.ps1](https://github.com/pbatard/Fido) by Pete Batard — the same script bundled with Rufus.

Outbound requests to Microsoft are optionally routed through a Cloudflare Worker (`cloudflare-worker/worker.js`). This distributes requests across Cloudflare's global edge IPs instead of a single server IP, preventing Microsoft's rate-limit block (error 715-123130) under high traffic. The Worker is opt-in via environment variables — omit them to go direct to Microsoft.

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

#### Optional: Cloudflare Worker (recommended for production)

Deploy `cloudflare-worker/worker.js` to Cloudflare Workers, then set these on the backend:

```env
CF_WORKER_URL=https://your-worker.your-name.workers.dev
CF_WORKER_SECRET=your-secret   # must match the CF_WORKER_SECRET secret set in the Worker's settings
```

Omit both to go direct to Microsoft (fine for local development and low-traffic self-hosting).

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

### `GET /evallinks?product=<slug>`

Returns direct CDN links for evaluation ISOs (Server/Enterprise). Links are resolved from Microsoft's Eval Center fwlink redirects and cached for 24 hours.

Valid slugs: `server-2025`, `server-2022`, `server-2019`, `server-2016`, `win11-ent`

```json
{
  "links": [
    { "arch": "x64", "lang": "en-us", "url": "https://software-static.download.prss.microsoft.com/..." },
    { "arch": "x64", "lang": "fr-fr", "url": "https://software-static.download.prss.microsoft.com/..." }
  ]
}
```

---

## Supported Products

### Consumer releases

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

### Evaluation editions (Server & Enterprise)

180-day trial ISOs sourced directly from Microsoft's Eval Center CDN. No registration required.

| Product | Slug | Architecture |
|---|---|---|
| Windows Server 2025 | `server-2025` | x64 |
| Windows Server 2022 | `server-2022` | x64 |
| Windows Server 2019 | `server-2019` | x64 |
| Windows Server 2016 | `server-2016` | x64 |
| Windows 11 Enterprise | `win11-ent` | x64 |

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

Recommended setup:

| Component | Platform |
|---|---|
| Frontend | Cloudflare Pages / Vercel (static) |
| Backend | VPS (Hetzner, DigitalOcean, Linode, etc.) |
| Outbound proxy | Cloudflare Worker (optional, recommended for public instances) |

> ⚠️ **Deploy the Go backend to a standard VPS**, not serverless platforms. The Cloudflare Worker is used only as an outbound proxy for Microsoft API calls — the backend itself must be a long-running process with session state.

---

## Contributing

Pull requests welcome. To add a new consumer Windows release:

1. Find the product ID on `www.microsoft.com/software-download-connector/api/`
2. Add it to `frontend/public/data/products.json` (name, archs, badge, related, active)
3. Update the product table in this README

To add a new evaluation edition:

1. Find the fwlink URL on `microsoft.com/en-us/evalcenter/download-*`
2. Add the slug and fwlink to the `evalProducts` map in `backend/main.go`
3. Add the product config to `frontend/src/data/evalProducts.ts`
4. Update the eval table in this README

---

## Disclaimer

This project is **not affiliated with, endorsed by, or sponsored by Microsoft Corporation**.
Windows is a registered trademark of Microsoft Corporation.
All ISO files are served directly from Microsoft's official CDN — this project does not host any files.

---

## License

[MIT](./LICENSE) © [TechLatest](https://tech-latest.com)
