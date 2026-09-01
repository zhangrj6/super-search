import type { Metadata } from "next";
import { JsonLd } from "@/components/json-ld";
import { ResourceSearch } from "@/components/resource-search";
import { absoluteUrl, SITE_DESCRIPTION, SITE_NAME } from "@/lib/seo";

export const metadata: Metadata = {
  title: `${SITE_NAME}｜网盘资源搜索`,
  description: SITE_DESCRIPTION,
  alternates: {
    canonical: "/",
  },
};

const websiteJsonLd: Record<string, unknown> = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  name: SITE_NAME,
  url: absoluteUrl("/"),
  description: SITE_DESCRIPTION,
  potentialAction: {
    "@type": "SearchAction",
    target: {
      "@type": "EntryPoint",
      urlTemplate: absoluteUrl("/search?q={search_term_string}"),
    },
    "query-input": "required name=search_term_string",
  },
};

export default function Home() {
  return (
    <>
      <JsonLd data={websiteJsonLd} />
      <div id="top" className="min-h-screen text-foreground">
        <a
          className="fixed top-3 left-3 z-50 -translate-y-20 rounded-lg bg-foreground px-3 py-2 text-sm font-medium text-background transition-transform focus:translate-y-0"
          href="#search"
        >
          跳到搜索
        </a>

        <ResourceSearch />
      </div>
    </>
  );
}
