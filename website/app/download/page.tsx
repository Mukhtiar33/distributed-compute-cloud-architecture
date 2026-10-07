"use client";

import { useState } from "react";

export default function DownloadPage() {
  const [downloading, setDownloading] = useState(false);

  const handleDownload = async () => {
    setDownloading(true);
    try {
      const response = await fetch("/api/download/worker-agent");
      if (!response.ok) {
        throw new Error("Download failed");
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "worker-agent";
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      alert("Download failed. Please try again later.");
    } finally {
      setDownloading(false);
    }
  };

  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 text-white">
      <div className="mx-auto max-w-4xl px-6 py-24">
        <h1 className="text-4xl font-bold tracking-tight">Download Worker Agent</h1>
        <p className="mt-4 text-lg text-slate-300">
          Contribute your spare compute capacity to the network.
        </p>

        <div className="mt-12 rounded-xl bg-slate-800/50 p-8">
          <h2 className="text-2xl font-semibold">How it works</h2>
          <ol className="mt-4 space-y-3 text-slate-300">
            <li>1. Download the Worker Agent binary</li>
            <li>2. Run it — it automatically connects to the platform</li>
            <li>3. Start earning credits for contributed compute</li>
          </ol>

          <div className="mt-8 rounded-lg bg-slate-700/50 p-4">
            <p className="text-sm text-slate-400">
              <span className="font-semibold text-slate-300">Usage:</span>
            </p>
            <code className="mt-2 block text-sm text-green-400">
              ./worker-agent https://your-laptop-ip:8443
            </code>
            <p className="mt-2 text-sm text-slate-400">
              Replace <code className="text-slate-300">your-laptop-ip</code> with your laptop&apos;s IP address.
            </p>
          </div>

          <button
            onClick={handleDownload}
            disabled={downloading}
            className="mt-8 rounded-lg bg-blue-600 px-8 py-3 text-lg font-semibold hover:bg-blue-500 disabled:opacity-50 transition-colors"
          >
            {downloading ? "Downloading..." : "Download Worker Agent"}
          </button>

          <p className="mt-4 text-sm text-slate-500">
            The agent runs as a background service and only uses resources when you&apos;re not actively using your machine.
          </p>
        </div>

        <div className="mt-8 rounded-xl bg-slate-800/50 p-8">
          <h2 className="text-2xl font-semibold">System Requirements</h2>
          <ul className="mt-4 space-y-2 text-slate-300">
            <li>• Windows 10/11, macOS 12+, or Linux</li>
            <li>• 512 MB available RAM</li>
            <li>• 100 MB available disk space</li>
          </ul>
        </div>
      </div>
    </main>
  );
}
