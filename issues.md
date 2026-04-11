# Issue Drafts

## Product detail metadata is missing for several valid product IDs

### Summary

Several product IDs exposed in `frontend/public/data/products.json` are not represented in `frontend/src/pages/ProductDetailPage.tsx`.

As a result, valid product pages render incomplete or incorrect metadata.

### Affected files

- `frontend/public/data/products.json`
- `frontend/src/pages/ProductDetailPage.tsx`

### Current behavior

The catalog currently includes product IDs that are missing from `PRODUCT_META` and/or `RELATED_GROUPS`, including:

- `3132`
- `3133`
- `3263`
- `3264`
- `3266`
- `3267`

For those routes, the page falls back to:

```ts
{ badge: '', archs: ['x64'] }
```

This causes visible errors such as:

- ARM64 releases showing `x64`
- missing release badges
- missing related-release links

### Expected behavior

Every product ID published in `products.json` should have correct display metadata, or the page should derive that metadata from a single shared catalog source.

### Suggested fix

- Add missing entries to `PRODUCT_META` and `RELATED_GROUPS`
- Prefer a single source of truth for product metadata instead of maintaining parallel maps
- Add a validation step or test that fails when `products.json` contains IDs not covered by the detail-page metadata

## Product detail page can show stale build and SEO metadata after navigation failures

### Summary

`ProductDetailPage.tsx` does not fully reset state when the route changes, which allows stale product metadata to remain visible after a failed or invalid product lookup.

### Affected file

- `frontend/src/pages/ProductDetailPage.tsx`

### Current behavior

In the product-loading effect:

- `buildStr` is only updated when the fetched product name contains a build number
- `buildStr` is not cleared before loading a new route
- the `catch` path only updates `productName`

This means a user can open a valid product page, navigate to an invalid or failed product route, and still see the previous product's build number and document metadata.

### Expected behavior

When the route changes:

- previous build information should be cleared immediately
- `document.title` and the meta description should always match the current route state
- failed catalog lookups should not leave inherited metadata on screen

### Suggested fix

- Reset `buildStr` at the start of the effect
- Update title and meta description in both success and failure paths
- Consider rendering an explicit not-found state for unknown product IDs instead of falling back to `Product {id}`

## Unknown product routes still trigger backend and Microsoft API requests

### Summary

The frontend accepts any `/product/:productId` route and immediately requests `/skuinfo`, even when the ID is not present in the local catalog.

This causes avoidable backend traffic and unnecessary requests to Microsoft.

### Affected file

- `frontend/src/pages/ProductDetailPage.tsx`

### Current behavior

The page fetches:

```ts
fetch(`${API_BASE}/skuinfo?product_id=${productId}`)
```

for any route param, without validating the ID against `frontend/public/data/products.json`.

As a result:

- typos still hit the backend
- crawler noise still hits Microsoft
- invalid URLs produce confusing pages like `Product 9999`
- unnecessary external calls consume rate-limit budget

### Expected behavior

Unknown product IDs should fail fast on the frontend and render a not-found state without calling the backend.

### Suggested fix

- Validate `productId` against the local catalog before requesting `/skuinfo`
- Render a 404 or invalid-product state for unknown IDs
- Optionally add backend allowlisting if the supported catalog is intentionally fixed

## Release counts are hardcoded in multiple places and already inconsistent

### Summary

The site displays inconsistent release totals because the count is hardcoded in multiple places instead of being derived from the catalog.

### Affected files

- `frontend/src/pages/HomePage.tsx`
- `frontend/src/components/StatsBar.tsx`
- `frontend/public/data/products.json`

### Current behavior

The current values do not match:

- `HomePage.tsx` shows `Browse all 16 releases`
- `StatsBar.tsx` shows `17 releases available`
- `products.json` currently contains `16` entries

### Expected behavior

The release count should be sourced from one canonical dataset and remain consistent across the UI.

### Suggested fix

- Derive the release total from `products.json`
- Pass the computed value into the homepage CTA and stats bar
- Remove hardcoded catalog totals from UI copy
