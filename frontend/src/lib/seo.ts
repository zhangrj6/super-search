export const SITE_NAME = "聚优盘";
export const DEFAULT_SITE_URL = "https://52juyou.com";
export const SITE_DESCRIPTION =
  "聚优盘是网盘资源搜索工具，聚合百度网盘、夸克网盘、阿里云盘、迅雷云盘和 UC 网盘公开分享资源，快速检索影视、课程、教程与文档。";

export const SITE_KEYWORDS = [
  "聚优盘",
  "网盘搜索",
  "网盘资源搜索",
  "百度网盘搜索",
  "夸克网盘搜索",
  "阿里云盘搜索",
  "迅雷云盘搜索",
  "UC网盘搜索",
  "影视资源搜索",
  "课程资源搜索",
];

function normalizeSiteUrl(value: string) {
  try {
    const url = new URL(value);
    if (url.protocol !== "http:" && url.protocol !== "https:") return DEFAULT_SITE_URL;
    url.pathname = url.pathname.replace(/\/+$/, "");
    url.search = "";
    url.hash = "";
    return url.toString().replace(/\/$/, "");
  } catch {
    return DEFAULT_SITE_URL;
  }
}

export const SITE_URL = normalizeSiteUrl(
  process.env.NEXT_PUBLIC_SITE_URL?.trim() ||
    process.env.PUBLIC_SITE_URL?.trim() ||
    DEFAULT_SITE_URL,
);

export function absoluteUrl(path: string) {
  return new URL(path, `${SITE_URL}/`).toString();
}

export function truncateMetaText(value: string, maxLength = 160) {
  const normalized = value.replace(/\s+/g, " ").trim();
  if (normalized.length <= maxLength) return normalized;
  return `${normalized.slice(0, Math.max(1, maxLength - 1))}…`;
}
