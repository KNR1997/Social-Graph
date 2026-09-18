/** The signed-in user, exactly as GET /v1/auth/me returns it. */
export interface SessionUser {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  created_at: string;
  updated_at: string;
}

/**
 * The roles the API knows about.
 *
 * This mirrors the Go domain's entity.Role. Keep the two in step: an unknown
 * role here silently hides navigation rather than failing loudly.
 */
export type UserRole = 'member' | 'admin';

/**
 * What a successful sign-in returns.
 *
 * There is no token field, and that is the point: the session token is in an
 * HttpOnly cookie the browser attaches automatically, so no script — including
 * this one — can read it.
 */
export interface SessionResponse {
  user: SessionUser;
  expires_at: string;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export interface RegisterPayload {
  email: string;
  name: string;
  password: string;
}

export interface ChangePasswordPayload {
  current_password: string;
  new_password: string;
}
