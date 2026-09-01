import type { MetadataRoute } from "next";
import type { ApiEnvelope } from "@/lib/resources";
import { fetchUrldb } from "@/lib/urldb-server";
import { absoluteUrl } from "@/lib/seo";

type SitemapResource = {
  key?: string;
  updated_at?: string;
  is_valid?: boolean;
  is_public?: boolean;
};

type ResourceListData = {
  data?: SitemapResource[];
};

const RESOURCE_LIMIT = 10000;

function validDate(value: string | undefined) {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date;
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const entries: MetadataRoute.Sitemap = [
    {
      url: absoluteUrl("/"),
      changeFrequency: "daily",
      priority: 1,
    },
  ];

  try {
    const response = await fetchUrldb(
      `/resources?is_valid=true&is_public=true&page=1&page_size=${RESOURCE_LIMIT}`,
    );
    if (!response.ok) return entries;

    const payload = (await response.json()) as ApiEnvelope<ResourceListData>;
    const resources = payload.success ? payload.data?.data ?? [] : [];
    for (const resource of resources) {
      if (!resource.key) continue;
      entries.push({
        url: absoluteUrl(`/resource/${encodeURIComponent(resource.key)}`),
        lastModified: validDate(resource.updated_at),
        changeFrequency: "weekly",
        priority: 0.7,
      });
    }
  } catch {
    // Keep the homepage available to crawlers when the resource API is down.
  }

  return entries;
}
