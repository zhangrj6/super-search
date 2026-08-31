"use client";

import {
  AlertCircle,
  ArrowDown,
  ArrowDownUp,
  ArrowUp,
  Archive,
  BookOpenText,
  ChevronLeft,
  ChevronRight,
  FileText,
  Film,
  Link2,
  LoaderCircle,
  Music2,
  RefreshCw,
  RotateCcw,
  Search,
  SearchX,
  X,
} from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState, useTransition } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { SearchLoadingOverlay } from "@/components/search-loading-overlay";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  DEFAULT_SEARCH_MODE,
  inferResourceType,
  PROVIDER_CODES,
  PROVIDER_VALUES,
  SEARCH_MODE_VALUES,
  providerFromDiskType,
  readApiData,
  sanitizeResourceDisplayMessage,
  sanitizeResourceDisplayText,
  type DriveProvider,
  type MelostSearchItem,
  type MelostSearchResponse,
  type MelostStageResponse,
  type Provider,
  type ResourceType,
  type SearchMode,
} from "@/lib/resources";
import logo from "@/assets/logo.png";

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
type DateSort = "default" | "desc" | "asc";

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
  if (!message || /https?:\/\//i.test(message)) return fallback;
  return sanitizeResourceDisplayMessage(message) || fallback;
}

type SearchResultsProps = {
  initialQuery: string;
  initialProvider: Provider;
  initialMode: SearchMode;
  initialPage: number;
  initialData: MelostSearchResponse | null;
};

export function SearchResults({
  initialQuery,
  initialProvider,
  initialMode,
  initialPage,
  initialData,
}: SearchResultsProps) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState(initialQuery);
  const [provider, setProvider] = useState(initialProvider);
  const [mode, setMode] = useState<SearchMode>(initialMode);
  const [searchData, setSearchData] = useState<MelostSearchResponse | null>(initialData);
  const [searchError, setSearchError] = useState("");
  const [isLoading, setIsLoading] = useState(Boolean(initialQuery && !initialData));
  const [reloadSequence, setReloadSequence] = useState(0);
  const [stageStates, setStageStates] = useState<Record<string, StageState>>({});
  const [dateSort, setDateSort] = useState<DateSort>("default");
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    if (!initialQuery) {
      setSearchData(null);
      setIsLoading(false);
      return;
    }

    if (initialData && reloadSequence === 0) {
      setSearchData(initialData);
      setIsLoading(false);
      return;
    }

    const controller = new AbortController();
    setIsLoading(true);
    setSearchError("");
    setStageStates({});

    void fetch("/api/search", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        q: initialQuery,
        type: PROVIDER_CODES[initialProvider],
        search_type: initialMode,
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
  }, [initialData, initialMode, initialPage, initialProvider, initialQuery, reloadSequence]);

  function buildSearchUrl(nextQuery: string, nextProvider: Provider, nextMode: SearchMode, nextPage = 1) {
    const params = new URLSearchParams({ q: nextQuery, provider: nextProvider });
    if (nextMode !== DEFAULT_SEARCH_MODE) params.set("mode", nextMode);
    if (nextPage > 1) params.set("page", String(nextPage));
    return `/search?${params.toString()}`;
  }

  function buildResourceUrl(key: string) {
    const params = new URLSearchParams({ q: initialQuery, provider });
    if (mode !== DEFAULT_SEARCH_MODE) params.set("mode", mode);
    if (initialPage > 1) params.set("page", String(initialPage));
    return `/resource/${encodeURIComponent(key)}?${params.toString()}`;
  }

  function submitSearch(value: string) {
    const nextQuery = value.trim();
    if (!nextQuery) {
      inputRef.current?.focus();
      return;
    }
    startTransition(() => router.push(buildSearchUrl(nextQuery, provider, mode)));
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    submitSearch(query);
  }

  function handleProviderChange(value: string) {
    const nextProvider = value as Provider;
    setProvider(nextProvider);
    startTransition(() => {
      router.replace(buildSearchUrl(initialQuery, nextProvider, mode), { scroll: false });
    });
  }

  function handleModeChange(value: SearchMode) {
    if (value === mode) return;
    setMode(value);
    setDateSort("default");
    startTransition(() => {
      router.replace(buildSearchUrl(initialQuery, provider, value), { scroll: false });
    });
  }

  function changePage(nextPage: number) {
    startTransition(() => {
      router.push(buildSearchUrl(initialQuery, provider, mode, nextPage));
    });
  }

  function cycleDateSort() {
    setDateSort((current) => (current === "default" ? "desc" : current === "desc" ? "asc" : "default"));
  }

  async function stageResource(item: MelostSearchItem) {
    const response = await fetch("/api/resources/stage", {
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
        source: item.source,
      }),
    });
    const data = await readApiData<MelostStageResponse>(response);
    if (!/^[A-Za-z0-9_-]{1,128}$/.test(data.resource_key)) {
      throw new Error("资源详情地址生成失败");
    }
    return data.resource_key;
  }

  async function stageAndOpen(item: MelostSearchItem) {
    const itemKey = item.doc_id || item.link;
    if (!item.can_stage || stageStates[itemKey]?.status === "opening") return;

    setStageStates((current) => ({ ...current, [itemKey]: { status: "opening" } }));
    try {
      const resourceKey = await stageResource(item);
      router.push(buildResourceUrl(resourceKey));
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
  const sortedResults = dateSort === "default"
    ? results
    : [...results].sort((left, right) => {
        const leftTime = Date.parse(left.shared_time);
        const rightTime = Date.parse(right.shared_time);
        const leftValue = Number.isNaN(leftTime) ? -Infinity : leftTime;
        const rightValue = Number.isNaN(rightTime) ? -Infinity : rightTime;
        return dateSort === "desc" ? rightValue - leftValue : leftValue - rightValue;
      });
  const dateSortLabel = dateSort === "default" ? "默认排序" : dateSort === "desc" ? "最新优先" : "最早优先";

  return (
    <div className="flex flex-col bg-transparent text-foreground">
      {isLoading || isPending ? <SearchLoadingOverlay /> : null}
      <a
        className="fixed top-3 left-3 z-50 -translate-y-20 rounded-lg bg-foreground px-3 py-2 text-sm font-medium text-background transition-transform focus:translate-y-0"
        href="#result-list"
      >
        跳到搜索结果
      </a>

      <header className="sticky top-0 z-20 border-b border-border bg-background/90 backdrop-blur-xl">
        <div className="mx-auto grid max-w-5xl grid-cols-[auto_1fr] items-center gap-3 px-5 py-3 sm:gap-4 sm:px-8">
          <Link
            className="flex min-h-10 items-center gap-2.5 rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/25"
            href="/"
            aria-label="返回聚优盘首页"
          >
            <Image className="h-8 w-auto" src={logo} alt="聚优盘" width={562} height={237} unoptimized />
          </Link>

          <form
            className="col-span-2 row-start-2 min-w-0 sm:col-span-1 sm:row-start-auto"
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

        </div>

        <div className="mx-auto flex w-full max-w-5xl flex-wrap items-center gap-3 px-5 pb-2 sm:px-8">
          <div className="inline-flex min-h-10 items-center gap-1 rounded-lg border border-border bg-card p-1">
            {SEARCH_MODE_VALUES.map((item) => (
              <button
                className={`min-h-8 rounded-md px-3 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/35 ${mode === item ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                type="button"
                key={item}
                aria-pressed={mode === item}
                disabled={isPending}
                onClick={() => handleModeChange(item)}
              >
                {item === "resource" ? "资源" : "视频"}
              </button>
            ))}
          </div>
          <Tabs
            className="min-w-0"
            value={provider}
            onValueChange={handleProviderChange}
          >
            <div className="w-full min-w-0 overflow-x-auto">
              <TabsList
                className="h-10 min-w-max gap-0 bg-transparent p-0"
                variant="line"
                aria-label={`按网盘来源筛选${mode === "video" ? "视频" : "资源"}`}
              >
                {PROVIDER_VALUES.map((item) => (
                  <TabsTrigger
                    className="h-9 flex-none px-3 text-xs font-normal data-active:font-semibold data-active:text-primary sm:px-3.5 sm:text-sm"
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
          <p
            className="basis-full text-left text-xs whitespace-nowrap text-muted-foreground sm:ml-auto sm:basis-auto sm:shrink-0 sm:text-right"
            role="status"
            aria-live="polite"
            aria-atomic="true"
          >
            共 <strong className="font-semibold text-foreground">{total.toLocaleString()}</strong> 条数据
          </p>
        </div>
      </header>

      <main
        id="result-list"
        className="mx-auto w-full max-w-5xl scroll-mt-32 px-5 py-7 pb-24 sm:px-8 sm:py-9 sm:pb-28"
        aria-busy={isLoading || isPending}
      >
        <div className="mb-5 flex items-end justify-between gap-4">
          <div className="min-w-0">
            <h1 className="mt-1 truncate text-xl font-semibold sm:text-2xl">
              {initialQuery ? `“${initialQuery}”` : "搜索网盘资源"}
            </h1>
          </div>
          <span className="hidden shrink-0 text-xs text-muted-foreground sm:block">
            {provider}
          </span>
        </div>

        {results.length > 0 ? (
          <div className="mb-3 flex items-center justify-between gap-3 rounded-xl border border-white/60 bg-white/35 px-3 py-2.5 shadow-[0_8px_24px_rgb(15_23_42_/_0.04)] backdrop-blur-lg sm:hidden">
            <span className="text-xs font-medium text-slate-500">更新时间</span>
            <div className="flex items-center gap-1.5">
              <button
                className="inline-flex min-h-8 items-center gap-1.5 rounded-lg px-2.5 text-xs font-medium text-slate-600 transition-colors hover:bg-white/55 hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30"
                type="button"
                aria-label={`更新日期排序：${dateSortLabel}，点击切换`}
                title={`当前：${dateSortLabel}，点击切换排序`}
                onClick={cycleDateSort}
              >
                {dateSortLabel}
                {dateSort === "desc" ? (
                  <ArrowDown className="size-3.5 text-primary" aria-hidden="true" />
                ) : dateSort === "asc" ? (
                  <ArrowUp className="size-3.5 text-primary" aria-hidden="true" />
                ) : (
                  <ArrowDownUp className="size-3.5 text-slate-400" aria-hidden="true" />
                )}
              </button>
              {dateSort !== "default" ? (
                <button
                  className="inline-flex size-8 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-white/55 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30"
                  type="button"
                  aria-label="恢复默认排序"
                  title="恢复默认排序"
                  onClick={() => setDateSort("default")}
                >
                  <RotateCcw className="size-3.5" aria-hidden="true" />
                </button>
              ) : null}
            </div>
          </div>
        ) : null}

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
          <div className="overflow-hidden rounded-2xl border border-white/60 bg-white/45 shadow-[0_18px_55px_rgb(15_23_42_/_0.08),inset_0_1px_0_rgb(255_255_255_/_0.72)] backdrop-blur-xl">
            <div className="w-full overflow-x-auto">
              <table className="block w-full border-collapse text-left sm:table sm:table-fixed">
                <caption className="sr-only">搜索结果列表</caption>
                <thead className="hidden sm:table-header-group">
                  <tr className="border-b border-slate-200/70 bg-slate-50/45 text-xs font-medium text-slate-500">
                    <th className="w-[43%] px-2 py-4 sm:w-[56%] sm:px-5">资源名称</th>
                    <th
                      className="w-[32%] px-2 py-4 sm:w-[20%] sm:px-5"
                      aria-sort={dateSort === "default" ? "none" : dateSort === "desc" ? "descending" : "ascending"}
                    >
                      <div className="flex items-center gap-1 sm:gap-1.5">
                        <button
                          className="inline-flex items-center gap-1 rounded-md px-0.5 py-1 text-left transition-colors hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 sm:gap-1.5 sm:px-1"
                          type="button"
                          aria-label={`更新日期排序：${dateSortLabel}，点击切换`}
                          title={`当前：${dateSortLabel}，点击切换排序`}
                          onClick={cycleDateSort}
                        >
                          更新日期
                          {dateSort === "desc" ? <ArrowDown className="size-3.5 text-primary" aria-hidden="true" /> : dateSort === "asc" ? <ArrowUp className="size-3.5 text-primary" aria-hidden="true" /> : <ArrowDownUp className="size-3.5 text-slate-400" aria-hidden="true" />}
                        </button>
                        {dateSort !== "default" ? (
                          <button
                            className="inline-flex size-5 items-center justify-center rounded-md text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 sm:size-6"
                            type="button"
                            aria-label="恢复默认排序"
                            title="恢复默认排序"
                            onClick={() => setDateSort("default")}
                          >
                            <RotateCcw className="size-3.5" aria-hidden="true" />
                          </button>
                        ) : null}
                      </div>
                    </th>
                    <th className="sticky right-0 z-20 w-[25%] bg-transparent px-1 py-4 text-right sm:w-[24%] sm:px-5">操作</th>
                  </tr>
                </thead>
                <tbody className="block divide-y divide-slate-200/70 sm:table-row-group">
                  {sortedResults.map((resource) => {
                    const resourceKey = resource.doc_id || resource.link;
                    const stageState = stageStates[resourceKey];
                    const resourceType = inferResourceType(`${resource.disk_name} ${resource.files}`);
                    const resourceProvider = providerFromDiskType(resource.disk_type);
                    const displayName = sanitizeResourceDisplayText(resource.disk_name) || "未命名资源";
                    const stageMessage = sanitizeResourceDisplayMessage(resource.stage_message ?? "");
                    const isOpening = stageState?.status === "opening";
                    const isDisabled = !resource.can_stage || isOpening;

                    return (
                      <tr className="group block align-top transition-colors hover:bg-white/60 sm:table-row" key={resourceKey}>
                        <td className="relative block w-full px-3 py-4 sm:table-cell sm:w-[56%] sm:px-5">
                          <div className="flex min-w-0 items-start gap-2 sm:gap-3">
                            <span className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${TYPE_STYLES[resourceType]}`} aria-hidden="true">
                              <ResourceIcon type={resourceType} />
                            </span>
                            <div className="min-w-0 pr-20 sm:pr-0">
                              <button
                                className="block max-w-full text-left text-sm font-semibold leading-5 text-slate-800 transition-colors hover:text-primary focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 disabled:cursor-not-allowed disabled:text-slate-500"
                                type="button"
                                aria-label={`进入${displayName}详情`}
                                disabled={isDisabled}
                                onClick={() => stageAndOpen(resource)}
                              >
                                <span className="line-clamp-2 [overflow-wrap:anywhere]">
                                {displayName}
                                </span>
                              </button>
                              {stageState?.status === "failed" ? (
                                <p className="mt-2 text-xs leading-5 text-red-600" role="alert">{stageState.error}</p>
                              ) : !resource.can_stage ? (
                                <p className="mt-2 text-xs leading-5 text-amber-700">{stageMessage || "该资源类型暂不支持"}</p>
                              ) : null}
                            </div>
                          </div>
                          <span className={`absolute top-4 right-3 inline-flex whitespace-nowrap rounded-lg px-2 py-1.5 text-xs font-medium sm:hidden ${PROVIDER_STYLES[resourceProvider]}`}>
                            {resourceProvider}
                          </span>
                        </td>
                        <td className="block w-full px-3 py-0 pb-3 sm:table-cell sm:w-[20%] sm:px-5 sm:py-4">
                          <div className="flex flex-col items-start gap-1.5">
                            <span className="whitespace-nowrap text-sm tabular-nums text-slate-500">
                              {resource.shared_time ? resource.shared_time.slice(0, 10) : "日期未知"}
                            </span>
                            <span className={`hidden whitespace-nowrap rounded-lg px-2 py-1.5 text-xs font-medium sm:inline-flex sm:px-3 ${PROVIDER_STYLES[resourceProvider]}`}>
                              {resourceProvider}
                            </span>
                          </div>
                        </td>
                        <td className="block w-full border-t border-slate-200/60 px-3 py-3 sm:table-cell sm:w-[24%] sm:border-t-0 sm:px-5 sm:py-4">
                          <div className="flex justify-end gap-1 sm:sticky sm:right-0 sm:z-10 sm:bg-transparent sm:px-0 sm:py-0 sm:shadow-none">
                            <Button
                              className="h-8 min-w-0 rounded-lg bg-primary px-2 text-[11px] font-semibold text-white shadow-[0_5px_14px_rgb(67_171_232_/_0.25)] transition-transform hover:-translate-y-px hover:bg-primary/90 hover:shadow-[0_7px_18px_rgb(67_171_232_/_0.32)] sm:h-9 sm:min-w-[104px] sm:w-auto sm:rounded-xl sm:px-3 sm:text-xs"
                              type="button"
                              aria-label="获取链接"
                              disabled={isDisabled}
                              onClick={() => stageAndOpen(resource)}
                            >
                              {isOpening ? <LoaderCircle className="size-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" /> : <Link2 className="size-3.5" aria-hidden="true" />}
                              获取链接
                            </Button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          <div className="flex min-h-80 flex-col items-center justify-center rounded-2xl border border-border bg-card px-6 text-center shadow-[0_1px_6px_rgb(17_24_39_/_0.03)]">
            <SearchX className="size-8 text-muted-foreground" aria-hidden="true" />
            <h2 className="mt-4 text-base font-semibold">没有找到匹配资源</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {mode === "video" ? "换个关键词再试试。" : "换个关键词或切换网盘来源再试试。"}
            </p>
            <Button className="mt-5 h-10 px-4" variant="outline" asChild>
              <Link href="/">返回首页</Link>
            </Button>
          </div>
        )}

      </main>

      {!isLoading && !searchError && totalPages > 1 ? (
        <nav className="fixed bottom-4 left-1/2 z-30 -translate-x-1/2 rounded-2xl border border-white/90 bg-white/90 p-2 shadow-[0_12px_30px_rgb(15_23_42_/_0.16)] backdrop-blur-xl sm:bottom-5" aria-label="搜索结果分页">
          <div className="flex items-center gap-2 sm:gap-3">
            <Button className="h-9 rounded-xl px-3 text-xs sm:px-4" variant="outline" type="button" disabled={initialPage <= 1 || isPending} onClick={() => changePage(initialPage - 1)}>
              <ChevronLeft data-icon="inline-start" aria-hidden="true" />
              上一页
            </Button>
            <span className="min-w-[4.5rem] text-center text-xs tabular-nums text-muted-foreground">第 {initialPage} / {totalPages} 页</span>
            <Button className="h-9 rounded-xl px-3 text-xs sm:px-4" variant="outline" type="button" disabled={initialPage >= totalPages || isPending} onClick={() => changePage(initialPage + 1)}>
              下一页
              <ChevronRight data-icon="inline-end" aria-hidden="true" />
            </Button>
          </div>
        </nav>
      ) : null}
    </div>
  );
}
