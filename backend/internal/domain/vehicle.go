package domain

import (
	"fmt"
	"strings"
	"time"
)

// MinimumModelYear is the earliest model year the workshop accepts.
const MinimumModelYear = 1901

// Vehicle is the automobile under service, owned by exactly one customer.
type Vehicle struct {
	ID         string
	CustomerID string
	Plate      string
	VIN        string
	Brand      string
	Model      string
	ModelYear  int
	CreatedAt  time.Time
}

// NewVehicle builds a vehicle after validating its identifying data. The plate
// and the VIN are stored uppercase so a lookup never depends on how the
// reception desk typed them.
func NewVehicle(id, customerID, plate, vin, brand, model string, modelYear int, createdAt time.Time) (Vehicle, error) {
	plate = strings.ToUpper(strings.TrimSpace(plate))
	vin = strings.ToUpper(strings.TrimSpace(vin))
	brand = strings.TrimSpace(brand)
	model = strings.TrimSpace(model)
	if id == "" {
		return Vehicle{}, fmt.Errorf("%w: vehicle identifier is required", ErrInvalidInput)
	}
	if customerID == "" {
		return Vehicle{}, fmt.Errorf("%w: the vehicle owner is required", ErrInvalidInput)
	}
	if plate == "" {
		return Vehicle{}, fmt.Errorf("%w: vehicle plate is required", ErrInvalidInput)
	}
	if vin == "" {
		return Vehicle{}, fmt.Errorf("%w: vehicle VIN is required", ErrInvalidInput)
	}
	if brand == "" || model == "" {
		return Vehicle{}, fmt.Errorf("%w: vehicle brand and model are required", ErrInvalidInput)
	}
	if modelYear < MinimumModelYear {
		return Vehicle{}, fmt.Errorf("%w: vehicle model year %d is not possible", ErrInvalidInput, modelYear)
	}
	if hasUnsafeMarkup(plate) || hasUnsafeMarkup(vin) || hasUnsafeMarkup(brand) || hasUnsafeMarkup(model) {
		return Vehicle{}, fmt.Errorf("%w: vehicle data carries unsafe characters", ErrInvalidInput)
	}
	return Vehicle{
		ID:         id,
		CustomerID: customerID,
		Plate:      plate,
		VIN:        vin,
		Brand:      brand,
		Model:      model,
		ModelYear:  modelYear,
		CreatedAt:  createdAt,
	}, nil
}
