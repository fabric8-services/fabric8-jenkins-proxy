package clients

type MockWit struct {
}

// SearchCodebase is a mock method
func (w *MockWit) SearchCodebase(repo string) (*WITInfo, error) {
	return &WITInfo{
		OwnedBy: "mockTenantID",
	}, nil
}
