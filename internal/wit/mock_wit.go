package wit

// Mock implementation of WIT Service
type Mock struct {
}

// SearchCodebase is a mock method
func (w *Mock) SearchCodebase(repo string) (*Info, error) {
	return &Info{
		OwnedBy: "mockTenantID",
	}, nil
}
