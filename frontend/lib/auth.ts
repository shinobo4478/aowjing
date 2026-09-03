// Auth calls against the Go backend's /auth/* endpoints. The session is an
// HttpOnly cookie, so the frontend can only learn its state by calling getMe().

import { request } from "./api";

export interface User {
  username: string;
}

export function login(username: string, password: string) {
  return request<{ user: User }>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  }).then((r) => r.user);
}

export function logout() {
  return request<void>("/auth/logout", { method: "POST" });
}

export function getMe() {
  return request<{ user: User }>("/auth/me").then((r) => r.user);
}
