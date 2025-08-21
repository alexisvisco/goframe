package genscaffold

import (
	"fmt"
)

// GenerateCRUDRoutes generates full CRUD routes for a scaffold
func (s *ScaffoldGenerator) GenerateCRUDRoutes(name string, fields []Field) error {
	if s.NoHandler {
		return nil
	}

	if err := s.generateCreateRoute(name, fields); err != nil {
		return fmt.Errorf("failed to generate create route: %w", err)
	}

	if err := s.generateGetByIDRoute(name); err != nil {
		return fmt.Errorf("failed to generate get by ID route: %w", err)
	}

	if err := s.generateFindAllRoute(name); err != nil {
		return fmt.Errorf("failed to generate find all route: %w", err)
	}

	if err := s.generatePatchRoute(name, fields); err != nil {
		return fmt.Errorf("failed to generate patch route: %w", err)
	}

	if err := s.generateDeleteRoute(name); err != nil {
		return fmt.Errorf("failed to generate delete route: %w", err)
	}

	if err := s.HTTPGen.Update(); err != nil {
		return fmt.Errorf("failed to generate HTTP routes: %w", err)
	}

	return nil
}
