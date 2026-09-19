package usecase

import (
	"context"
	"time"

	"workshop/internal/domain"
)

// PartUsageInput is one part row submitted with an intervention.
type PartUsageInput struct {
	PartName string
	Quantity int
}

// InterventionRepository is the narrow port the intervention use case needs.
// Save writes the intervention and its parts in one transaction.
type InterventionRepository interface {
	Save(ctx context.Context, intervention domain.Intervention) error
	FindByID(ctx context.Context, id string) (domain.Intervention, error)
	ListByServiceOrder(ctx context.Context, serviceOrderID string) ([]domain.Intervention, error)
	ListByVehicle(ctx context.Context, vehicleID string) ([]domain.Intervention, error)
}

// InterventionUseCase registers the physical work executed on a vehicle.
type InterventionUseCase struct {
	intervention InterventionRepository
	order        ServiceOrderRepository
	assignment   AssignmentRepository
	technician   TechnicianRepository
	newID        func() string
	now          func() time.Time
}

// NewInterventionUseCase wires the intervention use case.
func NewInterventionUseCase(
	intervention InterventionRepository,
	order ServiceOrderRepository,
	assignment AssignmentRepository,
	technician TechnicianRepository,
	newID func() string,
	now func() time.Time,
) InterventionUseCase {
	return InterventionUseCase{
		intervention: intervention,
		order:        order,
		assignment:   assignment,
		technician:   technician,
		newID:        newID,
		now:          now,
	}
}

// Register stores an intervention with its parts and moves the order to
// IN_REPAIR the first time work is recorded.
func (i InterventionUseCase) Register(
	ctx context.Context,
	serviceOrderID, actorUserID, description string,
	laborHourCount float64,
	part []PartUsageInput,
) (domain.Intervention, error) {
	profile, err := requireAssignedTechnician(ctx, i.assignment, i.technician, serviceOrderID, actorUserID)
	if err != nil {
		return domain.Intervention{}, err
	}
	order, err := i.order.FindByID(ctx, serviceOrderID)
	if err != nil {
		return domain.Intervention{}, err
	}
	performedAt := i.now()
	interventionID := i.newID()
	usage := make([]domain.PartUsage, 0, len(part))
	for _, item := range part {
		built, partErr := domain.NewPartUsage(i.newID(), interventionID, item.PartName, item.Quantity, performedAt)
		if partErr != nil {
			return domain.Intervention{}, partErr
		}
		usage = append(usage, built)
	}
	intervention, err := domain.NewIntervention(
		interventionID, serviceOrderID, profile.ID, description, laborHourCount, usage, performedAt,
	)
	if err != nil {
		return domain.Intervention{}, err
	}
	if order.Status == domain.StatusInDiagnosis {
		transition, moveErr := order.MoveTo(domain.StatusInRepair, i.newID(), actorUserID, performedAt)
		if moveErr != nil {
			return domain.Intervention{}, moveErr
		}
		if err := i.order.UpdateStatus(ctx, order, transition); err != nil {
			return domain.Intervention{}, err
		}
	}
	if err := i.intervention.Save(ctx, intervention); err != nil {
		return domain.Intervention{}, err
	}
	return intervention, nil
}

// ListByServiceOrder returns the interventions recorded on an order.
func (i InterventionUseCase) ListByServiceOrder(ctx context.Context, serviceOrderID string) ([]domain.Intervention, error) {
	return i.intervention.ListByServiceOrder(ctx, serviceOrderID)
}
