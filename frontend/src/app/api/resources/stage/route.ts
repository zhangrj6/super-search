import { forwardUrldbJson } from "@/lib/urldb-server";

export async function POST(request: Request) {
  let body: string;
  try {
    body = JSON.stringify(await request.json());
  } catch {
    return Response.json(
      { success: false, message: "资源参数无效", data: null, code: 400 },
      { status: 400 },
    );
  }

  return forwardUrldbJson(
    "/resources/stage",
    { method: "POST", body },
    "保存资源失败，请稍后重试",
  );
}
