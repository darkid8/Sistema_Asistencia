package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"workshop/internal/domain"
)

// WarrantyView is the read model of the warranty list: the warranty plus the
// order and the plate it belongs to, and whether it is valid at a given date.
type WarrantyView struct {
	Warranty     domain.Warranty
	OrderNumber  string
	VehiclePlate string
	Valid        bool
}

// WarrantyRepository is the narrow port the warranty use case needs.
type WarrantyRepository interface {
	Save(ctx context.Context, warranty domain.Warranty) error
	FindByID(ctx context.Context, id string) (domain.Warranty, error)
	FindByInterventionAndKind(ctx context.Context, interventionID string, kind domain.WarrantyKind) (domain.Warranty, error)
	List(ctx context.Context) ([]WarrantyView, error)
	ListByVehicle(ctx context.Context, vehicleID string) ([]domain.Warranty, error)
}

// WarrantyUseCase issues coverage over an intervention and reports validity.
type WarrantyUseCase struct {
	warranty     WarrantyRepository
	intervention InterventionRepository
	newID        func() string
	now          func() time.Time
}

// NewWarrantyUseCase wires the warranty use case.
func NewWarrantyUseCase(
	warranty WarrantyRepository,
	intervention InterventionRepository,
	newID func() string,
	now func() time.Time,
) WarrantyUseCase {
	return WarrantyUseCase{warranty: warranty, intervention: intervention, newID: newID, now: now}
}

// Issue creates a warranty over an existing intervention. The expiration is
// derived by the domain from the issue date plus the coverage in months.
// Issuing the same kind of coverage twice over the same intervention is a
// conflict: the check here gives a clear message, and the unique constraint
// in storage is what makes the rule hold under a concurrent double-submit
// (e.g. an impatient double click on "issue warranty").
func (w WarrantyUseCase) Issue(ctx context.Context, interventionID string, kind domain.WarrantyKind, coverageMonthCount int) (domain.Warranty, error) {
	if _, err := w.intervention.FindByID(ctx, interventionID); err != nil {
		return domain.Warranty{}, err
	}
	if _, err := w.warranty.FindByInterventionAndKind(ctx, interventionID, kind); err == nil {
		return domain.Warranty{}, fmt.Errorf(
			"%w: a %s warranty was already issued for this intervention", domain.ErrConflict, kind,
		)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Warranty{}, err
	}
	warranty, err := domain.NewWarranty(w.newID(), interventionID, kind, coverageMonthCount, w.now())
	if err != nil {
		return domain.Warranty{}, err
	}
	if err := w.warranty.Save(ctx, warranty); err != nil {
		return domain.Warranty{}, err
	}
	return warranty, nil
}

// List returns every warranty with its validity evaluated at the consulted
// date. A zero consulted date means now.
func (w WarrantyUseCase) List(ctx context.Context, consultedAt time.Time) ([]WarrantyView, error) {
	if consultedAt.IsZero() {
		consultedAt = w.now()
	}
	view, err := w.warranty.List(ctx)
	if err != nil {
		return nil, err
	}
	for index := range view {
		view[index].Valid = view[index].Warranty.IsValidAt(consultedAt)
	}
	return view, nil
}

// ValidityAt reports whether one warranty still covers its intervention.
func (w WarrantyUseCase) ValidityAt(ctx context.Context, warrantyID string, consultedAt time.Time) (domain.Warranty, bool, error) {
	if consultedAt.IsZero() {
		consultedAt = w.now()
	}
	warranty, err := w.warranty.FindByID(ctx, warrantyID)
	if err != nil {
		return domain.Warranty{}, false, err
	}
	return warranty, warranty.IsValidAt(consultedAt), nil
}
