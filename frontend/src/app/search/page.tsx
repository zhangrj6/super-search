import type { Metadata } from "next";
import { cache } from "react";
import { SearchResults } from "@/components/search-results";
import {
  absoluteUrl,
  SITE_DESCRIPTION,
  SITE_NAME,
  truncateMetaText,
} from "@/lib/seo";
import {
  isProvider,
  PROVIDER_CODES,
  type ApiEnvelope,
  type MelostSearchResponse,
  type Provider,
} from "@/lib/resources";
import { fetchUrldb } from "@/lib/urldb-server";

type SearchPageProps = {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

function firstValue(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
}

function searchPath(query: string, provider: Provider, page: number) {
  const params = new URLSearchParams();
  if (query) params.set("q", query);
  if (provider !== "全部") params.set("provider", provider);
  if (page > 1) params.set("page", String(page));
  const search = params.toString();
  return search ? `/search?${search}` : "/search";
}

const loadSearchData = cache(async (query: string, provider: Provider, page: number) => {
  if (!query) return null;

  try {
    const response = await fetchUrldb("/search", {
      method: "POST",
      body: JSON.stringify({
        q: query,
        type: PROVIDER_CODES[provider],
        page,
        size: 20,
      }),
    });
    if (!response.ok) return null;

    const payload = (await response.json()) as ApiEnvelope<MelostSearchResponse>;
    return payload.success ? payload.data : null;
  } catch {
    return null;
  }
});

export async function generateMetadata({ searchParams }: SearchPageProps): Promise<Metadata> {
  const params = await searchParams;
  const query = (firstValue(params.q) ?? "").trim().slice(0, 100);
  const providerValue = firstValue(params.provider);
  const provider: Provider = isProvider(providerValue) ? providerValue : "全部";
  const requestedPage = Number.parseInt(firstValue(params.page) ?? "1", 10);
  const page = Number.isFinite(requestedPage)
    ? Math.min(500, Math.max(1, requestedPage))
    : 1;
  const canonicalPath = searchPath(query, provider, page);
  const queryText = truncateMetaText(query, 72);
  const title = queryText
    ? `${queryText} 网盘搜索结果｜${SITE_NAME}`
    : `网盘资源搜索｜${SITE_NAME}`;
  const description = queryText
    ? `在聚优盘搜索“${queryText}”相关的网盘资源，覆盖百度网盘、夸克网盘、阿里云盘、迅雷云盘和 UC 网盘。`
    : SITE_DESCRIPTION;
  const indexable = Boolean(queryText) && page === 1;

  return {
    title,
    description,
    alternates: { canonical: canonicalPath },
    robots: {
      index: indexable,
      follow: true,
      googleBot: {
        index: indexable,
        follow: true,
        "max-image-preview": "large",
        "max-snippet": -1,
        "max-video-preview": -1,
      },
    },
    openGraph: {
      type: "website",
      url: absoluteUrl(canonicalPath),
      siteName: SITE_NAME,
      title,
      description,
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  };
}

export default async function SearchPage({ searchParams }: SearchPageProps) {
  const params = await searchParams;
  const query = (firstValue(params.q) ?? "").trim().slice(0, 100);
  const providerValue = firstValue(params.provider);
  const provider: Provider = isProvider(providerValue) ? providerValue : "全部";
  const requestedPage = Number.parseInt(firstValue(params.page) ?? "1", 10);
  const page = Number.isFinite(requestedPage)
    ? Math.min(500, Math.max(1, requestedPage))
    : 1;
  const initialData = await loadSearchData(query, provider, page);

  return (
    <SearchResults
      key={`${query}:${provider}:${page}`}
      initialQuery={query}
      initialProvider={provider}
      initialPage={page}
      initialData={initialData}
    />
  );
}
