package detector

import (
	"context"
	"testing"

	"aiops-platform/internal/types"
)

func TestStopTwiceDoesNotPanic(t *testing.T) {
	cfg := &types.Config{
		Server: types.ServerConfig{
			Host: "127.0.0.1",
			Port: 0,
		},
	}
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create detector: %v", err)
	}

	ctx := context.Background()
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("first stop returned error: %v", err)
	}
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("second stop returned error: %v", err)
	}
}
