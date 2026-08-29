"use client";

import Image from "next/image";
import { LoaderCircle, Search, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState, useTransition } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Provider } from "@/lib/resources";
import logo from "@/assets/logo.png";

export function ResourceSearch() {
  const router = useRouter();
  const [provider] = useState<Provider>("全部");
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
    <main className="flex min-h-screen flex-col overflow-hidden bg-transparent">
      <section className="relative z-10 mx-auto flex min-h-[calc(100svh-7rem)] w-full max-w-5xl flex-1 flex-col justify-center px-5 py-12 sm:px-8 sm:py-16">
        <div
          id="search"
          className="-translate-y-14 flex scroll-mt-6 flex-col items-center justify-center text-center sm:-translate-y-20"
        >
          <a
            className="mb-8 flex flex-col items-center rounded-xl outline-none transition-transform duration-200 hover:scale-[1.015] focus-visible:ring-4 focus-visible:ring-primary/15 sm:mb-9"
            href="#search"
            aria-label="聚优盘首页"
          >
            <Image
              className="h-auto w-[min(12.5rem,52vw)] drop-shadow-[0_12px_22px_rgb(47_147_205_/_0.18)] sm:w-[16rem]"
              src={logo}
              alt="聚优盘"
              width={562}
              height={237}
              priority
              unoptimized
            />
            <span className="mt-3 text-xs font-medium tracking-[0.28em] text-slate-500/85 sm:mt-3.5 sm:text-sm">
              尽在聚优 一搜即可
            </span>
          </a>

          <h1 className="text-2xl font-semibold tracking-tight text-slate-800 sm:text-3xl">
            聚优盘网盘资源搜索
          </h1>
          <p className="mt-2 max-w-xl text-sm leading-6 text-slate-500 sm:text-base">
            搜索百度网盘、夸克网盘、阿里云盘、迅雷云盘和 UC 网盘中的影视、课程、教程与文档资源。
          </p>

          <form className="mt-7 w-full max-w-2xl sm:mt-8" action="/search" method="get" onSubmit={handleSubmit}>
            <label className="sr-only" htmlFor="resource-query">
              搜索文件名或资源关键词
            </label>
            <div className="group flex h-[4.25rem] items-center gap-2 rounded-[1.25rem] border border-white/80 bg-white/80 pr-2.5 pl-5 shadow-[0_18px_55px_rgb(15_23_42_/_0.1),inset_0_1px_0_rgb(255_255_255_/_0.9)] backdrop-blur-xl transition-[border-color,box-shadow,transform,background-color] duration-300 focus-within:-translate-y-0.5 focus-within:border-primary/60 focus-within:bg-white/95 focus-within:shadow-[0_0_0_4px_rgb(67_171_232_/_0.13),0_22px_60px_rgb(15_23_42_/_0.14),inset_0_1px_0_rgb(255_255_255_/_0.95)] sm:gap-3 sm:pr-3 sm:pl-6">
              <Search
                className="size-5 shrink-0 text-slate-400 transition-colors group-focus-within:text-primary"
                strokeWidth={2}
                aria-hidden="true"
              />
              <Input
                ref={inputRef}
                id="resource-query"
                name="q"
                className="h-12 min-w-0 flex-1 border-0 bg-transparent px-0 text-base text-slate-800 shadow-none placeholder:text-slate-400 focus-visible:border-0 focus-visible:ring-0 md:text-base"
                type="text"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="搜索影视、教程、资源关键词..."
                aria-keyshortcuts="/"
                autoComplete="off"
              />
              {query ? (
                <Button
                  className="size-9 shrink-0 rounded-full text-slate-400 hover:text-slate-700"
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
                className="h-10 min-w-16 shrink-0 rounded-[0.9rem] bg-primary px-4 text-sm font-semibold text-white shadow-[0_6px_18px_rgb(67_171_232_/_0.3)] transition-all hover:-translate-y-px hover:bg-primary/90 hover:shadow-[0_8px_22px_rgb(67_171_232_/_0.36)] sm:min-w-20 sm:px-6"
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

        </div>
      </section>
      <footer className="relative z-10 mx-auto w-full max-w-4xl px-5 pb-5 text-center text-[11px] leading-5 text-slate-500/80 sm:px-8 sm:pb-6 sm:text-xs sm:leading-6">
        <p>
          免责声明：聚优盘仅提供公开网盘链接索引服务，本站不存储、不上传任何资源文件。所有资源版权归原作者所有，仅供个人学习交流，禁止商用。如存在侵权，请联系站长处理，我们将及时删除对应链接。
        </p>
      </footer>
    </main>
  );
}
