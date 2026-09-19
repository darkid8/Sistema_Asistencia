/* Route guard: without a session the user is sent back to the login screen.
   When requiredRole is set, a signed-in user with a different role is sent to
   the dashboard instead of seeing a screen meant for another role -- the
   backend already refuses the API calls, this just avoids showing an empty
   or broken page for a route the user has no business opening. */
import { Navigate } from 'react-router-dom';
import type { ReactNode } from 'react';

import { useSession } from '../shared/SessionContext';
import type { Session } from '../services/session_service';

export function ProtectedRoute({
  children,
  requiredRole,
}: {
  children: ReactNode;
  requiredRole?: Session['role'];
}) {
  const { session } = useSession();
  if (!session) {
    return <Navigate to="/login" replace />;
  }
  if (requiredRole && session.role !== requiredRole) {
    return <Navigate to="/dashboard" replace />;
  }
  return <>{children}</>;
}
