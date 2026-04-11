# Issue Drafts

## Inactive products still trigger backend requests

### Summary

The new `active: false` catalog state is not fully enforced on the product detail page.

Inactive products still pass validation and trigger `/skuinfo` requests before the UI shows the discontinued-product panel.

### Affected file

- `frontend/src/pages/ProductDetailPage.tsx`

### Current behavior

For a catalog entry with `active: false`:

- the page loads the product from `products.json`
- `setIsValidated(true)` still runs
- the follow-up effect calls `/skuinfo?product_id=...`
- the backend and Microsoft endpoints are still hit
- only afterward does the page render the `Product Discontinued` state

### Expected behavior

Inactive products should fail fast on the frontend and should not trigger backend or Microsoft API requests.

### Risk

The inactive-product flow still consumes backend capacity and external request budget even though the user cannot download that release.

### Suggested fix

- gate validation on `product.active !== false`
- avoid setting `isValidated` for inactive products
- ensure the `/skuinfo` effect only runs for active products

### Prevention

- add a test covering an inactive catalog entry and assert that no backend request is made
- model the route state explicitly so `active`, `not_found`, and `error` paths cannot fall through into fetch logic
