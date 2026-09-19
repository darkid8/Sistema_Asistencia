package usecase

import (
	"context"
	"time"

	"workshop/internal/domain"
)

// ServiceOrderSummary is the read model of the order list: the order plus the
// plate of its vehicle and the name of the technician holding it.
type ServiceOrderSummary struct {
	Order          domain.ServiceOrder
	VehiclePlate   string
	TechnicianName string
}

// ServiceOrderRepository is the narrow port the service order use case needs.
// UpdateStatus writes the order and its transition record in one transaction.
type ServiceOrderRepository interface {
	Save(ctx context.Context, order domain.ServiceOrder) error
	FindByID(ctx context.Context, id string) (domain.ServiceOrder, error)
	List(ctx context.Context, status string) ([]ServiceOrderSummary, error)
	ListByVehicle(ctx context.Context, vehicleID string) ([]domain.ServiceOrder, error)
	UpdateStatus(ctx context.Context, order domain.ServiceOrder, transition domain.StatusTransition) error
	ListTransition(ctx context.Context, serviceOrderID string) ([]domain.StatusTransition, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	NextOrderNumber(ctx context.Context) (string, error)
}

// ServiceOrderUseCase opens orders at check-in and advances their lifecycle.
type ServiceOrderUseCase struct {
	order      ServiceOrderRepository
	vehicle    VehicleRepository
	assignment AssignmentRepository
	technician TechnicianRepository
	newID      func() string
	now        func() time.Time
}

// NewServiceOrderUseCase wires the service order use case.
func NewServiceOrderUseCase(
	order ServiceOrderRepository,
	vehicle VehicleRepository,
	assignment AssignmentRepository,
	technician TechnicianRepository,
	newID func() string,
	now func() time.Time,
) ServiceOrderUseCase {
	return ServiceOrderUseCase{
		order: order, vehicle: vehicle, assignment: assignment, technician: technician, newID: newID, now: now,
	}
}

// Open registers the check-in of a vehicle and returns the created order.
func (s ServiceOrderUseCase) Open(ctx context.Context, vehicleID, reportedFailure string) (domain.ServiceOrder, error) {
	if _, err := s.vehicle.FindByID(ctx, vehicleID); err != nil {
		return domain.ServiceOrder{}, err
	}
	orderNumber, err := s.order.NextOrderNumber(ctx)
	if err != nil {
		return domain.ServiceOrder{}, err
	}
	order, err := domain.NewServiceOrder(s.newID(), orderNumber, vehicleID, reportedFailure, s.now())
	if err != nil {
		return domain.ServiceOrder{}, err
	}
	if err := s.order.Save(ctx, order); err != nil {
		return domain.ServiceOrder{}, err
	}
	return order, nil
}

// List returns the orders, optionally filtered by a lifecycle status.
func (s ServiceOrderUseCase) List(ctx context.Context, status string) ([]ServiceOrderSummary, error) {
	return s.order.List(ctx, status)
}

// Find returns one order by its identifier.
func (s ServiceOrderUseCase) Find(ctx context.Context, orderID string) (domain.ServiceOrder, error) {
	return s.order.FindByID(ctx, orderID)
}

// ListTransition returns the status history of an order.
func (s ServiceOrderUseCase) ListTransition(ctx context.Context, orderID string) ([]domain.StatusTransition, error) {
	return s.order.ListTransition(ctx, orderID)
}

// Advance moves an order to the next status. The domain rejects a move outside
// the lifecycle before anything is written, so the stored order is untouched.
// Only the administrator or the technician currently assigned to the order may
// perform the move; anyone else is refused before anything is read further.
// Reaching DELIVERED releases the technician who held the order.
func (s ServiceOrderUseCase) Advance(
	ctx context.Context, orderID string, next domain.ServiceOrderStatus, actorUserID string, actorRole domain.Role,
) (domain.ServiceOrder, error) {
	if actorRole != domain.RoleAdministrator {
		if _, err := requireAssignedTechnician(ctx, s.assignment, s.technician, orderID, actorUserID); err != nil {
			return domain.ServiceOrder{}, err
		}
	}
	order, err := s.order.FindByID(ctx, orderID)
	if err != nil {
		return domain.ServiceOrder{}, err
	}
	changedAt := s.now()
	transition, err := order.MoveTo(next, s.newID(), actorUserID, changedAt)
	if err != nil {
		return domain.ServiceOrder{}, err
	}
	if err := s.order.UpdateStatus(ctx, order, transition); err != nil {
		return domain.ServiceOrder{}, err
	}
	if next == domain.StatusDelivered {
		if err := s.assignment.ReleaseByServiceOrder(ctx, order.ID, changedAt); err != nil {
			return domain.ServiceOrder{}, err
		}
	}
	return order, nil
}
