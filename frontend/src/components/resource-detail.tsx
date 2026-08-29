"use client";

import {
  AlertCircle,
  AlertTriangle,
  Archive,
  ArrowLeft,
  BookOpenText,
  CalendarDays,
  Check,
  Copy,
  ExternalLink,
  FileText,
  Film,
  LoaderCircle,
  Music2,
  Share2,
} from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  formatResourceDate,
  inferResourceType,
  providerFromResource,
  readApiData,
  resourceTagNames,
  sanitizeResourceDisplayText,
  type DriveProvider,
  type ResourceLinkResponse,
  type ResourceType,
  type UrldbResource,
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

function ResourceIcon({ type, className = "size-6" }: { type: ResourceType; className?: string }) {
  if (type === "课程") return <BookOpenText className={className} />;
  if (type === "视频") return <Film className={className} />;
  if (type === "音频") return <Music2 className={className} />;
  if (type === "素材") return <Archive className={className} />;
  return <FileText className={className} />;
}

type ResourceDetailProps = {
  resource: UrldbResource;
  backHref: string;
};

type CopyStatus = "idle" | "copied" | "error";
type TransferStatus = "idle" | "loading" | "success" | "error";

async function writeClipboard(value: string) {
  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    try {
      const textarea = document.createElement("textarea");
      textarea.value = value;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.append(textarea);
      textarea.select();
      const copied = document.execCommand("copy");
      textarea.remove();
      return copied;
    } catch {
      return false;
    }
  }
}

function isSafeShareLink(value: string) {
  try {
    return new URL(value).protocol === "https:";
  } catch {
    return false;
  }
}

export function ResourceDetail({ resource, backHref }: ResourceDetailProps) {
  const [shareStatus, setShareStatus] = useState<CopyStatus>("idle");
  const [linkCopyStatus, setLinkCopyStatus] = useState<CopyStatus>("idle");
  const [transferStatus, setTransferStatus] = useState<TransferStatus>("idle");
  const [deliveryUrl, setDeliveryUrl] = useState("");
  const shareResetTimer = useRef<number | null>(null);
  const linkResetTimer = useRef<number | null>(null);

  const provider = providerFromResource(resource);
  const resourceType = inferResourceType(`${resource.title} ${resource.description}`);
  const uploadedAt = formatResourceDate(resource.created_at);
  const displayTitle = sanitizeResourceDisplayText(resource.title) || "未命名资源";
  const displayDescription = sanitizeResourceDisplayText(resource.description);
  const keywords = resourceTagNames(resource);

  useEffect(() => {
    return () => {
      if (shareResetTimer.current) window.clearTimeout(shareResetTimer.current);
      if (linkResetTimer.current) window.clearTimeout(linkResetTimer.current);
    };
  }, []);

  async function sharePage() {
    try {
      if (navigator.share) {
        await navigator.share({
          title: displayTitle,
          text: displayDescription || displayTitle,
          url: window.location.href,
        });
      } else if (!(await writeClipboard(window.location.href))) {
        throw new Error("share failed");
      }
      setShareStatus("copied");
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") return;
      setShareStatus("error");
    }
    if (shareResetTimer.current) window.clearTimeout(shareResetTimer.current);
    shareResetTimer.current = window.setTimeout(() => setShareStatus("idle"), 2200);
  }

  async function copyDeliveryLink() {
    if (!deliveryUrl) return;
    const copied = await writeClipboard(deliveryUrl);
    setLinkCopyStatus(copied ? "copied" : "error");
    if (linkResetTimer.current) window.clearTimeout(linkResetTimer.current);
    linkResetTimer.current = window.setTimeout(() => setLinkCopyStatus("idle"), 2200);
  }

  async function requestDeliveryLink() {
    if (transferStatus === "loading") return;
    setTransferStatus("loading");
    setDeliveryUrl("");
    setLinkCopyStatus("idle");

    try {
      const response = await fetch(`/api/resources/${resource.id}/link`, {
        method: "GET",
        cache: "no-store",
      });
      const data = await readApiData<ResourceLinkResponse>(response);
      if (data.type !== "transferred" || !isSafeShareLink(data.url)) {
        throw new Error("invalid transferred link");
      }
      setDeliveryUrl(data.url);
      setTransferStatus("success");
    } catch {
      setDeliveryUrl("");
      setTransferStatus("error");
    }
  }

  const shareLabel = shareStatus === "copied" ? "已分享" : shareStatus === "error" ? "分享失败" : "分享";

  return (
    <div className="flex min-h-screen flex-col bg-transparent text-foreground">
      <a
        className="fixed top-3 left-3 z-50 -translate-y-20 rounded-lg bg-foreground px-3 py-2 text-sm font-medium text-background transition-transform focus:translate-y-0"
        href="#resource-content"
      >
        跳到资源详情
      </a>

      <header className="sticky top-0 z-20 border-b border-border bg-background/90 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-5xl items-center gap-2 px-4 sm:gap-3 sm:px-8">
          <Button
            className="h-10 px-2.5 text-muted-foreground hover:bg-card hover:text-foreground sm:px-3"
            variant="ghost"
            asChild
          >
            <Link href={backHref}>
              <ArrowLeft data-icon="inline-start" aria-hidden="true" />
              <span className="sm:hidden">返回</span>
              <span className="hidden sm:inline">返回搜索结果</span>
            </Link>
          </Button>
          <span className="mx-1 h-4 w-px bg-border" aria-hidden="true" />
          <Link
            className="flex min-h-10 items-center gap-2 rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/25"
            href="/"
            aria-label="聚优盘首页"
          >
            <Image className="h-7 w-auto" src={logo} alt="聚优盘" width={562} height={237} unoptimized />
          </Link>
          <span className={`ml-auto shrink-0 rounded-full px-3 py-1.5 text-xs font-medium ${PROVIDER_STYLES[provider]}`}>
            {provider}
          </span>
        </div>
      </header>

      <main id="resource-content" className="mx-auto w-full max-w-5xl flex-1 scroll-mt-20 px-5 py-6 sm:px-8 sm:py-8">
        <div className="grid gap-5">
          <div className="flex min-w-0 flex-col gap-5">
            <section className="rounded-2xl border border-border bg-card p-5 shadow-[0_2px_16px_rgb(17_24_39_/_0.04)] sm:p-6">
              <div className="flex items-start gap-4 sm:gap-5">
                <div className={`flex size-14 shrink-0 items-center justify-center rounded-2xl sm:size-16 ${TYPE_STYLES[resourceType]}`} aria-hidden="true">
                  <ResourceIcon type={resourceType} />
                </div>
                <div className="min-w-0 flex-1">
                  <h1 className="text-lg leading-7 font-semibold break-words sm:text-xl">{displayTitle}</h1>
                  <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-muted-foreground sm:text-sm">
                    <span className="flex items-center gap-1.5">
                      <CalendarDays className="size-3.5" aria-hidden="true" />
                      {uploadedAt}
                    </span>
                  </div>
                  {keywords.length > 0 ? (
                    <div className="mt-4 flex flex-wrap gap-2">
                      {keywords.map((keyword) => (
                        <span className="rounded-lg bg-muted px-2.5 py-1 text-xs text-muted-foreground" key={keyword}>{keyword}</span>
                      ))}
                    </div>
                  ) : null}
                </div>
              </div>

              <div className="mt-6 flex flex-wrap gap-2.5 border-t border-border pt-5">
                {transferStatus === "success" && deliveryUrl ? (
                  <Button className="h-11 rounded-xl px-5 shadow-[0_3px_10px_rgb(79_110_247_/_0.22)]" asChild>
                    <a href={deliveryUrl} target="_blank" rel="noopener noreferrer">
                      <ExternalLink data-icon="inline-start" aria-hidden="true" />
                      打开新链接
                    </a>
                  </Button>
                ) : (
                  <Button
                    className="h-11 rounded-xl px-5 shadow-[0_3px_10px_rgb(79_110_247_/_0.22)]"
                    type="button"
                    onClick={requestDeliveryLink}
                    disabled={transferStatus === "loading"}
                  >
                    {transferStatus === "loading" ? (
                      <LoaderCircle className="animate-spin motion-reduce:animate-none" data-icon="inline-start" aria-hidden="true" />
                    ) : (
                      <ExternalLink data-icon="inline-start" aria-hidden="true" />
                    )}
                    {transferStatus === "loading" ? "正在获取链接" : transferStatus === "error" ? "重新获取" : "获取链接"}
                  </Button>
                )}
                <Button className="h-11 rounded-xl px-4" variant="outline" type="button" onClick={sharePage}>
                  {shareStatus === "copied" ? <Check data-icon="inline-start" aria-hidden="true" /> : <Share2 data-icon="inline-start" aria-hidden="true" />}
                  {shareLabel}
                </Button>
              </div>

              {transferStatus === "success" && deliveryUrl ? (
                <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50 p-4" role="status">
                  <div className="flex items-start gap-3">
                    <Check className="mt-0.5 size-4 shrink-0 text-emerald-700" aria-hidden="true" />
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-semibold text-emerald-800">新的分享链接已生成</p>
                      <a className="mt-2 block break-all text-xs leading-5 text-emerald-800 underline underline-offset-2" href={deliveryUrl} target="_blank" rel="noopener noreferrer">
                        {deliveryUrl}
                      </a>
                      <Button className="mt-3 h-10 border-emerald-300 bg-white px-3 text-xs text-emerald-800 hover:bg-emerald-100" variant="outline" type="button" onClick={copyDeliveryLink}>
                        {linkCopyStatus === "copied" ? <Check data-icon="inline-start" aria-hidden="true" /> : <Copy data-icon="inline-start" aria-hidden="true" />}
                        {linkCopyStatus === "copied" ? "已复制新链接" : linkCopyStatus === "error" ? "复制失败" : "复制新链接"}
                      </Button>
                    </div>
                  </div>
                </div>
              ) : transferStatus === "error" ? (
                <div className="mt-5 flex gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-red-700" role="alert">
                  <AlertCircle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                  <p className="text-xs leading-6">获取链接失败，请稍后重试。</p>
                </div>
              ) : null}

              <p className="sr-only" aria-live="polite">
                {shareStatus === "copied" ? "详情页已分享" : shareStatus === "error" ? "详情页分享失败" : ""}
              </p>
            </section>

            <section className="rounded-2xl border border-border bg-card p-5 shadow-[0_1px_6px_rgb(17_24_39_/_0.03)] sm:p-6">
              <h2 className="text-sm font-semibold">资源描述</h2>
              <p className="mt-3 whitespace-pre-line text-sm leading-7 break-words text-muted-foreground">
                {displayDescription || "暂无资源描述。"}
              </p>
            </section>

            <section className="rounded-2xl border border-border bg-card p-5 shadow-[0_1px_6px_rgb(17_24_39_/_0.03)] sm:p-6">
              <h2 className="text-sm font-semibold">详细信息</h2>
              <dl className="mt-4">
                {[
                  { label: "资源名称", value: displayTitle },
                  { label: "资源类型", value: resourceType },
                  { label: "来源网盘", value: provider },
                  { label: "收录日期", value: uploadedAt, mono: true },
                ].map((row, index, rows) => (
                  <div className={`grid grid-cols-[5rem_minmax(0,1fr)] gap-4 py-3.5 text-xs ${index < rows.length - 1 ? "border-b border-border" : ""}`} key={row.label}>
                    <dt className="font-medium text-muted-foreground">{row.label}</dt>
                    <dd className={`min-w-0 break-words leading-5 ${row.mono ? "font-mono" : ""}`}>{row.value}</dd>
                  </div>
                ))}
              </dl>
            </section>

            <aside className="flex gap-3 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-amber-800">
              <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
              <p className="text-xs leading-6">本站仅聚合互联网公开分享信息，不存储资源文件。请确认资源授权范围并遵守相关法律法规，如有侵权请及时联系处理。</p>
            </aside>
          </div>

        </div>
      </main>

      <footer className="px-5 py-5 text-center text-xs text-muted-foreground sm:px-8">
        聚合公开分享资源 · 请尊重版权并遵守相关法律法规
      </footer>
    </div>
  );
}
