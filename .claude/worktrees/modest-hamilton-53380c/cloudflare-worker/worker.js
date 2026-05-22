const ALLOWED_HOSTS = [
  "www.microsoft.com",
  "vlscppe.microsoft.com",
  "ov-df.microsoft.com",
];

export default {
  async fetch(request) {
    const url = new URL(request.url);

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

    const upstreamRequest = new Request(url.toString(), {
      method: request.method,
      headers: {
        "User-Agent": request.headers.get("User-Agent") || "Mozilla/5.0",
        "Referer": request.headers.get("Referer") || "",
        "Accept": request.headers.get("Accept") || "application/json",
      },
    });

    const upstream = await fetch(upstreamRequest);

    return new Response(upstream.body, {
      status: upstream.status,
      headers: {
        "Content-Type": upstream.headers.get("Content-Type") || "application/json",
        "Access-Control-Allow-Origin": "*",
      },
    });
  },
};
