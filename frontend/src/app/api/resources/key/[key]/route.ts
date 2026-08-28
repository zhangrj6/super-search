import type { ApiEnvelope, ResourceGroupResponse } from "@/lib/resources";
import { fetchUrldb } from "@/lib/urldb-server";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ key: string }> },
) {
  const { key } = await params;
  if (!/^[A-Za-z0-9_-]{1,128}$/.test(key)) {
    return Response.json(
      { success: false, message: "资源地址无效", data: null, code: 400 },
      { status: 400 },
    );
  }

  try {
    const upstream = await fetchUrldb(`/resources/key/${encodeURIComponent(key)}`);
    const payload = (await upstream.json()) as ApiEnvelope<ResourceGroupResponse>;
    if (!upstream.ok || !payload.success) {
      return Response.json(payload, { status: upstream.status });
    }

    const data = {
      ...payload.data,
      resources: payload.data.resources.map((resource) => ({
        ...resource,
        url: "",
        save_url: "",
        error_msg: "",
      })),
    };
    return Response.json(
      { ...payload, data },
      { headers: { "cache-control": "no-store" } },
    );
  } catch {
    return Response.json(
      { success: false, message: "资源详情暂时不可用", data: null, code: 502 },
      { status: 502, headers: { "cache-control": "no-store" } },
    );
  }
}
