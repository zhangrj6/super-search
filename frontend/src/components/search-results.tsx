"use client";

import {
  AlertCircle,
  Archive,
  BookOpenText,
  ChevronLeft,
  ChevronRight,
  FileText,
  Film,
  LoaderCircle,
  Music2,
  RefreshCw,
  Search,
  SearchX,
  X,
} from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState, useTransition } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  formatBytes,
  inferResourceType,
  PROVIDER_CODES,
  PROVIDER_VALUES,
  providerFromDiskType,
  readApiData,
  type DriveProvider,
  type MelostSearchItem,
  type MelostSearchResponse,
  type MelostStageResponse,
  type Provider,
  type ResourceType,
} from "@/lib/resources";

const PROVIDER_STYLES: Record<DriveProvider, string> = {
  百度网盘: "bg-blue-50 text-blue-600",
  阿里云盘: "bg-orange-50 text-orange-600",
  夸克网盘: "bg-violet-50 text-violet-600",
  迅雷云盘: "bg-red-50 text-red-600",
  UC网盘: "bg-emerald-50 text-emerald-700",
};

const TYPE_STYLES: Record<ResourceType, string> = {
  课程: "bg-cyan-50 text-cyan-700",
  视频: "bg-violet-50 text-violet-600",
  音频: "bg-emerald-50 text-emerald-700",
  文档: "bg-blue-50 text-blue-600",
  素材: "bg-orange-50 text-orange-600",
};

type StageState = { status: "opening" | "failed"; error?: string };

function ResourceIcon({ type }: { type: ResourceType }) {
  const className = "size-5";
  if (type === "课程") return <BookOpenText className={className} />;
  if (type === "视频") return <Film className={className} />;
  if (type === "音频") return <Music2 className={className} />;
  if (type === "素材") return <Archive className={className} />;
  return <FileText className={className} />;
}

function safeErrorMessage(error: unknown, fallback: string) {
  const message = error instanceof Error ? error.message.trim() : "";
  return !message || /https?:\/\//i.test(message) ? fallback : message;
}

type SearchResultsProps = {
  initialQuery: string;
  initialProvider: Provider;
  initialPage: number;
};

export function SearchResults({
  initialQuery,
  initialProvider,
  initialPage,
}: SearchResultsProps) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState(initialQuery);
  const [provider, setProvider] = useState(initialProvider);
  const [searchData, setSearchData] = useState<MelostSearchResponse | null>(null);
  const [searchError, setSearchError] = useState("");
  const [isLoading, setIsLoading] = useState(Boolean(initialQuery));
  const [reloadSequence, setReloadSequence] = useState(0);
  const [stageStates, setStageStates] = useState<Record<string, StageState>>({});
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    if (!initialQuery) {
      setSearchData(null);
      setIsLoading(false);
      return;
    }

    const controller = new AbortController();
    setIsLoading(true);
    setSearchError("");
    setStageStates({});

    void fetch("/api/melost/search", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        q: initialQuery,
        type: PROVIDER_CODES[initialProvider],
        page: initialPage,
        size: 20,
      }),
      signal: controller.signal,
    })
      .then(readApiData<MelostSearchResponse>)
      .then((data) => setSearchData(data))
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setSearchData(null);
        setSearchError(safeErrorMessage(error, "搜索服务暂时不可用，请稍后重试"));
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false);
      });

    return () => controller.abort();
  }, [initialPage, initialProvider, initialQuery, reloadSequence]);

  function buildSearchUrl(nextQuery: string, nextProvider: Provider, nextPage = 1) {
    const params = new URLSearchParams({ q: nextQuery });
    if (nextProvider !== "全部") params.set("provider", nextProvider);
    if (nextPage > 1) params.set("page", String(nextPage));
    return `/search?${params.toString()}`;
  }

  function buildResourceUrl(key: string) {
    const params = new URLSearchParams({ q: initialQuery });
    if (provider !== "全部") params.set("provider", provider);
    if (initialPage > 1) params.set("page", String(initialPage));
    return `/resource/${encodeURIComponent(key)}?${params.toString()}`;
  }

  function submitSearch(value: string) {
    const nextQuery = value.trim();
    if (!nextQuery) {
      inputRef.current?.focus();
      return;
    }
    startTransition(() => router.push(buildSearchUrl(nextQuery, provider)));
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    submitSearch(query);
  }

  function handleProviderChange(value: string) {
    const nextProvider = value as Provider;
    setProvider(nextProvider);
    startTransition(() => {
      router.replace(buildSearchUrl(initialQuery, nextProvider), { scroll: false });
    });
  }

  function changePage(nextPage: number) {
    startTransition(() => {
      router.push(buildSearchUrl(initialQuery, provider, nextPage));
    });
  }

  async function stageAndOpen(item: MelostSearchItem) {
    const itemKey = item.doc_id || item.link;
    if (!item.can_stage || stageStates[itemKey]?.status === "opening") return;

    setStageStates((current) => ({ ...current, [itemKey]: { status: "opening" } }));
    try {
      const response = await fetch("/api/melost/resources", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          doc_id: item.doc_id,
          title: item.disk_name,
          link: item.link,
          disk_type: item.disk_type,
          disk_pass: item.disk_pass,
          files: item.files,
          tags: item.tags,
          shared_time: item.shared_time,
          share_user: item.share_user,
          size: item.size,
        }),
      });
      const data = await readApiData<MelostStageResponse>(response);
      if (!/^[A-Za-z0-9_-]{1,128}$/.test(data.resource_key)) {
        throw new Error("资源详情地址生成失败");
      }
      router.push(buildResourceUrl(data.resource_key));
    } catch (error) {
      setStageStates((current) => ({
        ...current,
        [itemKey]: {
          status: "failed",
          error: safeErrorMessage(error, "保存资源失败，请重试"),
        },
      }));
    }
  }

  const results = searchData?.items ?? [];
  const total = searchData?.total ?? 0;
  const pageSize = searchData?.page_size || 20;
  const totalPages = Math.max(1, Math.min(500, Math.ceil(total / pageSize)));

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <a
        className="fixed top-3 left-3 z-50 -translate-y-20 rounded-lg bg-foreground px-3 py-2 text-sm font-medium text-background transition-transform focus:translate-y-0"
        href="#result-list"
      >
        跳到搜索结果
      </a>

      <header className="sticky top-0 z-20 border-b border-border bg-background/90 backdrop-blur-xl">
        <div className="mx-auto grid max-w-5xl grid-cols-[auto_1fr_auto] items-center gap-3 px-5 py-3 sm:gap-4 sm:px-8">
          <Link
            className="flex min-h-10 items-center gap-2.5 rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/25"
            href="/"
            aria-label="返回 PanSearch 首页"
          >
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground shadow-[0_3px_10px_rgb(79_110_247_/_0.24)]">
              P
            </span>
            <span className="hidden text-sm font-semibold sm:inline">PanSearch</span>
          </Link>

          <form
            className="col-span-3 row-start-2 min-w-0 sm:col-span-1 sm:row-start-auto"
            onSubmit={handleSubmit}
          >
            <label className="sr-only" htmlFor="results-query">
              重新搜索资源
            </label>
            <div className="group flex h-11 items-center gap-2 rounded-xl border border-border bg-card pr-1.5 pl-3.5 shadow-[0_1px_6px_rgb(17_24_39_/_0.03)] transition-[border-color,box-shadow] focus-within:border-primary focus-within:shadow-[0_0_0_3px_rgb(79_110_247_/_0.08)]">
              <Search
                className="size-4 shrink-0 text-muted-foreground group-focus-within:text-primary"
                strokeWidth={1.8}
                aria-hidden="true"
              />
              <Input
                ref={inputRef}
                id="results-query"
                className="h-9 min-w-0 border-0 bg-transparent px-0 text-sm shadow-none focus-visible:border-0 focus-visible:ring-0"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                autoComplete="off"
              />
              {query ? (
                <Button
                  className="size-8 rounded-full text-muted-foreground"
                  size="icon"
                  variant="ghost"
                  type="button"
                  onClick={() => {
                    setQuery("");
                    inputRef.current?.focus();
                  }}
                  aria-label="清空搜索框"
                >
                  <X className="size-3.5" aria-hidden="true" />
                </Button>
              ) : null}
              <Button
                className="h-8 min-w-14 rounded-lg px-3 text-xs font-semibold"
                type="submit"
                disabled={isPending}
              >
                {isPending ? (
                  <LoaderCircle className="size-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" />
                ) : (
                  "搜索"
                )}
              </Button>
            </div>
          </form>

          <p className="text-xs whitespace-nowrap text-muted-foreground" role="status" aria-live="polite">
            {isLoading ? (
              "正在搜索"
            ) : (
              <><strong className="font-semibold text-foreground">{total.toLocaleString()}</strong> 个结果</>
            )}
          </p>
        </div>

        <Tabs
          className="w-full min-w-0 overflow-hidden"
          value={provider}
          onValueChange={handleProviderChange}
        >
          <div className="mx-auto w-full min-w-0 max-w-5xl overflow-x-auto px-5 sm:px-8">
            <TabsList
              className="h-11 min-w-max gap-0 bg-transparent p-0"
              variant="line"
              aria-label="按网盘来源筛选"
            >
              {PROVIDER_VALUES.map((item) => (
                <TabsTrigger
                  className="h-10 flex-none px-3.5 text-sm font-normal data-active:font-semibold data-active:text-primary sm:px-4"
                  value={item}
                  key={item}
                  disabled={isPending}
                >
                  {item}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>
        </Tabs>
      </header>

      <main
        id="result-list"
        className="mx-auto w-full max-w-5xl flex-1 scroll-mt-32 px-5 py-7 sm:px-8 sm:py-9"
        aria-busy={isLoading}
      >
        <div className="mb-5 flex items-end justify-between gap-4">
          <div className="min-w-0">
            <p className="text-xs text-muted-foreground">melost.cn 搜索结果</p>
            <h1 className="mt-1 truncate text-xl font-semibold sm:text-2xl">
              {initialQuery ? `“${initialQuery}”` : "搜索网盘资源"}
            </h1>
          </div>
          <span className="hidden shrink-0 text-xs text-muted-foreground sm:block">
            {provider === "全部" ? "全部来源" : provider}
          </span>
        </div>

        {isLoading ? (
          <div className="grid min-w-0 grid-cols-[minmax(0,1fr)] gap-3" role="status" aria-label="正在加载搜索结果">
            {Array.from({ length: 4 }).map((_, index) => (
              <div
                className="h-36 animate-pulse rounded-2xl border border-border bg-card motion-reduce:animate-none"
                key={index}
              />
            ))}
          </div>
        ) : searchError ? (
          <div className="flex min-h-80 flex-col items-center justify-center rounded-2xl border border-border bg-card px-6 text-center shadow-[0_1px_6px_rgb(17_24_39_/_0.03)]" role="alert">
            <AlertCircle className="size-8 text-red-500" aria-hidden="true" />
            <h2 className="mt-4 text-base font-semibold">搜索暂时不可用</h2>
            <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">{searchError}</p>
            <Button className="mt-5 h-11 px-4" variant="outline" type="button" onClick={() => setReloadSequence((value) => value + 1)}>
              <RefreshCw data-icon="inline-start" aria-hidden="true" />
              重新搜索
            </Button>
          </div>
        ) : results.length > 0 ? (
          <div className="grid min-w-0 grid-cols-[minmax(0,1fr)] gap-3">
            {results.map((resource) => {
              const resourceKey = resource.doc_id || resource.link;
              const stageState = stageStates[resourceKey];
              const resourceType = inferResourceType(`${resource.disk_name} ${resource.files}`);
              const resourceProvider = providerFromDiskType(resource.disk_type);
              const isOpening = stageState?.status === "opening";
              const isDisabled = !resource.can_stage || isOpening;

              return (
                <button
                  className={`group min-w-0 max-w-full overflow-hidden rounded-2xl border border-border bg-card text-left shadow-[0_1px_6px_rgb(17_24_39_/_0.03)] outline-none transition-[border-color,box-shadow] focus-visible:border-primary focus-visible:ring-3 focus-visible:ring-ring/20 ${
                    resource.can_stage
                      ? "cursor-pointer hover:border-blue-200 hover:shadow-[0_6px_20px_rgb(17_24_39_/_0.06)]"
                      : "cursor-not-allowed opacity-70"
                  }`}
                  type="button"
                  key={resourceKey}
                  disabled={isDisabled}
                  onClick={() => stageAndOpen(resource)}
                  aria-label={`打开${resource.disk_name}的资源详情`}
                >
                  <article className="flex min-w-0 max-w-full items-start gap-3 p-4 sm:gap-4 sm:p-5">
                    <span
                      className={`flex size-11 shrink-0 items-center justify-center rounded-xl sm:size-12 ${TYPE_STYLES[resourceType]}`}
                      aria-hidden="true"
                    >
                      <ResourceIcon type={resourceType} />
                    </span>
                    <span className="min-w-0 flex-1 overflow-hidden">
                      <span className="block text-sm leading-6 font-semibold [overflow-wrap:anywhere] transition-colors group-hover:text-primary sm:text-base">
                        {resource.disk_name}
                      </span>
                      {resource.files ? (
                        <span className="mt-1 line-clamp-2 block text-sm leading-6 [overflow-wrap:anywhere] text-muted-foreground">
                          {resource.files}
                        </span>
                      ) : null}
                      {resource.tags.length > 0 ? (
                        <span className="mt-3 flex flex-wrap gap-1.5">
                          {resource.tags.slice(0, 4).map((tag) => (
                            <span className="rounded-md bg-muted px-2 py-1 text-[11px] text-muted-foreground" key={tag}>
                              {tag}
                            </span>
                          ))}
                        </span>
                      ) : null}
                      <span className="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground sm:hidden">
                        <span className={`rounded-md px-2 py-1 ${PROVIDER_STYLES[resourceProvider]}`}>
                          {resourceProvider}
                        </span>
                        <span>{formatBytes(resource.size)}</span>
                        {resource.shared_time ? <span>{resource.shared_time.slice(0, 10)}</span> : null}
                      </span>
                      {stageState?.status === "failed" ? (
                        <span className="mt-3 block text-xs leading-5 text-red-600" role="alert">{stageState.error}</span>
                      ) : !resource.can_stage ? (
                        <span className="mt-3 block text-xs leading-5 text-amber-700">{resource.stage_message || "该资源类型暂不支持"}</span>
                      ) : null}
                    </span>
                    <span className="hidden shrink-0 items-center gap-3 sm:flex">
                      <span className="text-right text-xs text-muted-foreground">
                        <span className={`inline-flex rounded-lg px-2.5 py-1 font-medium ${PROVIDER_STYLES[resourceProvider]}`}>
                          {resourceProvider}
                        </span>
                        <span className="mt-2 block tabular-nums">
                          {formatBytes(resource.size)}{resource.shared_time ? ` · ${resource.shared_time.slice(0, 10)}` : ""}
                        </span>
                      </span>
                      <span className="flex size-11 items-center justify-center rounded-lg text-muted-foreground transition-colors group-hover:bg-blue-50 group-hover:text-primary">
                        {isOpening ? (
                          <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
                        ) : (
                          <ChevronRight className="size-4" aria-hidden="true" />
                        )}
                      </span>
                    </span>
                  </article>
                </button>
              );
            })}
          </div>
        ) : (
          <div className="flex min-h-80 flex-col items-center justify-center rounded-2xl border border-border bg-card px-6 text-center shadow-[0_1px_6px_rgb(17_24_39_/_0.03)]">
            <SearchX className="size-8 text-muted-foreground" aria-hidden="true" />
            <h2 className="mt-4 text-base font-semibold">没有找到匹配资源</h2>
            <p className="mt-2 text-sm text-muted-foreground">换个关键词或选择“全部”来源再试试。</p>
            <Button className="mt-5 h-10 px-4" variant="outline" asChild>
              <Link href="/">返回首页</Link>
            </Button>
          </div>
        )}

        {!isLoading && !searchError && totalPages > 1 ? (
          <nav className="mt-6 flex items-center justify-between gap-4 border-t border-border pt-5" aria-label="搜索结果分页">
            <Button className="h-11 px-4" variant="outline" type="button" disabled={initialPage <= 1 || isPending} onClick={() => changePage(initialPage - 1)}>
              <ChevronLeft data-icon="inline-start" aria-hidden="true" />
              上一页
            </Button>
            <span className="text-xs tabular-nums text-muted-foreground">第 {initialPage} / {totalPages} 页</span>
            <Button className="h-11 px-4" variant="outline" type="button" disabled={initialPage >= totalPages || isPending} onClick={() => changePage(initialPage + 1)}>
              下一页
              <ChevronRight data-icon="inline-end" aria-hidden="true" />
            </Button>
          </nav>
        ) : null}
      </main>

      <footer className="px-5 py-5 text-center text-xs text-muted-foreground sm:px-8">
        搜索结果由 melost.cn 提供 · 请尊重版权并遵守相关法律法规
      </footer>
    </div>
  );
}
