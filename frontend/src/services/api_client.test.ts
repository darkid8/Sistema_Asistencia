import { describe, expect, it, vi } from 'vitest';

import { ApiError, request } from './api_client';

function jsonResponse(status: number, payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('api client', () => {
  it('sends the request with credentials so the session cookie is attached', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { ok: true }));
    vi.stubGlobal('fetch', fetchMock);

    await request('/customer');

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/customer');
    expect(init.credentials).toBe('include');
  });

  it('never builds an authorization header: the HttpOnly cookie carries the session', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, {}));
    vi.stubGlobal('fetch', fetchMock);

    await request('/session', { method: 'POST', body: { username: 'admin' } });

    const [, init] = fetchMock.mock.calls[0];
    expect((init.headers as Record<string, string>).Authorization).toBeUndefined();
    expect(init.method).toBe('POST');
  });

  it('turns a backend error into the Spanish message it returned', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse(409, { code: 'conflict', message: 'El tecnico ya tiene una orden activa.' }),
      ),
    );

    await expect(request('/service-order/1/assignment', { method: 'POST' })).rejects.toMatchObject({
      status: 409,
      code: 'conflict',
      message: 'El tecnico ya tiene una orden activa.',
    });
  });

  it('reports a connection failure in Spanish instead of leaking the cause', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')));

    const failure = await request('/dashboard').catch((error: unknown) => error);

    expect(failure).toBeInstanceOf(ApiError);
    expect((failure as ApiError).message).toBe('No se pudo conectar con el servidor.');
  });
});
