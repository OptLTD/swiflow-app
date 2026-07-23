/**
 * Cloudflare Worker for https://dl.option.ltd
 *
 * Multi-product CDN: /{product}/… → https://r2.option.ltd/{product}/…
 * No product allowlist — missing keys surface as R2 404.
 *
 * Cache:
 *   update.json / SHA256SUMS → TTL 300s
 *   other assets             → TTL 1d
 *
 * Deploy on the dl.option.ltd zone. Keep in sync with each app's
 * defaultUpdateManifestURL and release.yml ASSET_BASE / R2_PREFIX.
 *
 * Swiflow: https://dl.option.ltd/swiflow/update.json
 * (Same worker as option-worth / other OptLTD apps.)
 */

const R2_DOMAIN = "https://r2.option.ltd";

export default {
  async fetch(request, env, ctx) {
    const { pathname } = new URL(request.url);

    if (pathname === "" || pathname === "/") {
      return new Response("dl.option.ltd\n", {
        headers: { "Content-Type": "text/plain; charset=utf-8" },
      });
    }

    // Require /{product}/… (at least one segment after product).
    const segments = pathname.replace(/^\/+|\/+$/g, "").split("/").filter(Boolean);
    if (segments.length < 2) {
      return new Response("Not Found", { status: 404 });
    }

    const key = segments.join("/");
    const ttl = /(^|\/)(update\.json|SHA256SUMS)$/i.test(key) ? 300 : 86400;
    return proxyR2(key, ttl);
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
