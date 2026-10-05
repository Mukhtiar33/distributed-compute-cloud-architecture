import { NextResponse } from "next/server";

export async function POST(request: Request) {
  try {
    const formData = await request.formData();
    const manifest = formData.get("manifest");
    const zip = formData.get("zip");

    if (!manifest || !zip) {
      return NextResponse.json(
        { error: "manifest and zip are required" },
        { status: 400 }
      );
    }

    // Forward to Control Plane
    const controlPlaneUrl = process.env.CONTROL_PLANE_URL || "https://localhost:8443";

    const forwardFormData = new FormData();
    forwardFormData.append("manifest", manifest as string);
    forwardFormData.append("zip", zip as Blob);

    const response = await fetch(`${controlPlaneUrl}/jobs`, {
      method: "POST",
      body: forwardFormData,
    });

    const data = await response.json();

    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data, { status: 201 });
  } catch (error) {
    return NextResponse.json(
      { error: "Failed to submit job" },
      { status: 500 }
    );
  }
}
