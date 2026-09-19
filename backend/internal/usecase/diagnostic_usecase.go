package usecase

import (
	"context"
	"time"

	"workshop/internal/domain"
)

// DiagnosticRepository is the narrow port the diagnostic use case needs.
type DiagnosticRepository interface {
	Save(ctx context.Context, diagnostic domain.Diagnostic) error
	FindByServiceOrder(ctx context.Context, serviceOrderID string) (domain.Diagnostic, error)
	ListByVehicle(ctx context.Context, vehicleID string) ([]domain.Diagnostic, error)
}

// DiagnosticUseCase records the technical evaluation of a service order.
type DiagnosticUseCase struct {
	diagnostic DiagnosticRepository
	order      ServiceOrderRepository
	assignment AssignmentRepository
	technician TechnicianRepository
	newID      func() string
	now        func() time.Time
}

// NewDiagnosticUseCase wires the diagnostic use case.
func NewDiagnosticUseCase(
	diagnostic DiagnosticRepository,
	order ServiceOrderRepository,
	assignment AssignmentRepository,
	technician TechnicianRepository,
	newID func() string,
	now func() time.Time,
) DiagnosticUseCase {
	return DiagnosticUseCase{
		diagnostic: diagnostic,
		order:      order,
		assignment: assignment,
		technician: technician,
		newID:      newID,
		now:        now,
	}
}

// Record stores the diagnostic written by the assigned technician and moves
// the order to IN_DIAGNOSIS. A technician who does not hold the order is
// refused before anything is written.
func (d DiagnosticUseCase) Record(ctx context.Context, serviceOrderID, actorUserID, finding, componentToRepair string) (domain.Diagnostic, error) {
	profile, err := requireAssignedTechnician(ctx, d.assignment, d.technician, serviceOrderID, actorUserID)
	if err != nil {
		return domain.Diagnostic{}, err
	}
	order, err := d.order.FindByID(ctx, serviceOrderID)
	if err != nil {
		return domain.Diagnostic{}, err
	}
	diagnostic, err := domain.NewDiagnostic(d.newID(), serviceOrderID, profile.ID, finding, componentToRepair, d.now())
	if err != nil {
		return domain.Diagnostic{}, err
	}
	if order.Status == domain.StatusReceived {
		changedAt := d.now()
		transition, moveErr := order.MoveTo(domain.StatusInDiagnosis, d.newID(), actorUserID, changedAt)
		if moveErr != nil {
			return domain.Diagnostic{}, moveErr
		}
		if err := d.order.UpdateStatus(ctx, order, transition); err != nil {
			return domain.Diagnostic{}, err
		}
	}
	if err := d.diagnostic.Save(ctx, diagnostic); err != nil {
		return domain.Diagnostic{}, err
	}
	return diagnostic, nil
}

// FindByServiceOrder returns the diagnostic of an order when it exists.
func (d DiagnosticUseCase) FindByServiceOrder(ctx context.Context, serviceOrderID string) (domain.Diagnostic, error) {
	return d.diagnostic.FindByServiceOrder(ctx, serviceOrderID)
}
