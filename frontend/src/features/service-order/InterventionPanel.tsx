/* Intervention panel: the work executed on the vehicle, with a repeatable part
   row, plus the action that issues a warranty over a recorded intervention. */
import { useState } from 'react';

import { ApiError } from '../../services/api_client';
import { listIntervention, registerIntervention } from '../../services/service_order_service';
import type { PartUsage } from '../../services/service_order_service';
import { issueWarranty, listWarranty } from '../../services/warranty_service';
import type { WarrantyKind } from '../../services/warranty_service';
import { DataState, ErrorBanner, SuccessBanner } from '../../shared/DataState';
import { useAsyncData } from '../../shared/useAsyncData';
import { useSession, useToken } from '../../shared/SessionContext';
import { formatDateTime } from '../../shared/format';

interface InterventionPanelProps {
  serviceOrderId: string;
  onChange: () => void;
}

interface WarrantyDraft {
  kind: WarrantyKind;
  monthCount: string;
}

const EMPTY_PART: PartUsage = { partName: '', quantity: 1 };
const DEFAULT_WARRANTY_DRAFT: WarrantyDraft = { kind: 'LABOR', monthCount: '12' };
const WARRANTY_KIND_LABEL: Record<WarrantyKind, string> = { LABOR: 'Mano de obra', PART: 'Repuesto' };

export function InterventionPanel({ serviceOrderId, onChange }: InterventionPanelProps) {
  const token = useToken();
  const { isAdministrator } = useSession();
  const intervention = useAsyncData(
    () => listIntervention(token, serviceOrderId),
    [token, serviceOrderId],
  );
  // Every warranty ever issued, only to know which (intervention, kind) pairs
  // are already covered -- not filtered by date, coverage here is about
  // existence, not current validity. Technicians never see this panel's
  // warranty controls, so skip the call entirely for them instead of hitting
  // an endpoint that would answer 403.
  const warrantyList = useAsyncData(
    () => (isAdministrator ? listWarranty(token, '') : Promise.resolve([])),
    [token, isAdministrator],
  );
  const [description, setDescription] = useState('');
  const [laborHourCount, setLaborHourCount] = useState('1');
  const [part, setPart] = useState<PartUsage[]>([EMPTY_PART]);
  const [warrantyDraft, setWarrantyDraft] = useState<Record<string, WarrantyDraft>>({});
  const [issuingId, setIssuingId] = useState('');
  const [error, setError] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [sending, setSending] = useState(false);

  const coveredKind = new Set(
    (warrantyList.data ?? []).map((item) => item.interventionId + ':' + item.kind),
  );
  const draftFor = (interventionId: string): WarrantyDraft =>
    warrantyDraft[interventionId] ?? DEFAULT_WARRANTY_DRAFT;
  const updateDraft = (interventionId: string, patch: Partial<WarrantyDraft>) =>
    setWarrantyDraft((previous) => ({
      ...previous,
      [interventionId]: { ...draftFor(interventionId), ...patch },
    }));

  const updatePart = (index: number, field: keyof PartUsage, value: string) =>
    setPart((previous) =>
      previous.map((item, position) =>
        position === index
          ? { ...item, [field]: field === 'quantity' ? Number(value) : value }
          : item,
      ),
    );

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    setConfirmation('');
    setSending(true);
    try {
      await registerIntervention(
        token,
        serviceOrderId,
        description,
        Number(laborHourCount),
        part.filter((item) => item.partName.trim().length > 0),
      );
      setDescription('');
      setLaborHourCount('1');
      setPart([EMPTY_PART]);
      setConfirmation('Intervencion registrada.');
      intervention.reload();
      onChange();
    } catch (failure) {
      setError(
        failure instanceof ApiError ? failure.message : 'No se pudo registrar la intervencion.',
      );
    } finally {
      setSending(false);
    }
  };

  const issue = async (interventionId: string) => {
    const draft = draftFor(interventionId);
    const monthCount = Number(draft.monthCount);
    if (!Number.isInteger(monthCount) || monthCount <= 0) {
      setError('La cobertura en meses debe ser un numero entero mayor que cero.');
      return;
    }
    setError('');
    setConfirmation('');
    // Guards the same double click the backend now also refuses: this just
    // saves the round trip and gives immediate feedback instead of a 409.
    setIssuingId(interventionId + ':' + draft.kind);
    try {
      await issueWarranty(token, interventionId, draft.kind, monthCount);
      setConfirmation('Garantia de ' + WARRANTY_KIND_LABEL[draft.kind] + ' emitida por ' + monthCount + ' meses.');
      warrantyList.reload();
    } catch (failure) {
      setError(failure instanceof ApiError ? failure.message : 'No se pudo emitir la garantia.');
    } finally {
      setIssuingId('');
    }
  };

  return (
    <section className="card">
      <h3 className="card__title">Intervenciones</h3>
      <ErrorBanner message={error} />
      <SuccessBanner message={confirmation} />

      {isAdministrator ? null : (
        <form onSubmit={submit} noValidate>
          <div className="form-grid">
            <div className="field">
              <label className="field__label" htmlFor="description">
                Descripcion
              </label>
              <input
                className="field__input"
                id="description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                required
              />
            </div>
            <div className="field">
              <label className="field__label" htmlFor="laborHourCount">
                Horas de trabajo
              </label>
              <input
                className="field__input"
                id="laborHourCount"
                type="number"
                min="0.5"
                step="0.5"
                value={laborHourCount}
                onChange={(event) => setLaborHourCount(event.target.value)}
                required
              />
            </div>
          </div>
          {part.map((item, index) => (
            <div className="form-grid" key={index}>
              <div className="field">
                <label className="field__label" htmlFor={'partName-' + index}>
                  Repuesto
                </label>
                <input
                  className="field__input"
                  id={'partName-' + index}
                  value={item.partName}
                  onChange={(event) => updatePart(index, 'partName', event.target.value)}
                />
              </div>
              <div className="field">
                <label className="field__label" htmlFor={'quantity-' + index}>
                  Cantidad
                </label>
                <input
                  className="field__input"
                  id={'quantity-' + index}
                  type="number"
                  min="1"
                  value={String(item.quantity)}
                  onChange={(event) => updatePart(index, 'quantity', event.target.value)}
                />
              </div>
            </div>
          ))}
          <button
            type="button"
            className="button button--secondary"
            onClick={() => setPart((previous) => [...previous, { ...EMPTY_PART }])}
          >
            Agregar repuesto
          </button>
          <button type="submit" className="button button--primary" disabled={sending}>
            {sending ? 'Registrando...' : 'Registrar intervencion'}
          </button>
        </form>
      )}

      <DataState
        loading={intervention.loading}
        error={intervention.error}
        empty={(intervention.data ?? []).length === 0}
        emptyMessage="Esta orden aun no tiene intervenciones."
      >
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Descripcion</th>
                <th scope="col">Horas</th>
                <th scope="col">Repuestos</th>
                <th scope="col">Fecha</th>
                <th scope="col">Garantia</th>
              </tr>
            </thead>
            <tbody>
              {(intervention.data ?? []).map((item) => {
                const draft = draftFor(item.id);
                const alreadyCovered = coveredKind.has(item.id + ':' + draft.kind);
                const busy = issuingId === item.id + ':' + draft.kind;
                return (
                  <tr key={item.id}>
                    <td>{item.description}</td>
                    <td>{item.laborHourCount}</td>
                    <td>
                      {item.part.length === 0
                        ? 'Sin repuestos'
                        : item.part.map((used) => used.partName).join(', ')}
                    </td>
                    <td>{formatDateTime(item.performedAt)}</td>
                    <td>
                      {isAdministrator ? (
                        <div className="warranty-issue">
                          <select
                            aria-label={'Tipo de garantia para ' + item.description}
                            value={draft.kind}
                            onChange={(event) =>
                              updateDraft(item.id, { kind: event.target.value as WarrantyKind })
                            }
                          >
                            <option value="LABOR">Mano de obra</option>
                            <option value="PART">Repuesto</option>
                          </select>
                          <input
                            aria-label={'Meses de cobertura para ' + item.description}
                            type="number"
                            min="1"
                            step="1"
                            value={draft.monthCount}
                            onChange={(event) => updateDraft(item.id, { monthCount: event.target.value })}
                            disabled={alreadyCovered}
                          />
                          <button
                            type="button"
                            className="button button--secondary"
                            onClick={() => issue(item.id)}
                            disabled={alreadyCovered || busy}
                          >
                            {alreadyCovered
                              ? 'Ya cubierta'
                              : busy
                                ? 'Emitiendo...'
                                : 'Emitir garantia'}
                          </button>
                        </div>
                      ) : null}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </DataState>
    </section>
  );
}
