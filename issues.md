# Issue Drafts

## Resolved on `modernize-ui-and-catalog`

### Product detail metadata drift from catalog

Status: resolved on this branch.

The branch removes the hardcoded `PRODUCT_META` and `RELATED_GROUPS` maps from `frontend/src/pages/ProductDetailPage.tsx` and derives badge, architecture, and related-release data from the structured catalog in `frontend/public/data/products.json`.

That closes the earlier mismatch where valid IDs such as `3132`, `3133`, `3263`, `3264`, `3266`, and `3267` rendered incomplete metadata.

### Unknown product routes triggering backend requests

Status: resolved on this branch.

`ProductDetailPage.tsx` now validates `productId` against the local catalog before calling `/skuinfo`. Unknown IDs render a dedicated not-found state instead of immediately hitting the backend and Microsoft endpoints.

### Hardcoded and inconsistent release counts

Status: resolved on this branch.

`HomePage.tsx` and `StatsBar.tsx` now derive the release total from `frontend/public/data/products.json` instead of using stale hardcoded values.

### Product detail stale build and metadata after navigation

Status: mostly resolved on this branch.

The route-loading effect now resets `buildStr`, `productName`, related metadata, and validation state before loading the next product. It also sets explicit title and description values for the not-found path.

Residual concern:

- fetch failures are still surfaced as a not-found style state, which is misleading and is tracked below as a separate issue

### Catalog merge behavior in `msdls_v3.py`

Status: not treated as a defect.

`msdls_v3.py` is used manually for periodic catalog maintenance, and `frontend/public/data/products.json` is curated rather than treated as a strict mirror of current upstream availability.

Because of that, preserving existing entries and metadata during a write is acceptable behavior for this workflow.

## Open issues

### Backend session bootstrap timeout was reduced too aggressively

#### Summary

`backend/main.go` reduces the HTTP client timeout in `setupSession()` from `10 * time.Second` to `3 * time.Second`.

#### Affected file

- `backend/main.go`

#### Current behavior

The session bootstrap performs multiple external Microsoft requests. With the shorter timeout, slower or transient network conditions are more likely to force the code into the “proceed anyway” path.

#### Expected behavior

Session setup should remain tolerant of ordinary network latency and only give up early if there is a clear reliability or UX reason to do so.

#### Risk

This can reduce download-link reliability for users on slower connections or during temporary Microsoft-side latency spikes.

#### Suggested fix

- restore the previous timeout, or choose a value based on observed latency rather than guesswork
- if faster failure is desired, measure each request separately and log timeout frequency before tightening the limit

#### Prevention

- treat external timeout changes as reliability-sensitive behavior changes
- record timeout/error rates when tuning values for upstream services
- add lightweight observability around bootstrap failures before optimizing timeout thresholds

### Catalog fetch failures are shown as “Product Not Found”

#### Summary

`ProductDetailPage.tsx` maps catalog load failures to a not-found style UI, even when the requested `productId` may be valid.

#### Affected file

- `frontend/src/pages/ProductDetailPage.tsx`

#### Current behavior

If fetching or parsing `/data/products.json` fails, the page runs this fallback behavior:

- `setIsNotFound(true)`
- `setProductName('Error Loading Product')`

The rendered state then tells the user the product does not exist or was removed from the catalog.

#### Expected behavior

Operational failures loading the catalog should render an error state distinct from a true unknown product ID.

#### Risk

Users receive incorrect information, and debugging production issues becomes harder because network or asset problems are disguised as missing products.

#### Suggested fix

- separate “catalog load failed” from “catalog lookup missed”
- render a retryable error state for fetch/parse failures
- reserve the not-found state for IDs that were successfully checked against a loaded catalog

#### Prevention

- model route states explicitly: `loading`, `ready`, `not_found`, and `error`
- add a UI test covering catalog fetch failure and asserting that the page does not present a false 404
- avoid reusing not-found messaging for transport or asset failures
