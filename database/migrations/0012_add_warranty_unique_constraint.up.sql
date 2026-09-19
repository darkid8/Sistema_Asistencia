-- Prevents issuing the same kind of coverage twice over the same
-- intervention (e.g. two LABOR warranties on one job). This is the
-- authority the application-level check in warranty_usecase.go cannot be on
-- its own: a unique index is what makes the rule hold even when two
-- requests to issue the same warranty race each other.
ALTER TABLE warranty
  ADD CONSTRAINT uq_warranty_intervention_kind UNIQUE (intervention_id, warranty_kind);
