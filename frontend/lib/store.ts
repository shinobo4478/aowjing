// TEMPORARY in-memory store for the frontend shell.
//
// This exists only so the UI has something to talk to before the Go backend
// is built. It runs inside the Next.js server process, is not persisted, and
// resets whenever the dev server restarts. Phase 1 replaces every call site
// (the route handlers under app/api/) with real HTTP calls to the Go API.
//
// The `globalThis` guard keeps the data alive across dev-server hot reloads.

import { randomUUID } from "node:crypto";
import type { Profile, ProfileInput } from "./types";

interface DB {
  profiles: Map<string, Profile>;
}

const g = globalThis as unknown as { __acmpDB?: DB };

function seed(): DB {
  const now = new Date().toISOString();
  const db: DB = { profiles: new Map() };
  const samples: ProfileInput[] = [
    {
      name: "Retro Arcade Vault",
      niche: "retro gaming history",
      description: "Warm, nostalgic voiceover. 60–90s shorts on one arcade cabinet each.",
    },
    {
      name: "Quiet Kitchen",
      niche: "minimalist cooking",
      description: "No talking. Close-up ASMR prep, single dish, calm pacing.",
    },
  ];
  for (const s of samples) {
    const id = randomUUID();
    db.profiles.set(id, { id, ...s, createdAt: now, updatedAt: now });
  }
  return db;
}

const db: DB = g.__acmpDB ?? (g.__acmpDB = seed());

export function listProfiles(): Profile[] {
  return [...db.profiles.values()].sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : -1,
  );
}

export function getProfile(id: string): Profile | undefined {
  return db.profiles.get(id);
}

export function createProfile(input: ProfileInput): Profile {
  const now = new Date().toISOString();
  const profile: Profile = {
    id: randomUUID(),
    name: input.name.trim(),
    niche: input.niche.trim(),
    description: input.description.trim(),
    createdAt: now,
    updatedAt: now,
  };
  db.profiles.set(profile.id, profile);
  return profile;
}

export function updateProfile(
  id: string,
  input: ProfileInput,
): Profile | undefined {
  const existing = db.profiles.get(id);
  if (!existing) return undefined;
  const updated: Profile = {
    ...existing,
    name: input.name.trim(),
    niche: input.niche.trim(),
    description: input.description.trim(),
    updatedAt: new Date().toISOString(),
  };
  db.profiles.set(id, updated);
  return updated;
}

export function deleteProfile(id: string): boolean {
  return db.profiles.delete(id);
}
