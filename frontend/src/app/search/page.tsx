import type { Metadata } from "next";
import { SearchResults } from "@/components/search-results";
import { isProvider, type Provider } from "@/lib/resources";

export const metadata: Metadata = {
  title: "搜索结果 - 聚优盘",
  description: "查看聚优盘聚合的网盘资源搜索结果。",
};

type SearchPageProps = {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

function firstValue(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
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

  return (
    <SearchResults
      key={`${query}:${provider}:${page}`}
      initialQuery={query}
      initialProvider={provider}
      initialPage={page}
    />
  );
}
