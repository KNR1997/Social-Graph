/**
 * The HTTP client every feature's `api/service.ts` should call the Go API
 * through.
 *
 * Two things here are load-bearing for authentication:
 *
 * 1. `credentials: 'include'` — without it `fetch` sends no cookies, so the
 *    session cookie the API set on sign-in would never come back and every
 *    request would be anonymous.
 * 2. The default base URL is empty, meaning requests go to the same origin as
 *    the page. In development that works because `vite.config.ts` proxies
 *    `/v1` to the API on :8080. Same-origin is not a convenience: a session
 *    cookie is `SameSite=Lax`, and pointing this at a different origin would
 *    stop the browser sending it on cross-site requests.
 */

const BASE_URL = import.meta.env.VITE_API_URL ?? '';

/** The error body the Go API returns for every non-2xx response. */
interface ApiErrorBody {
  error: string;
  code: string;
}

/**
 * A non-2xx response from the API.
 *
 * `code` is the stable machine-readable string the Go handlers document
 * ('invalid_credentials', 'email_taken', …). Branch on that, never on
 * `message`, which is prose and free to be reworded.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }

  /** True when the request was rejected for want of a valid session. */
  get isUnauthorized() {
    return this.status === 401;
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
}

/**
 * Issues a request and returns the decoded JSON body, throwing `ApiError` on
 * any non-2xx response.
 *
 * The generic is an unchecked assertion about the response shape, exactly like
 * `res.json()` is. It buys call-site ergonomics, not safety.
 */
export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal } = options;

  const response = await fetch(`${BASE_URL}${path}`, {
    method,
    signal,
    credentials: 'include',
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body)
  });

  if (!response.ok) {
    throw new ApiError(response.status, ...(await readError(response)));
  }

  // 204, and any other empty body, decodes to undefined rather than exploding
  // in JSON.parse.
  if (response.status === 204 || response.headers.get('content-length') === '0') {
    return undefined as T;
  }

  return (await response.json()) as T;
}

/**
 * Pulls the code and message out of an error response.
 *
 * A failure here is itself a failure mode: a 502 from a proxy is HTML, not the
 * API's JSON error shape, and trying to parse it must not replace the real
 * status with a parse error.
 */
async function readError(response: Response): Promise<[code: string, message: string]> {
  try {
    const body = (await response.json()) as ApiErrorBody;

    if (typeof body?.error === 'string' && typeof body?.code === 'string') {
      return [body.code, body.error];
    }
  } catch {
    // Fall through to the generic message below.
  }

  return ['unknown_error', `Request failed with status ${response.status}`];
}
