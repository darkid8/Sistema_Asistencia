import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { InterventionPanel } from './InterventionPanel';
import { SessionProvider } from '../../shared/SessionContext';

const INTERVENTION = {
  id: 'intervention-1',
  serviceOrderId: 'order-1',
  technicianId: 'technician-1',
  description: 'Cambio de bujias',
  laborHourCount: 1.5,
  performedAt: '2026-03-02T10:00:00Z',
  part: [],
};

const LABOR_WARRANTY = {
  id: 'warranty-1',
  interventionId: 'intervention-1',
  orderNumber: 'OS-0001',
  vehiclePlate: 'ABC123',
  kind: 'LABOR',
  coverageMonthCount: 12,
  issuedAt: '2026-03-02T10:00:00Z',
  expirationDate: '2027-03-02T10:00:00Z',
  valid: true,
};

function storeAdministratorSession() {
  window.localStorage.setItem(
    'workshop.session',
    JSON.stringify({
      token: 'a-token',
      expiresAt: new Date(Date.now() + 3600000).toISOString(),
      userId: 'user-1',
      username: 'admin',
      fullName: 'Administrador del taller',
      role: 'ADMINISTRATOR',
    }),
  );
}

function jsonResponse(payload: unknown) {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

function stubApi(options: { warranty?: unknown[]; onIssue?: (body: unknown) => void }) {
  const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    if (url.startsWith('/api/warranty') && init?.method === 'POST') {
      options.onIssue?.(init.body ? JSON.parse(String(init.body)) : null);
      return Promise.resolve(
        jsonResponse({
          id: 'warranty-2',
          interventionId: 'intervention-1',
          kind: 'PART',
          coverageMonthCount: 6,
          issuedAt: '2026-03-02T10:00:00Z',
          expirationDate: '2026-09-02T10:00:00Z',
          valid: true,
        }),
      );
    }
    if (url.startsWith('/api/warranty')) {
      return Promise.resolve(jsonResponse(options.warranty ?? []));
    }
    if (url.startsWith('/api/service-order/order-1/intervention')) {
      return Promise.resolve(jsonResponse([INTERVENTION]));
    }
    return Promise.resolve(jsonResponse([]));
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function renderPanel() {
  return render(
    <SessionProvider>
      <InterventionPanel serviceOrderId="order-1" onChange={() => undefined} />
    </SessionProvider>,
  );
}

describe('warranty issuance in the intervention panel', () => {
  it('lets the administrator choose the kind and the coverage in months', async () => {
    storeAdministratorSession();
    stubApi({ warranty: [] });
    renderPanel();

    expect(await screen.findByText('Cambio de bujias')).toBeInTheDocument();
    expect(screen.getByLabelText('Tipo de garantia para Cambio de bujias')).toBeInTheDocument();
    expect(screen.getByLabelText('Meses de cobertura para Cambio de bujias')).toHaveValue(12);
  });

  it('sends the chosen kind and months instead of a fixed 12-month LABOR warranty', async () => {
    storeAdministratorSession();
    let sentBody: unknown = null;
    stubApi({ warranty: [], onIssue: (body) => { sentBody = body; } });
    renderPanel();

    await screen.findByText('Cambio de bujias');
    await userEvent.selectOptions(
      screen.getByLabelText('Tipo de garantia para Cambio de bujias'),
      'PART',
    );
    await userEvent.clear(screen.getByLabelText('Meses de cobertura para Cambio de bujias'));
    await userEvent.type(screen.getByLabelText('Meses de cobertura para Cambio de bujias'), '6');
    await userEvent.click(screen.getByRole('button', { name: 'Emitir garantia' }));

    expect(await screen.findByText('Garantia de Repuesto emitida por 6 meses.')).toBeInTheDocument();
    expect(sentBody).toEqual({ interventionId: 'intervention-1', kind: 'PART', coverageMonthCount: 6 });
  });

  it('disables the button for a kind already covered, instead of allowing a duplicate', async () => {
    storeAdministratorSession();
    stubApi({ warranty: [LABOR_WARRANTY] });
    renderPanel();

    await screen.findByText('Cambio de bujias');
    const alreadyCovered = await screen.findByRole('button', { name: 'Ya cubierta' });
    expect(alreadyCovered).toBeDisabled();
  });
});
