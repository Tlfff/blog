package model

import "testing"

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{"deleted", Deleted, "已删除"},
		{"draft", Draft, "草稿"},
		{"published", Published, "已发表"},
		{"unknown", Status(99), "未知状态"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Status.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFindStatusById(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr bool
	}{
		{"deleted ok", 0, false},
		{"draft ok", 1, false},
		{"published ok", 2, false},
		{"invalid", 99, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FindStatusById(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindStatusById() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
