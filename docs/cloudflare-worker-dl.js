/**
 * Cloudflare Worker for https://dl.swiflow.cc
 *
 * Origin (R2 public): https://r2.swiflow.cc/release-assets/...
 *
 * Routes:
 *   GET /update.json              → release-assets/update.json  (TTL 300s)
 *   GET /release-assets/*         → same key on R2              (binaries TTL 1d;
 *                                   update.json / SHA256SUMS TTL 300s)
 *   GET /clear-cache              → drop Cache API entry for /update.json
 *
 * Deploy on the dl.swiflow.cc zone. Keep in sync with:
 *   - cmd/desktop/updater.go          (defaultUpdateManifestURL)
 *   - .github/workflows/release.yml   (ASSET_BASE)
 */

const R2_DOMAIN = "https://r2.swiflow.cc";
const R2_PREFIX = "release-assets";
const DL_ORIGIN = "https://dl.swiflow.cc";

export default {
  async fetch(request, env, ctx) {
    const { pathname } = new URL(request.url);

    if (pathname === "" || pathname === "/") {
      return new Response("hello");
    }

    if (pathname === "/clear-cache") {
      const cache = caches.default;
      const deleted = await cache.delete(new Request(`${DL_ORIGIN}/update.json`));
      return new Response(deleted ? "success\n" : "not found\n", {
        headers: { "Content-Type": "text/plain; charset=utf-8" },
      });
    }

    if (pathname === "/update.json") {
      return proxyR2(`${R2_PREFIX}/update.json`, 300);
    }

    if (pathname.startsWith(`/${R2_PREFIX}/`)) {
      const key = pathname.slice(1);
      const ttl = /\/(update\.json|SHA256SUMS)$/i.test(pathname) ? 300 : 86400;
      return proxyR2(key, ttl);
    }

    return new Response("Not Found", { status: 404 });
  },
};

function proxyR2(key, cacheTtl) {
  return fetch(`${R2_DOMAIN}/${key}`, {
    cf: {
      cacheTtl,
      cacheEverything: true,
    },
  });
}
