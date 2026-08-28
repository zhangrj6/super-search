export default function ResourceLoading() {
  return (
    <div className="min-h-screen bg-transparent text-foreground">
      <header className="h-16 border-b border-border bg-background" />
      <main className="mx-auto max-w-5xl px-5 py-6 sm:px-8 sm:py-8">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_18rem] lg:gap-6">
          <div className="grid gap-5">
            <div className="h-64 animate-pulse rounded-2xl border border-border bg-card motion-reduce:animate-none" />
            <div className="h-40 animate-pulse rounded-2xl border border-border bg-card motion-reduce:animate-none" />
            <div className="h-96 animate-pulse rounded-2xl border border-border bg-card motion-reduce:animate-none" />
          </div>
          <div className="grid content-start gap-5">
            <div className="h-56 animate-pulse rounded-2xl border border-border bg-card motion-reduce:animate-none" />
            <div className="h-72 animate-pulse rounded-2xl border border-border bg-card motion-reduce:animate-none" />
          </div>
        </div>
      </main>
    </div>
  );
}
