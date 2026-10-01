export default function Home() {
  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 text-white">
      <div className="mx-auto max-w-4xl px-6 py-24 text-center">
        <h1 className="text-5xl font-bold tracking-tight">
          Distributed Compute Cloud
        </h1>
        <p className="mt-6 text-xl text-slate-300">
          Transform idle personal computers into a secure, trustworthy
          distributed compute cloud.
        </p>
        <p className="mt-4 text-lg text-slate-400">
          Consumers submit work without thinking about infrastructure.
          Providers contribute spare capacity without compromising their own
          machine&apos;s safety.
        </p>
        <div className="mt-12 flex flex-col items-center gap-4">
          <a
            href="#download"
            className="rounded-lg bg-blue-600 px-8 py-3 text-lg font-semibold hover:bg-blue-500 transition-colors"
          >
            Download Worker Agent
          </a>
          <span className="text-sm text-slate-500">
            Coming soon — Phase 0 placeholder
          </span>
        </div>
      </div>
    </main>
  );
}
