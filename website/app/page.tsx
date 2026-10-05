import Link from "next/link";

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
        <div className="mt-12 flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
          <Link
            href="/submit"
            className="rounded-lg bg-blue-600 px-8 py-3 text-lg font-semibold hover:bg-blue-500 transition-colors"
          >
            Submit a Job
          </Link>
          <Link
            href="/download"
            className="rounded-lg bg-slate-700 px-8 py-3 text-lg font-semibold hover:bg-slate-600 transition-colors"
          >
            Become a Provider
          </Link>
        </div>
      </div>

      <div className="mx-auto max-w-6xl px-6 pb-24">
        <div className="grid gap-8 md:grid-cols-3">
          <div className="rounded-xl bg-slate-800/50 p-6">
            <h3 className="text-xl font-semibold">For Consumers</h3>
            <p className="mt-2 text-slate-300">
              Upload your project as a zip file. We handle environment setup,
              scheduling, and return your results.
            </p>
          </div>
          <div className="rounded-xl bg-slate-800/50 p-6">
            <h3 className="text-xl font-semibold">For Providers</h3>
            <p className="mt-2 text-slate-300">
              Install our lightweight agent and earn credits for contributing
              spare compute capacity.
            </p>
          </div>
          <div className="rounded-xl bg-slate-800/50 p-6">
            <h3 className="text-xl font-semibold">Secure by Design</h3>
            <p className="mt-2 text-slate-300">
              Zero Trust architecture with sandboxed execution, mutual TLS,
              and full isolation between jobs.
            </p>
          </div>
        </div>
      </div>
    </main>
  );
}
