import { NextResponse } from "next/server";
import { existsSync, readFileSync } from "fs";
import path from "path";

export async function GET() {
  const binaryPath = path.join(process.cwd(), "public", "worker-agent");

  if (!existsSync(binaryPath)) {
    return NextResponse.json(
      { error: "Worker Agent binary not found. Please build it first." },
      { status: 404 }
    );
  }

  const binary = readFileSync(binaryPath);

  return new NextResponse(binary, {
    headers: {
      "Content-Type": "application/octet-stream",
      "Content-Disposition": 'attachment; filename="worker-agent"',
    },
  });
}
