import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Distributed Compute Cloud",
  description: "Transform idle personal computers into a secure, trustworthy distributed compute cloud",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="antialiased">
        <nav className="bg-slate-900/80 backdrop-blur-sm border-b border-slate-700">
          <div className="mx-auto max-w-6xl px-6 py-4 flex items-center justify-between">
            <Link href="/" className="text-xl font-bold text-white">
              Distributed Compute Cloud
            </Link>
            <div className="flex items-center gap-6">
              <Link href="/submit" className="text-slate-300 hover:text-white transition-colors">
                Submit Job
              </Link>
              <Link href="/download" className="text-slate-300 hover:text-white transition-colors">
                Download
              </Link>
              <Link href="/account" className="text-slate-300 hover:text-white transition-colors">
                Account
              </Link>
            </div>
          </div>
        </nav>
        {children}
      </body>
    </html>
  );
}
