const ALLOWED_HOSTS = [
  "www.microsoft.com",
  "vlscppe.microsoft.com",
  "ov-df.microsoft.com",
];

export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    if (env.CF_WORKER_SECRET) {
      const secret = request.headers.get("X-Worker-Secret");
      if (secret !== env.CF_WORKER_SECRET) {
        return new Response(JSON.stringify({ error: "Forbidden" }), {
          status: 403,
          headers: { "Content-Type": "application/json" },
        });
      }
    }

    // Target host comes from the ?host= param; path+search are forwarded as-is
    const targetHost = url.searchParams.get("host");
    if (!targetHost || !ALLOWED_HOSTS.includes(targetHost)) {
      return new Response(JSON.stringify({ error: "Invalid or missing host parameter" }), {
        status: 400,
        headers: { "Content-Type": "application/json" },
      });
    }

    url.searchParams.delete("host");
    url.hostname = targetHost;
    url.protocol = "https:";
    url.port = "";

    const forwardHeaders = {
      "User-Agent": request.headers.get("User-Agent") || "Mozilla/5.0",
      "Referer": request.headers.get("Referer") || "",
      "Accept": request.headers.get("Accept") || "application/json",
    };
    const cookie = request.headers.get("Cookie");
    if (cookie) forwardHeaders["Cookie"] = cookie;

    const upstreamRequest = new Request(url.toString(), {
      method: request.method,
      headers: forwardHeaders,
    });

    const upstream = await fetch(upstreamRequest);

    const responseHeaders = new Headers();
    responseHeaders.set("Content-Type", upstream.headers.get("Content-Type") || "application/json");
    responseHeaders.set("Access-Control-Allow-Origin", "*");

    // Forward Set-Cookie headers so the backend's cookie jar can accumulate them
    const setCookies = upstream.headers.getAll
      ? upstream.headers.getAll("set-cookie")
      : (upstream.headers.get("set-cookie") ? [upstream.headers.get("set-cookie")] : []);
    for (const c of setCookies) responseHeaders.append("Set-Cookie", c);

    return new Response(upstream.body, {
      status: upstream.status,
      headers: responseHeaders,
    });
  },
};
