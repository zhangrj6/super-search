import { LoaderCircle } from "lucide-react";

export function SearchLoadingOverlay() {
  return (
    <div
      className="pointer-events-none fixed inset-0 z-50 grid place-items-center px-4"
      role="status"
      aria-live="polite"
      aria-atomic="true"
    >
      <div className="flex max-w-full items-center gap-3 rounded-lg border border-white/90 bg-white/95 px-5 py-4 text-sm font-medium whitespace-nowrap text-slate-700 shadow-[0_18px_50px_rgb(15_23_42_/_0.18)] backdrop-blur-xl">
        <LoaderCircle
          className="size-5 shrink-0 animate-spin text-primary motion-reduce:animate-none"
          aria-hidden="true"
        />
        <span>正在全力搜索中...</span>
      </div>
    </div>
  );
}
