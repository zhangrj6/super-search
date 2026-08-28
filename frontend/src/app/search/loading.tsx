export default function SearchLoading() {
  return (
    <div className="min-h-screen bg-transparent text-foreground">
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex h-18 max-w-5xl items-center gap-4 px-5 sm:px-8">
          <div className="size-8 rounded-lg bg-primary" />
          <div className="h-11 flex-1 animate-pulse rounded-xl bg-muted motion-reduce:animate-none" />
          <div className="h-4 w-14 animate-pulse rounded bg-muted motion-reduce:animate-none" />
        </div>
        <div className="mx-auto flex h-11 max-w-5xl items-center gap-3 px-5 sm:px-8">
          {Array.from({ length: 5 }).map((_, index) => (
            <div
              className="h-5 w-18 animate-pulse rounded bg-muted motion-reduce:animate-none"
              key={index}
            />
          ))}
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-5 py-8 sm:px-8">
        <div className="mb-6 h-7 w-48 animate-pulse rounded bg-muted motion-reduce:animate-none" />
        <div className="grid gap-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <div
              className="h-32 animate-pulse rounded-2xl border border-border bg-card motion-reduce:animate-none"
              key={index}
            />
          ))}
        </div>
      </main>
    </div>
  );
}
