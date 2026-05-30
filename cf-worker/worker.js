// Cloudflare Worker: proxy for api.telegram.org.
//
// Deploy:
//   1. npm i -g wrangler
//   2. wrangler login
//   3. wrangler secret put BOT_TOKEN   (paste your token; used only as an allowlist check)
//   4. wrangler deploy
//
// Then point the bot at the worker:
//   TELEGRAM_API_BASE_URL=https://<your-worker>.workers.dev

const TELEGRAM_HOST = "https://api.telegram.org";

export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    if (env.BOT_TOKEN) {
      const allowed = `/bot${env.BOT_TOKEN}/`;
      const allowedFile = `/file/bot${env.BOT_TOKEN}/`;
      if (!url.pathname.startsWith(allowed) && !url.pathname.startsWith(allowedFile)) {
        return new Response("forbidden", { status: 403 });
      }
    }

    const target = TELEGRAM_HOST + url.pathname + url.search;

    const headers = new Headers(request.headers);
    headers.delete("host");
    headers.delete("cf-connecting-ip");
    headers.delete("cf-ray");
    headers.delete("cf-visitor");
    headers.delete("x-forwarded-for");
    headers.delete("x-forwarded-proto");
    headers.delete("x-real-ip");

    const init = {
      method: request.method,
      headers,
      redirect: "follow",
    };
    if (!["GET", "HEAD"].includes(request.method)) {
      init.body = request.body;
    }

    const upstream = await fetch(target, init);
    return new Response(upstream.body, {
      status: upstream.status,
      statusText: upstream.statusText,
      headers: upstream.headers,
    });
  },
};
