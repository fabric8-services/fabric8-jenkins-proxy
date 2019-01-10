package wit

// DefaultMockOwner is a mock owner value to be used by tests
const DefaultMockOwner = "mockTenantID"

// Mock implementation of WIT Service
type Mock struct {
	OwnedBy string
}

// SearchCodebase is a mock method
func (w *Mock) SearchCodebase(repo string) (*Info, error) {
	return &Info{
		OwnedBy: w.OwnedBy,
	}, nil
}
