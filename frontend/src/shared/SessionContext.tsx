/*
 * Session provider. It holds the signed in identity for the whole tree and is
 * the only consumer of the session service, so no screen reads storage.
 */
import { createContext, useCallback, useContext, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

import {
  clearStoredSession,
  readStoredSession,
  signIn as requestSignIn,
  signOut as requestSignOut,
  storeSession,
} from '../services/session_service';
import type { Session } from '../services/session_service';

interface SessionContextValue {
  session: Session | null;
  signIn: (username: string, password: string) => Promise<void>;
  signOut: () => void;
  isAdministrator: boolean;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(() => readStoredSession());

  const signIn = useCallback(async (username: string, password: string) => {
    const created = await requestSignIn(username, password);
    storeSession(created);
    // The in-memory session never needs the real token either: nothing reads
    // it anymore now that the browser attaches the HttpOnly cookie on its
    // own, so there is no reason for it to sit in React state for the rest
    // of the tab's life.
    setSession({ ...created, token: '' });
  }, []);

  const signOut = useCallback(() => {
    // The screen reacts immediately; clearing the HttpOnly cookie itself is a
    // best-effort call to the server, since a network hiccup here should
    // never trap the user in a "signed in" screen they can no longer use.
    clearStoredSession();
    setSession(null);
    void requestSignOut().catch(() => undefined);
  }, []);

  const value = useMemo<SessionContextValue>(
    () => ({
      session,
      signIn,
      signOut,
      isAdministrator: session?.role === 'ADMINISTRATOR',
    }),
    [session, signIn, signOut],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionContextValue {
  const value = useContext(SessionContext);
  if (!value) {
    throw new Error('useSession must be used inside SessionProvider');
  }
  return value;
}

/** The token of the current session, or an empty string when signed out. */
export function useToken(): string {
  return useSession().session?.token ?? '';
}
