import { NextResponse } from "next/server";
import { createProfile, listProfiles } from "@/lib/store";
import { parseProfileInput } from "@/lib/validate";

// GET /api/profiles — list all profiles
export async function GET() {
  return NextResponse.json({ profiles: listProfiles() });
}

// POST /api/profiles — create a profile
export async function POST(request: Request) {
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

  const profile = createProfile(value);
  return NextResponse.json({ profile }, { status: 201 });
}
