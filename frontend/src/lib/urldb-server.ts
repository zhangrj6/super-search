import "server-only";

const DEFAULT_API_BASE = "https://52juyou.com/api";

function getApiBase() {
  const rawBase = process.env.URLDB_API_BASE?.trim() || DEFAULT_API_BASE;
  const base = new URL(rawBase);
  if (base.protocol !== "http:" && base.protocol !== "https:") {
    throw new Error("URLDB_API_BASE must use http or https");
  }
  return base.toString().replace(/\/$/, "");
}

export function fetchUrldb(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers);
  headers.set("accept", "application/json");
  if (init.body != null) headers.set("content-type", "application/json");

  return fetch(`${getApiBase()}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });
}

export async function forwardUrldbJson(
  path: string,
  init: RequestInit,
  failureMessage: string,
) {
  try {
    const upstream = await fetchUrldb(path, init);
    const body = await upstream.text();
    return new Response(body, {
      status: upstream.status,
      headers: {
        "cache-control": "no-store",
        "content-type": "application/json; charset=utf-8",
      },
    });
  } catch {
    return Response.json(
      { success: false, message: failureMessage, data: null, code: 502 },
      { status: 502, headers: { "cache-control": "no-store" } },
    );
  }
}
