import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it } from 'vitest';

import { ProtectedRoute } from './ProtectedRoute';
import { SessionProvider } from '../shared/SessionContext';
import type { Session } from '../services/session_service';

function seedSession(role: Session['role']): void {
  const session: Session = {
    token: 'a-token',
    expiresAt: new Date(Date.now() + 3600000).toISOString(),
    userId: 'user-1',
    username: role === 'ADMINISTRATOR' ? 'admin' : 'jperez',
    fullName: role === 'ADMINISTRATOR' ? 'Administrador del taller' : 'Juan Perez',
    role,
  };
  window.localStorage.setItem('workshop.session', JSON.stringify(session));
}

function renderGuarded(initialEntry: string, requiredRole?: Session['role']) {
  return render(
    <SessionProvider>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/login" element={<div>Pantalla de ingreso</div>} />
          <Route path="/dashboard" element={<div>Panel general</div>} />
          <Route
            path="/customers"
            element={
              <ProtectedRoute requiredRole={requiredRole}>
                <div>Registro de clientes</div>
              </ProtectedRoute>
            }
          />
        </Routes>
      </MemoryRouter>
    </SessionProvider>,
  );
}

describe('ProtectedRoute', () => {
  it('sends an anonymous visitor to the login screen', () => {
    renderGuarded('/customers', 'ADMINISTRATOR');
    expect(screen.getByText('Pantalla de ingreso')).toBeInTheDocument();
  });

  it('lets an administrator into an administrator-only route', () => {
    seedSession('ADMINISTRATOR');
    renderGuarded('/customers', 'ADMINISTRATOR');
    expect(screen.getByText('Registro de clientes')).toBeInTheDocument();
  });

  it('redirects a technician away from an administrator-only route', () => {
    seedSession('TECHNICIAN');
    renderGuarded('/customers', 'ADMINISTRATOR');
    expect(screen.getByText('Panel general')).toBeInTheDocument();
    expect(screen.queryByText('Registro de clientes')).not.toBeInTheDocument();
  });

  it('lets a technician into a route with no required role', () => {
    seedSession('TECHNICIAN');
    renderGuarded('/customers');
    expect(screen.getByText('Registro de clientes')).toBeInTheDocument();
  });
});
