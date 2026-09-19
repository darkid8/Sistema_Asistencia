/*
 * Session service: it signs the user in and is the only place that reads or
 * writes the stored session, so no component touches browser storage.
 *
 * The session token itself is no longer part of what this module persists.
 * The backend sets it as an HttpOnly cookie the browser attaches to every
 * request on its own; storing a copy in localStorage would defeat the point
 * of HttpOnly (any injected script could read it back out through that copy)
 * for no benefit, since nothing here needs to read the token to use it.
 */
import { request } from './api_client';

const STORAGE_KEY = 'workshop.session';

export interface Session {
  /**
   * @deprecated Kept only for the response shape the backend still returns
   * and for older call sites that read `session.token`. It is blanked out
   * before the session is persisted (see storeSession) and is never sent
   * back to the server -- the HttpOnly cookie carries the real credential.
   */
  token: string;
  expiresAt: string;
  userId: string;
  username: string;
  fullName: string;
  role: 'ADMINISTRATOR' | 'TECHNICIAN';
}

export async function signIn(username: string, password: string): Promise<Session> {
  return request<Session>('/session', { method: 'POST', body: { username, password } });
}

/** Clears the HttpOnly session cookie on the server. A script cannot delete
 * that cookie on its own, so signing out has to be a request, not just
 * forgetting the local copy of the session. */
export async function signOut(): Promise<void> {
  await request<void>('/session/sign-out', { method: 'POST' });
}

export function readStoredSession(): Session | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const stored = JSON.parse(raw) as Session;
    if (!stored.userId || !stored.role || new Date(stored.expiresAt).getTime() <= Date.now()) {
      window.localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return stored;
  } catch {
    return null;
  }
}

export function storeSession(session: Session): void {
  try {
    // Never write the token to localStorage: keep everything else, which is
    // only display data (name, role, expiry), so the UI can restore a
    // signed-in look after a refresh without holding the credential itself.
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ ...session, token: '' }));
  } catch {
    // A browser with storage disabled still works for the current tab.
  }
}

export function clearStoredSession(): void {
  try {
    window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing to clean when storage is unavailable.
  }
}
