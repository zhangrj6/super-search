import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
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
  title: "聚优盘 - 网盘资源搜索",
  description: "聚合公开分享的网盘资源，快速找到学习、设计与开源内容。",
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
