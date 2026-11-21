# Responsive browser regression tests

Run from `frontend`:

```sh
npm ci
npx playwright install chromium
npm run test:mobile
```

The tests start Vite on port 5186 and intercept API calls with isolated fixtures.
They never use production credentials, upload real books, or launch parsing jobs.
Set `PLAYWRIGHT_CHROMIUM_EXECUTABLE` to use an already installed Chromium.

Coverage includes 360/390/430/768/1024/1440 px layouts, navigation focus and
independent desktop preferences, the composer at reduced viewport height,
landscape and enlarged text, settings editing/saving, upload persistence,
Wiki reading position, and graph pan/pinch/zoom. Screenshot output is saved
under `test-results` (ignored by Git).

Chromium touch emulation and viewport resizing do not validate a physical
Android/iOS keyboard, Safari viewport behavior, or screen-reader announcements.
Those remain device checks; do not report emulation as physical-device testing.

# noVNC regression

Apply shared infrastructure PR #4141, then run `npm run test:e2e -- e2e/novnc.spec.ts`. The regression emulates a never-resolving VideoDecoder capability probe: upstream's static noVNC import blocks chat startup, while the lazy import leaves chat available. The browser harness and CI proposal are reviewed separately in #4141.
