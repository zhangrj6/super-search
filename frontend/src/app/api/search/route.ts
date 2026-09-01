import { forwardUrldbJson } from "@/lib/urldb-server";
import { DEFAULT_SEARCH_MODE, PROVIDER_CODES, isSearchMode } from "@/lib/resources";

const SEARCH_PROVIDER_CODES = new Set(Object.values(PROVIDER_CODES));

export async function POST(request: Request) {
  let payload: Record<string, unknown>;
  try {
    const value = await request.json();
    if (typeof value !== "object" || value == null || Array.isArray(value)) throw new Error();
    payload = value as Record<string, unknown>;
  } catch {
    return Response.json(
      { success: false, message: "搜索参数无效", data: null, code: 400 },
      { status: 400 },
    );
  }

  const searchType = typeof payload.search_type === "string"
    ? payload.search_type.trim().toLowerCase()
    : DEFAULT_SEARCH_MODE;
  if (!isSearchMode(searchType)) {
    return Response.json(
      { success: false, message: "仅支持资源和视频搜索", data: null, code: 400 },
      { status: 400 },
    );
  }

  const providerCode = typeof payload.type === "string" ? payload.type.trim().toUpperCase() : "";
  if (!SEARCH_PROVIDER_CODES.has(providerCode)) {
    return Response.json(
      { success: false, message: "仅支持搜索夸克网盘和迅雷云盘", data: null, code: 400 },
      { status: 400 },
    );
  }

  const body = JSON.stringify({ ...payload, type: providerCode, search_type: searchType });

  return forwardUrldbJson(
    "/search",
    { method: "POST", body },
    "搜索服务暂时不可用，请稍后重试",
  );
}
