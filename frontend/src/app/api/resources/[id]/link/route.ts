import type { ApiEnvelope, ResourceLinkResponse } from "@/lib/resources";
import { fetchUrldb } from "@/lib/urldb-server";

function isSafeTransferredLink(data: ResourceLinkResponse | undefined) {
  if (!data || data.type !== "transferred") return false;
  try {
    return new URL(data.url).protocol === "https:";
  } catch {
    return false;
  }
}

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id } = await params;
  if (!/^\d+$/.test(id)) {
    return Response.json(
      { success: false, message: "资源编号无效", data: null, code: 400 },
      { status: 400 },
    );
  }

  try {
    const upstream = await fetchUrldb(`/resources/${id}/link`);
    if (!upstream.ok) {
      return Response.json(
        { success: false, message: "获取链接失败，请稍后重试", data: null, code: 502 },
        { status: 502, headers: { "cache-control": "no-store" } },
      );
    }

    const payload = (await upstream.json()) as ApiEnvelope<ResourceLinkResponse>;
    if (!payload.success || !isSafeTransferredLink(payload.data)) {
      return Response.json(
        { success: false, message: "获取链接失败，请稍后重试", data: null, code: 502 },
        { status: 502, headers: { "cache-control": "no-store" } },
      );
    }

    return Response.json(
      {
        success: true,
        message: "已生成新的分享链接",
        data: {
          url: payload.data.url,
          type: "transferred",
          platform: payload.data.platform,
          resource_id: payload.data.resource_id,
          message: payload.data.message,
        },
        code: 200,
      },
      { headers: { "cache-control": "no-store" } },
    );
  } catch {
    return Response.json(
      { success: false, message: "获取链接失败，请稍后重试", data: null, code: 502 },
      { status: 502, headers: { "cache-control": "no-store" } },
    );
  }
}
