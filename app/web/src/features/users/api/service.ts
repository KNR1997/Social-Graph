// ============================================================
// User Service — Data Access Layer
// ============================================================
// This is the ONLY file you modify when connecting to your backend.
// Queries (queries.ts) and components import from here — they never change.
//
// Pick your pattern and replace the function bodies below:
//
// 1. Direct external API (the natural fit for an SPA)
//    → const res = await fetch(`${import.meta.env.VITE_API_URL}/users?...`)
//    → return res.json()
//
// 2. Your own API server (Express / Hono / Fastify / Laravel / Go)
//    → Run it alongside Vite and proxy /api to it in vite.config.ts,
//      then fetch('/api/users?...') from here.
//
// 3. Backend-as-a-service SDK (Supabase, Firebase, Appwrite)
//    → Call the client SDK directly in each function.
//
// Note: the Next version of this file also documented server actions and
// route handlers. Neither exists in a static SPA — every call in here runs
// in the browser, so anything secret must live behind your own API.
//
// Current: Mock (in-memory fake data for demo/prototyping)
// ============================================================

import { fakeUsers } from '@/constants/mock-api-users';
import type { UserFilters, UsersResponse, UserMutationPayload } from './types';

export async function getUsers(filters: UserFilters): Promise<UsersResponse> {
  return fakeUsers.getUsers(filters);
}

export async function createUser(data: UserMutationPayload) {
  return fakeUsers.createUser(data);
}

export async function updateUser(id: number, data: UserMutationPayload) {
  return fakeUsers.updateUser(id, data);
}

export async function deleteUser(id: number) {
  return fakeUsers.deleteUser(id);
}
