import { ResourceSearch } from "@/components/resource-search";

export default function Home() {
  return (
    <div id="top" className="min-h-screen text-foreground">
      <a
        className="fixed top-3 left-3 z-50 -translate-y-20 rounded-lg bg-foreground px-3 py-2 text-sm font-medium text-background transition-transform focus:translate-y-0"
        href="#search"
      >
        跳到搜索
      </a>

      <ResourceSearch />
    </div>
  );
}
