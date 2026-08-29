import { forwardUrldbJson } from "@/lib/urldb-server";

export async function POST(request: Request) {
  let body: string;
  try {
    body = JSON.stringify(await request.json());
  } catch {
    return Response.json(
      { success: false, message: "搜索参数无效", data: null, code: 400 },
      { status: 400 },
    );
  }

  return forwardUrldbJson(
    "/search",
    { method: "POST", body },
    "搜索服务暂时不可用，请稍后重试",
  );
}
