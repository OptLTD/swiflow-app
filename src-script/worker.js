// Thanks for cloudflare's worker service

const GITHUB_REPO = "OptLTD/swiflow-app";
const R2_DOMAIN = "https://r2.swiflow.cc";

function extractVersionFromFilename(filename) {
  const match = filename.match(/Swiflow_(\d+\.\d+\.\d+)_/);
  return match ? match[1] : null;
}

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const pathname = url.pathname;

    if (pathname == '' || pathname == '/') {
      return new Response("hello");
    }

    if (pathname === "/clear-cache") {
      const cache = caches.default;
      const cacheKey = new Request("https://dl.swiflow.cc/release.json");
      const cached = await cache.match(cacheKey);
      if (cached) {
        ctx.waitUntil(cache.delete(cacheKey));
        return new Response("success");
      }
      return new Response("not found");
    }


    if (pathname === "/release.json") {
      return handleReleaseJson(ctx);
    }

    const filename = pathname.slice(1);

    if (filename.startsWith("Swiflow_latest_")) {
      return handleLatestRedirect(filename, ctx, request);
    }

    const version = extractVersionFromFilename(filename);
    if (!filename || !version) {
      return new Response("Invalid filename or missing version", { status: 400 });
    }

    const isCN = request.headers.get("cf-ipcountry") === "CN";
    const githubURL = `https://github.com/${GITHUB_REPO}/releases/download/v${version}/${filename}`;
    const r2URL = `${R2_DOMAIN}/${filename}`;
    const target = isCN ? r2URL : githubURL;
    return Response.redirect(target, 302);
  }
};

async function handleReleaseJson(ctx) {
  const cache = caches.default;
  const cacheKey = new Request("https://dl.swiflow.cc/release.json");
  const cached = await cache.match(cacheKey);
  if (cached) return cached;

  const githubApi = `https://api.github.com/repos/${GITHUB_REPO}/releases/latest`;
  const res = await fetch(githubApi, {
    headers: {
      "User-Agent": "Swiflow-Worker",
      "Accept": "application/vnd.github+json"
    }
  });

  if (!res.ok) {
    return new Response("Failed to fetch release info", { status: 502 });
  }

  const data = await res.json();
  const result = {
    tag: data.tag_name.replace(/^v/, ""),
    time: data.published_at,
    assets: data.assets.map(a => ({
      name: a.name, size: a.size,
      hash: a.digest, type: a.content_type,
      url: `https://dl.swiflow.cc/${a.name}`
    })),
    body: data.body,
  };

  const response = new Response(JSON.stringify(result), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600"
    }
  });

  ctx.waitUntil(cache.put(cacheKey, response.clone()));
  return response;
}

async function handleLatestRedirect(filename, ctx, request) {
  const cache = caches.default;
  const cacheKey = new Request("https://dl.swiflow.cc/release.json");
  let json;

  // 尝试从缓存读取 release.json
  const cached = await cache.match(cacheKey);
  if (cached) {
    json = await cached.json();
  } else {
    const response = await handleReleaseJson(ctx);
    json = await response.json();
  }

  const matchedAsset = json.assets.find(a => {
    if (a.name.endsWith(filename.replace("latest", json.tag))) {
      return true
    }
    if (filename.endsWith('x64.dmg') && a.name.includes('amd64.dmg')) {
      return true
    }
    if (filename.includes('aarch64.dmg') && a.name.includes('arm64.dmg')) {
      return true
    }
    if (filename.includes('setup.exe') && a.name.includes('installer.exe')) {
      return true
    }
    return false
  });
  if (!matchedAsset) {
    return new Response("No matching latest asset", { status: 404 });
  }

  const realFilename = matchedAsset.name;
  const isCN = request.headers.get("cf-ipcountry") === "CN";
  const targetURL = isCN
    ? `${R2_DOMAIN}/release-assets/${realFilename}`
    : `https://github.com/${GITHUB_REPO}/releases/download/v${json.tag}/${realFilename}`;

  return Response.redirect(targetURL, 302);
}
