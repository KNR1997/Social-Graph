// ============================================================
// Product Service — Data Access Layer
// ============================================================
// This is the ONLY file you modify when connecting to your backend.
// Queries (queries.ts) and components import from here — they never change.
//
// Pick your pattern and replace the function bodies below:
//
// 1. Direct external API (the natural fit for an SPA)
//    → const res = await fetch(`${import.meta.env.VITE_API_URL}/products?...`)
//    → return res.json()
//
// 2. Your own API server (Express / Hono / Fastify / Laravel / Go)
//    → Run it alongside Vite and proxy /api to it in vite.config.ts,
//      then fetch('/api/products?...') from here.
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

import { fakeProducts } from '@/constants/mock-api';
import type {
  ProductFilters,
  ProductsResponse,
  ProductByIdResponse,
  ProductMutationPayload
} from './types';

export async function getProducts(filters: ProductFilters): Promise<ProductsResponse> {
  return fakeProducts.getProducts(filters);
}

export async function getProductById(id: number): Promise<ProductByIdResponse> {
  return fakeProducts.getProductById(id) as Promise<ProductByIdResponse>;
}

export async function createProduct(data: ProductMutationPayload) {
  return fakeProducts.createProduct(data);
}

export async function updateProduct(id: number, data: ProductMutationPayload) {
  return fakeProducts.updateProduct(id, data);
}

export async function deleteProduct(id: number) {
  return fakeProducts.deleteProduct(id);
}
