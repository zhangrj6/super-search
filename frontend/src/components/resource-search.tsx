"use client";

import { LoaderCircle, Search, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState, useTransition } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { Provider } from "@/lib/resources";

const PROVIDERS: Array<{ value: Provider; className: string }> = [
  {
    value: "全部",
    className: "data-active:bg-primary/10 data-active:text-primary",
  },
  {
    value: "百度网盘",
    className: "data-active:bg-blue-50 data-active:text-blue-600",
  },
  {
    value: "阿里云盘",
    className: "data-active:bg-orange-50 data-active:text-orange-600",
  },
  {
    value: "夸克网盘",
    className: "data-active:bg-violet-50 data-active:text-violet-600",
  },
  {
    value: "UC网盘",
    className: "data-active:bg-emerald-50 data-active:text-emerald-700",
  },
  {
    value: "迅雷云盘",
    className: "data-active:bg-red-50 data-active:text-red-600",
  },
];

const HOT_SEARCHES = [
  "Python 教程",
  "机器学习",
  "前端开发",
  "4K 壁纸",
  "Blender",
  "算法导论",
  "Linux 内核",
  "设计模板",
];

const STATS = [
  { label: "收录资源", value: "2,400万+" },
  { label: "支持网盘", value: "5 个" },
  { label: "今日更新", value: "12万+" },
  { label: "日均搜索", value: "80万+" },
];

export function ResourceSearch() {
  const router = useRouter();
  const [provider, setProvider] = useState<Provider>("全部");
  const [query, setQuery] = useState("");
  const [isPending, startTransition] = useTransition();
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    function focusSearch(event: KeyboardEvent) {
      if (
        event.key === "/" &&
        document.activeElement?.tagName !== "INPUT" &&
        document.activeElement?.tagName !== "TEXTAREA"
      ) {
        event.preventDefault();
        inputRef.current?.focus();
      }
    }

    window.addEventListener("keydown", focusSearch);
    return () => window.removeEventListener("keydown", focusSearch);
  }, []);

  function runSearch(value: string) {
    const nextQuery = value.trim();
    if (!nextQuery) {
      inputRef.current?.focus();
      return;
    }

    setQuery(nextQuery);
    const params = new URLSearchParams({ q: nextQuery });
    if (provider !== "全部") params.set("provider", provider);

    startTransition(() => {
      router.push(`/search?${params.toString()}`);
    });
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    runSearch(query);
  }

  return (
    <main className="flex flex-1 flex-col">
      <section className="mx-auto flex min-h-[calc(100svh-7rem)] w-full max-w-6xl flex-col px-5 pb-7 sm:min-h-[calc(100svh-7.5rem)] sm:px-8 sm:pb-8">
        <div
          id="search"
          className="flex flex-1 scroll-mt-6 flex-col items-center justify-center py-10 text-center sm:py-12"
        >
          <div className="mb-6 flex items-center gap-2 rounded-full border border-blue-200 bg-blue-50 px-4 py-2 text-xs font-medium text-blue-600 sm:mb-7">
            <span
              className="size-1.5 rounded-full bg-blue-500 motion-safe:animate-pulse"
              aria-hidden="true"
            />
            实时收录 · 每日更新 12 万+ 资源
          </div>

          <h1 className="max-w-3xl text-[2.375rem] leading-[1.2] font-bold tracking-normal sm:text-5xl sm:leading-[1.15]">
            搜遍全网<span className="text-primary">网盘资源</span>
          </h1>
          <p
            id="search-description"
            className="mt-4 max-w-lg text-sm leading-7 text-muted-foreground sm:text-base"
          >
            聚合百度网盘、阿里云盘、夸克网盘等主流平台
            <span className="hidden sm:inline">，</span>
            <br className="hidden sm:block" />
            一键找到你想要的文件。
          </p>

          <form className="mt-8 w-full max-w-2xl" onSubmit={handleSubmit}>
            <label className="sr-only" htmlFor="resource-query">
              搜索文件名或资源关键词
            </label>
            <div className="group flex h-16 items-center gap-2 rounded-2xl border border-border bg-card pr-2.5 pl-4 shadow-[0_2px_16px_rgb(17_24_39_/_0.04)] transition-[border-color,box-shadow] duration-200 focus-within:border-primary focus-within:shadow-[0_0_0_4px_rgb(79_110_247_/_0.08),0_8px_28px_rgb(17_24_39_/_0.06)] sm:gap-3 sm:px-3 sm:pl-5">
              <Search
                className="size-5 shrink-0 text-muted-foreground transition-colors group-focus-within:text-primary"
                strokeWidth={1.8}
                aria-hidden="true"
              />
              <Input
                ref={inputRef}
                id="resource-query"
                className="h-12 min-w-0 flex-1 border-0 bg-transparent px-0 text-base shadow-none focus-visible:border-0 focus-visible:ring-0 md:text-base"
                type="text"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="搜索文件名、资源关键词..."
                aria-describedby="search-description"
                aria-keyshortcuts="/"
                autoComplete="off"
              />
              {query ? (
                <Button
                  className="size-9 shrink-0 rounded-full text-muted-foreground hover:text-foreground"
                  variant="secondary"
                  size="icon"
                  type="button"
                  onClick={() => {
                    setQuery("");
                    inputRef.current?.focus();
                  }}
                  aria-label="清空搜索"
                >
                  <X className="size-4" aria-hidden="true" />
                </Button>
              ) : null}
              <Button
                className="h-11 min-w-16 shrink-0 rounded-xl px-4 text-sm font-semibold shadow-[0_3px_10px_rgb(79_110_247_/_0.25)] hover:bg-primary/90 sm:min-w-20 sm:px-6"
                type="submit"
                disabled={isPending}
              >
                {isPending ? (
                  <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
                ) : (
                  "搜索"
                )}
              </Button>
            </div>
          </form>

          <div className="mt-5 flex w-full max-w-2xl flex-col items-center gap-2.5">
            <span className="text-xs text-muted-foreground">热门搜索</span>
            <div className="flex flex-wrap justify-center gap-2">
              {HOT_SEARCHES.map((item) => (
                <Button
                  className="h-8 rounded-lg border-border bg-card px-3 text-xs font-normal text-muted-foreground shadow-none hover:border-blue-300 hover:bg-blue-50 hover:text-blue-600"
                  variant="outline"
                  size="sm"
                  type="button"
                  key={item}
                  onClick={() => runSearch(item)}
                  disabled={isPending}
                >
                  {item}
                </Button>
              ))}
            </div>
          </div>

          <div id="sources" className="mt-9 w-full scroll-mt-8 sm:mt-11">
            <Tabs
              value={provider}
              onValueChange={(value) => setProvider(value as Provider)}
            >
              <div className="flex flex-col items-center justify-center gap-3 sm:flex-row sm:gap-4">
                <span className="shrink-0 text-xs text-muted-foreground">已接入</span>
                <TabsList className="flex h-auto w-full max-w-2xl flex-wrap gap-2 bg-transparent p-0">
                  {PROVIDERS.map((item) => (
                    <TabsTrigger
                      className={`h-8 flex-none rounded-lg border-0 bg-card px-2.5 text-xs font-medium text-muted-foreground shadow-none data-active:shadow-none ${item.className}`}
                      value={item.value}
                      key={item.value}
                      disabled={isPending}
                    >
                      {item.value}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </div>
            </Tabs>
          </div>
        </div>

        <dl className="mx-auto grid w-full max-w-3xl grid-cols-2 overflow-hidden rounded-2xl border border-border bg-card px-3 py-4 shadow-[0_2px_16px_rgb(17_24_39_/_0.04)] sm:grid-cols-4 sm:px-0 sm:py-5">
          {STATS.map((stat, index) => (
            <div
              className={`relative px-3 py-2 text-center sm:px-6 ${
                index % 2 === 0 ? "border-r border-border sm:border-r" : "sm:border-r"
              } ${index < 2 ? "border-b border-border sm:border-b-0" : ""} ${
                index === STATS.length - 1 ? "sm:border-r-0" : ""
              }`}
              key={stat.label}
            >
              <dt className="mt-1 text-xs text-muted-foreground">{stat.label}</dt>
              <dd className="text-xl font-bold tabular-nums sm:text-2xl">{stat.value}</dd>
            </div>
          ))}
        </dl>
      </section>
    </main>
  );
}
