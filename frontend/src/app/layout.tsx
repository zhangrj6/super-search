import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import {
  SITE_DESCRIPTION,
  SITE_KEYWORDS,
  SITE_NAME,
  SITE_URL,
} from "@/lib/seo";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: `${SITE_NAME}｜网盘资源搜索`,
  description: SITE_DESCRIPTION,
  keywords: SITE_KEYWORDS,
  applicationName: SITE_NAME,
  authors: [{ name: SITE_NAME, url: SITE_URL }],
  creator: SITE_NAME,
  publisher: SITE_NAME,
  alternates: {
    canonical: "/",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-image-preview": "large",
      "max-snippet": -1,
      "max-video-preview": -1,
    },
  },
  openGraph: {
    type: "website",
    locale: "zh_CN",
    url: SITE_URL,
    siteName: SITE_NAME,
    title: `${SITE_NAME}｜网盘资源搜索`,
    description: SITE_DESCRIPTION,
  },
  twitter: {
    card: "summary",
    title: `${SITE_NAME}｜网盘资源搜索`,
    description: SITE_DESCRIPTION,
  },
  icons: {
    icon: "/favicon.ico",
  },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="zh-CN"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body>
        <div className="site-shell">
          <div className="site-shell__grid" aria-hidden="true" />
          <div className="site-shell__scanline" aria-hidden="true" />
          {children}
        </div>
      </body>
    </html>
  );
}
