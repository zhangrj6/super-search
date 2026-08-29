import { AlertCircle } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ResourceDetail } from "@/components/resource-detail";
import { Button } from "@/components/ui/button";
import {
  type ApiEnvelope,
  isProvider,
  type Provider,
  type ResourceGroupResponse,
  type UrldbResource,
} from "@/lib/resources";
import { fetchUrldb } from "@/lib/urldb-server";

export const metadata: Metadata = {
  title: "资源详情 - 聚优盘",
  description: "查看网盘资源详情并获取新的分享链接。",
};

type ResourcePageProps = {
  params: Promise<{ id: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

function firstValue(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
}

async function loadResource(key: string) {
  try {
    const response = await fetchUrldb(`/resources/key/${encodeURIComponent(key)}`);
    if (response.status === 404) return { status: "not-found" as const };
    if (!response.ok) return { status: "unavailable" as const };

    const payload = (await response.json()) as ApiEnvelope<ResourceGroupResponse>;
    const resource = payload.success ? payload.data?.resources?.[0] : undefined;
    if (!resource) return { status: "not-found" as const };

    const sanitizedResource: UrldbResource = {
      ...resource,
      url: "",
      save_url: "",
      error_msg: "",
    };
    return { status: "ready" as const, resource: sanitizedResource };
  } catch {
    return { status: "unavailable" as const };
  }
}

function ResourceUnavailable({ backHref }: { backHref: string }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-transparent px-5 text-center text-foreground">
      <div className="flex w-full max-w-md flex-col items-center rounded-2xl border border-border bg-card px-6 py-12 shadow-[0_2px_16px_rgb(17_24_39_/_0.04)]">
        <span className="flex size-14 items-center justify-center rounded-2xl bg-red-50 text-red-600">
          <AlertCircle className="size-6" aria-hidden="true" />
        </span>
        <h1 className="mt-5 text-xl font-semibold">资源详情暂时不可用</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">服务连接失败，请稍后再试。</p>
        <Button className="mt-6 h-11 px-4" asChild>
          <Link href={backHref}>返回搜索结果</Link>
        </Button>
      </div>
    </main>
  );
}

export default async function ResourcePage({ params, searchParams }: ResourcePageProps) {
  const { id: key } = await params;
  if (!/^[A-Za-z0-9_-]{1,128}$/.test(key)) notFound();

  const currentSearch = await searchParams;
  const query = (firstValue(currentSearch.q) ?? "").trim().slice(0, 100);
  const providerValue = firstValue(currentSearch.provider);
  const provider: Provider = isProvider(providerValue) ? providerValue : "全部";
  const requestedPage = Number.parseInt(firstValue(currentSearch.page) ?? "1", 10);
  const page = Number.isFinite(requestedPage) ? Math.min(500, Math.max(1, requestedPage)) : 1;

  let backHref = "/";
  if (query) {
    const backParams = new URLSearchParams({ q: query });
    if (provider !== "全部") backParams.set("provider", provider);
    if (page > 1) backParams.set("page", String(page));
    backHref = `/search?${backParams.toString()}`;
  }

  const result = await loadResource(key);
  if (result.status === "not-found") notFound();
  if (result.status === "unavailable") return <ResourceUnavailable backHref={backHref} />;

  return <ResourceDetail resource={result.resource} backHref={backHref} />;
}
