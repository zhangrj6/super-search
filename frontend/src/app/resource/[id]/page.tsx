import { AlertCircle } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { cache } from "react";
import { JsonLd } from "@/components/json-ld";
import { ResourceDetail } from "@/components/resource-detail";
import { Button } from "@/components/ui/button";
import {
  DEFAULT_PROVIDER,
  DEFAULT_SEARCH_MODE,
  isSearchMode,
  isProvider,
  resourceTagNames,
  sanitizeResourceDisplayText,
  type ApiEnvelope,
  type Provider,
  type ResourceGroupResponse,
  type SearchMode,
  type UrldbResource,
} from "@/lib/resources";
import {
  absoluteUrl,
  SITE_NAME,
  truncateMetaText,
} from "@/lib/seo";
import { fetchUrldb } from "@/lib/urldb-server";

type ResourcePageProps = {
  params: Promise<{ id: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

function firstValue(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
}

const loadResource = cache(async (key: string) => {
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
});

export async function generateMetadata({ params }: ResourcePageProps): Promise<Metadata> {
  const { id: key } = await params;
  if (!/^[A-Za-z0-9_-]{1,128}$/.test(key)) {
    return {
      title: "资源不存在",
      robots: { index: false, follow: false },
    };
  }

  const result = await loadResource(key);
  if (result.status !== "ready") {
    return {
      title: result.status === "not-found" ? "资源不存在" : "资源详情暂时不可用",
      description: "聚优盘网盘资源详情页面。",
      robots: { index: false, follow: false },
    };
  }

  const resourceTitle = truncateMetaText(
    sanitizeResourceDisplayText(result.resource.title) || "网盘资源",
    72,
  );
  const resourceDescription = truncateMetaText(
    sanitizeResourceDisplayText(result.resource.description) ||
      `在聚优盘查看${resourceTitle}相关的网盘资源信息。`,
  );
  const canonicalPath = `/resource/${encodeURIComponent(key)}`;
  const title = `${resourceTitle}｜网盘资源｜${SITE_NAME}`;

  return {
    title,
    description: resourceDescription,
    alternates: { canonical: canonicalPath },
    robots: {
      index: true,
      follow: true,
      googleBot: {
        index: true,
        follow: true,
        "max-image-preview": "large",
        "max-snippet": -1,
        "max-video-preview": -1,
      },
    },
    openGraph: {
      type: "article",
      url: absoluteUrl(canonicalPath),
      siteName: SITE_NAME,
      title,
      description: resourceDescription,
      publishedTime: result.resource.created_at,
      modifiedTime: result.resource.updated_at,
    },
    twitter: {
      card: "summary",
      title,
      description: resourceDescription,
    },
  };
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
  const provider: Provider = isProvider(providerValue) ? providerValue : DEFAULT_PROVIDER;
  const modeValue = firstValue(currentSearch.mode);
  const mode: SearchMode = isSearchMode(modeValue) ? modeValue : DEFAULT_SEARCH_MODE;
  const requestedPage = Number.parseInt(firstValue(currentSearch.page) ?? "1", 10);
  const page = Number.isFinite(requestedPage) ? Math.min(500, Math.max(1, requestedPage)) : 1;

  let backHref = "/";
  if (query) {
    const backParams = new URLSearchParams({ q: query, provider });
    if (mode !== DEFAULT_SEARCH_MODE) backParams.set("mode", mode);
    if (page > 1) backParams.set("page", String(page));
    backHref = `/search?${backParams.toString()}`;
  }

  const result = await loadResource(key);
  if (result.status === "not-found") notFound();
  if (result.status === "unavailable") return <ResourceUnavailable backHref={backHref} />;

  const displayTitle = sanitizeResourceDisplayText(result.resource.title) || "网盘资源";
  const displayDescription = sanitizeResourceDisplayText(result.resource.description);
  const canonicalUrl = absoluteUrl(`/resource/${encodeURIComponent(key)}`);
  const resourceJsonLd: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "CreativeWork",
    name: displayTitle,
    description: displayDescription || `在${SITE_NAME}查看${displayTitle}相关的网盘资源信息。`,
    url: canonicalUrl,
    mainEntityOfPage: canonicalUrl,
    inLanguage: "zh-CN",
    isAccessibleForFree: true,
    datePublished: result.resource.created_at,
    dateModified: result.resource.updated_at,
    keywords: resourceTagNames(result.resource),
    publisher: {
      "@type": "Organization",
      name: SITE_NAME,
      url: absoluteUrl("/"),
    },
  };

  return (
    <>
      <JsonLd data={resourceJsonLd} />
      <ResourceDetail resource={result.resource} backHref={backHref} />
    </>
  );
}
