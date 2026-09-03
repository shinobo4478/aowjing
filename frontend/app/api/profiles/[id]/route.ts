import { NextResponse } from "next/server";
import { deleteProfile, getProfile, updateProfile } from "@/lib/store";
import { parseProfileInput } from "@/lib/validate";

type Params = { params: Promise<{ id: string }> };

// GET /api/profiles/:id
export async function GET(_request: Request, { params }: Params) {
  const { id } = await params;
  const profile = getProfile(id);
  if (!profile) {
    return NextResponse.json({ error: "Profile not found." }, { status: 404 });
  }
  return NextResponse.json({ profile });
}

// PATCH /api/profiles/:id — replace the editable fields
export async function PATCH(request: Request, { params }: Params) {
  const { id } = await params;
  if (!getProfile(id)) {
    return NextResponse.json({ error: "Profile not found." }, { status: 404 });
  }

  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON body." }, { status: 400 });
  }

  const { ok, errors, value } = parseProfileInput(body);
  if (!ok) {
    return NextResponse.json({ errors }, { status: 422 });
  }

  const profile = updateProfile(id, value);
  return NextResponse.json({ profile });
}

// DELETE /api/profiles/:id
export async function DELETE(_request: Request, { params }: Params) {
  const { id } = await params;
  if (!deleteProfile(id)) {
    return NextResponse.json({ error: "Profile not found." }, { status: 404 });
  }
  return new NextResponse(null, { status: 204 });
}
