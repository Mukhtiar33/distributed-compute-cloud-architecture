"use client";

import { useState } from "react";

export default function AccountPage() {
  const [role, setRole] = useState<"consumer" | "provider" | null>(null);
  const [registered, setRegistered] = useState(false);

  const handleRegister = (selectedRole: "consumer" | "provider") => {
    setRole(selectedRole);
    setRegistered(true);
  };

  if (registered && role) {
    return (
      <main className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 text-white">
        <div className="mx-auto max-w-4xl px-6 py-24">
          <h1 className="text-4xl font-bold tracking-tight">Account Created</h1>
          <div className="mt-8 rounded-xl bg-slate-800/50 p-8">
            <p className="text-lg">
              Your <span className="font-semibold text-blue-400">{role}</span> account has been created.
            </p>
            <p className="mt-4 text-slate-300">
              {role === "consumer"
                ? "You can now submit jobs from the Submit page."
                : "You can now download the Worker Agent from the Download page."}
            </p>
            <div className="mt-6 rounded-lg bg-slate-700/50 p-4">
              <p className="text-sm text-slate-400">
                Account ID: {role}-{Date.now().toString(36)}
              </p>
              <p className="text-sm text-slate-400">
                Role: {role}
              </p>
            </div>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 text-white">
      <div className="mx-auto max-w-4xl px-6 py-24">
        <h1 className="text-4xl font-bold tracking-tight">Create Account</h1>
        <p className="mt-4 text-lg text-slate-300">
          Choose how you want to participate in the network.
        </p>

        <div className="mt-12 grid gap-6 md:grid-cols-2">
          <button
            onClick={() => handleRegister("consumer")}
            className="rounded-xl bg-slate-800/50 p-8 text-left hover:bg-slate-700/50 transition-colors"
          >
            <h2 className="text-2xl font-semibold">Consumer</h2>
            <p className="mt-2 text-slate-300">
              Submit jobs and get results. No infrastructure management needed.
            </p>
            <ul className="mt-4 space-y-1 text-sm text-slate-400">
              <li>• Submit zip files</li>
              <li>• Automatic environment setup</li>
              <li>• Results delivered as zip</li>
            </ul>
          </button>

          <button
            onClick={() => handleRegister("provider")}
            className="rounded-xl bg-slate-800/50 p-8 text-left hover:bg-slate-700/50 transition-colors"
          >
            <h2 className="text-2xl font-semibold">Provider</h2>
            <p className="mt-2 text-slate-300">
              Contribute spare compute capacity and earn credits.
            </p>
            <ul className="mt-4 space-y-1 text-sm text-slate-400">
              <li>• Run Worker Agent</li>
              <li>• Earn credits for compute</li>
              <li>• Resource-throttled, non-disruptive</li>
            </ul>
          </button>
        </div>

        <div className="mt-8 rounded-xl bg-slate-800/50 p-6">
          <p className="text-sm text-slate-400">
            <span className="font-semibold text-slate-300">Note:</span> You can register for both roles separately.
            Each role gets its own identity, ensuring proper isolation between your consumer and provider activities.
          </p>
        </div>
      </div>
    </main>
  );
}
