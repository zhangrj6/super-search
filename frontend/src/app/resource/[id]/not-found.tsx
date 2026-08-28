import { FileQuestion } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function ResourceNotFound() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-transparent px-5 text-center text-foreground">
      <div className="flex w-full max-w-md flex-col items-center rounded-2xl border border-border bg-card px-6 py-12 shadow-[0_2px_16px_rgb(17_24_39_/_0.04)]">
        <span className="flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
          <FileQuestion className="size-6" aria-hidden="true" />
        </span>
        <h1 className="mt-5 text-xl font-semibold">资源不存在</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          该资源可能已失效、被移除，或者链接地址不正确。
        </p>
        <Button className="mt-6 h-10 px-4" asChild>
          <Link href="/">返回首页</Link>
        </Button>
      </div>
    </main>
  );
}
