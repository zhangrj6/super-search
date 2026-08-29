export const PROVIDER_VALUES = [
  "全部",
  "百度网盘",
  "阿里云盘",
  "夸克网盘",
  "迅雷云盘",
  "UC网盘",
] as const;

export type Provider = (typeof PROVIDER_VALUES)[number];
export type DriveProvider = Exclude<Provider, "全部">;
export type ResourceType = "课程" | "视频" | "音频" | "文档" | "素材";

export const PROVIDER_CODES: Record<Provider, string> = {
  全部: "",
  百度网盘: "BDY",
  阿里云盘: "ALY",
  夸克网盘: "QUARK",
  迅雷云盘: "XUNLEI",
  UC网盘: "UC",
};

export type ApiEnvelope<T> = {
  success: boolean;
  message: string;
  data: T;
  code: number;
};

export type MelostSearchItem = {
  doc_id: string;
  disk_name: string;
  disk_type: string;
  link: string;
  disk_pass: string;
  files: string;
  tags: string[];
  shared_time: string;
  share_user: string;
  size: number;
  can_stage: boolean;
  stage_message?: string;
};

export type MelostSearchResponse = {
  total: number;
  page: number;
  page_size: number;
  took: number;
  items: MelostSearchItem[];
};

export type MelostStageResponse = {
  status: "staged";
  existing: boolean;
  resource_id: number;
  resource_key: string;
};

export type ResourceTag = {
  id?: number;
  name?: string;
};

export type ResourcePan = {
  id: number;
  name: string;
  key: number;
  remark: string;
};

export type UrldbResource = {
  id: number;
  key: string;
  title: string;
  description: string;
  url: string;
  pan_id: number | null;
  save_url: string;
  file_size: string;
  category_id: number | null;
  category_name: string;
  view_count: number;
  is_valid: boolean;
  is_public: boolean;
  created_at: string;
  updated_at: string;
  tags: ResourceTag[];
  cover: string;
  author: string;
  error_msg: string;
  source: string;
  external_id: string;
  pan?: ResourcePan;
};

export type ResourceGroupResponse = {
  key: string;
  resources: UrldbResource[];
  total: number;
};

export type ResourceLinkResponse = {
  url: string;
  type: "transferred";
  platform: string;
  resource_id: number;
  message?: string;
};

const DISK_PROVIDER_MAP: Record<string, DriveProvider> = {
  BDY: "百度网盘",
  BAIDU: "百度网盘",
  ALY: "阿里云盘",
  ALIPAN: "阿里云盘",
  ALIYUN: "阿里云盘",
  QUARK: "夸克网盘",
  XUNLEI: "迅雷云盘",
  UC: "UC网盘",
};

export function isProvider(value: string | undefined): value is Provider {
  return PROVIDER_VALUES.some((provider) => provider === value);
}

export function providerFromDiskType(diskType: string): DriveProvider {
  return DISK_PROVIDER_MAP[diskType.trim().toUpperCase()] ?? "夸克网盘";
}

export function providerFromResource(resource: UrldbResource): DriveProvider {
  const identity = `${resource.pan?.name ?? ""} ${resource.pan?.remark ?? ""}`.toUpperCase();
  if (identity.includes("百度") || identity.includes("BAIDU")) return "百度网盘";
  if (identity.includes("阿里") || identity.includes("ALI")) return "阿里云盘";
  if (identity.includes("迅雷") || identity.includes("XUNLEI")) return "迅雷云盘";
  if (identity.includes("UC")) return "UC网盘";
  return "夸克网盘";
}

export function inferResourceType(value: string): ResourceType {
  const text = value.toLocaleLowerCase("zh-CN");
  if (/课程|教程|教学|lecture|course/.test(text)) return "课程";
  if (/电影|电视剧|动漫|视频|\b4k\b|\b1080p\b|mkv|mp4/.test(text)) return "视频";
  if (/音乐|音频|原声|专辑|mp3|flac|wav/.test(text)) return "音频";
  if (/壁纸|素材|模板|字体|图标|psd|图片/.test(text)) return "素材";
  return "文档";
}

export function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) return "大小未知";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(size) / Math.log(1024)), units.length - 1);
  return `${(size / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

export function formatResourceDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "日期未知";
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeZone: "Asia/Shanghai",
  }).format(date);
}

/** Remove known upstream advertising labels from user-visible resource text. */
export function sanitizeResourceDisplayText(value: string) {
  return value
    .replace(/影盘社/gi, "")
    .replace(/[ \t]{2,}/g, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

export function sanitizeResourceDisplayMessage(value: string) {
  return sanitizeResourceDisplayText(value)
    .replace(/转存/g, "获取链接")
    .trim();
}

export function resourceTagNames(resource: UrldbResource) {
  return (resource.tags ?? [])
    .map((tag) => tag.name?.trim() ?? "")
    .map(sanitizeResourceDisplayText)
    .filter(Boolean)
    .slice(0, 6);
}

export async function readApiData<T>(response: Response): Promise<T> {
  let payload: ApiEnvelope<T> | null = null;
  try {
    payload = (await response.json()) as ApiEnvelope<T>;
  } catch {
    // The caller receives a stable message for non-JSON gateway failures.
  }

  if (!response.ok || !payload?.success || payload.data == null) {
    throw new Error(payload?.message || "请求失败，请稍后重试");
  }
  return payload.data;
}
