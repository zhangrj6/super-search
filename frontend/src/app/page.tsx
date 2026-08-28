import { Button } from "@/components/ui/button";
import { ResourceSearch } from "@/components/resource-search";

const NAV_ITEMS = [
  { label: "功能", href: "#search" },
  { label: "来源", href: "#sources" },
  { label: "关于", href: "#about" },
];

export default function Home() {
  return (
    <div id="top" className="flex min-h-screen flex-col bg-background text-foreground">
      <a
        className="fixed top-3 left-3 z-50 -translate-y-20 rounded-lg bg-foreground px-3 py-2 text-sm font-medium text-background transition-transform focus:translate-y-0"
        href="#search"
      >
        跳到搜索
      </a>

      <header className="relative z-10 shrink-0">
        <div className="flex h-16 items-center justify-between px-5 sm:h-18 sm:px-8">
          <a
            className="flex min-h-11 items-center gap-2.5 rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/25"
            href="#top"
            aria-label="PanSearch 首页"
          >
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground shadow-[0_3px_10px_rgb(79_110_247_/_0.28)]">
              P
            </span>
            <span className="text-sm font-semibold">PanSearch</span>
          </a>

          <nav className="flex items-center" aria-label="首页导航">
            {NAV_ITEMS.map((item) => (
              <Button
                key={item.href}
                className="h-10 px-3 text-sm font-normal text-muted-foreground hover:bg-card hover:text-foreground sm:px-4"
                variant="ghost"
                asChild
              >
                <a href={item.href}>{item.label}</a>
              </Button>
            ))}
          </nav>
        </div>
      </header>

      <ResourceSearch />

      <footer id="about" className="shrink-0 px-5 py-4 text-center text-xs text-muted-foreground sm:px-8">
        聚合公开分享资源 · 请尊重版权并遵守相关法律法规
      </footer>
    </div>
  );
}
