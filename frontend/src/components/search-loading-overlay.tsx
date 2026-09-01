import { Film, Link2, LoaderCircle, Search } from "lucide-react";
import type { SearchMode } from "@/lib/resources";

type SearchLoadingOverlayProps = {
  mode?: SearchMode;
  variant?: "search" | "link";
};

export function SearchLoadingOverlay({ mode = "resource", variant = "search" }: SearchLoadingOverlayProps) {
  const isVideo = mode === "video";
  const isLink = variant === "link";
  const accent = isVideo
    ? {
        icon: "border-cyan-200/80 bg-cyan-50/80 text-cyan-700",
        halo: "bg-cyan-400/20",
        progress: "bg-cyan-500",
      }
    : {
        icon: "border-sky-200/80 bg-sky-50/80 text-primary",
        halo: "bg-primary/15",
        progress: "bg-primary",
      };

  return (
    <div
      className="pointer-events-none fixed inset-0 z-50 grid place-items-center px-4"
      role="status"
      aria-live="polite"
      aria-atomic="true"
    >
      <div className="relative w-full max-w-[22rem] overflow-hidden rounded-[1.35rem] border border-white/80 bg-white/75 px-5 py-5 text-slate-700 shadow-[0_18px_50px_rgb(15_23_42_/_0.16),inset_0_1px_0_rgb(255_255_255_/_0.8)] backdrop-blur-2xl">
        <div className={`absolute inset-x-8 top-0 h-px ${accent.progress} opacity-70`} aria-hidden="true" />
        <div className="flex items-center gap-3.5">
          <div className={`relative flex size-11 shrink-0 items-center justify-center rounded-xl border ${accent.icon}`}>
            <span className={`absolute inset-1 rounded-lg ${accent.halo} animate-pulse motion-reduce:animate-none`} aria-hidden="true" />
            {isLink ? (
              <Link2 className="relative size-5" strokeWidth={1.9} aria-hidden="true" />
            ) : isVideo ? (
              <Film className="relative size-5" strokeWidth={1.9} aria-hidden="true" />
            ) : (
              <Search className="relative size-5" strokeWidth={1.9} aria-hidden="true" />
            )}
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold text-slate-800">
              {isLink ? "正在获取链接..." : isVideo ? "正在全力搜索视频..." : "正在全力搜索中..."}
            </p>
            <p className="mt-1 text-xs leading-5 text-slate-500">
              {isLink ? "正在生成可访问的资源链接" : isVideo ? "正在匹配片源与网盘链接" : "正在整理可用资源链接"}
            </p>
          </div>
          <LoaderCircle className="size-4 shrink-0 animate-spin text-slate-400 motion-reduce:animate-none" aria-hidden="true" />
        </div>
        <div className="mt-4 h-1 overflow-hidden rounded-full bg-slate-200/75" aria-hidden="true">
          <div className={`h-full w-2/5 rounded-full ${accent.progress} animate-[search-loading-sweep_1.8s_ease-in-out_infinite] motion-reduce:animate-none`} />
        </div>
        <div className="mt-2.5 flex items-center justify-between text-[11px] text-slate-400">
          <span>{isLink ? "链接处理中" : isVideo ? "视频模式" : "资源模式"}</span>
          <span>请稍候</span>
        </div>
      </div>
    </div>
  );
}
