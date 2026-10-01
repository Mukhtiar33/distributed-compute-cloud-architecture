import type { Metadata } from "next";
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
      <body className="antialiased">{children}</body>
    </html>
  );
}
