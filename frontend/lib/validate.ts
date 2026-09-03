import type { ProfileInput } from "./types";

export interface ValidationResult {
  ok: boolean;
  errors: Partial<Record<keyof ProfileInput, string>>;
  value: ProfileInput;
}

/**
 * Normalises and validates a raw request body into a ProfileInput.
 * Kept deliberately small — the Go backend will own the real validation later.
 */
export function parseProfileInput(body: unknown): ValidationResult {
  const b = (body ?? {}) as Record<string, unknown>;
  const value: ProfileInput = {
    name: typeof b.name === "string" ? b.name.trim() : "",
    niche: typeof b.niche === "string" ? b.niche.trim() : "",
    description: typeof b.description === "string" ? b.description.trim() : "",
  };

  const errors: ValidationResult["errors"] = {};
  if (value.name.length < 2) errors.name = "Name must be at least 2 characters.";
  if (value.name.length > 80) errors.name = "Name must be 80 characters or fewer.";
  if (value.niche.length < 2) errors.niche = "Niche must be at least 2 characters.";
  if (value.description.length > 600)
    errors.description = "Description must be 600 characters or fewer.";

  return { ok: Object.keys(errors).length === 0, errors, value };
}
