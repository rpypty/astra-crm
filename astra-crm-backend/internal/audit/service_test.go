package audit

import (
	"context"
	"strings"
	"testing"
)

func TestServiceWriteRedactsPayloadBeforeInsert(t *testing.T) {
	writer := &captureWriter{}
	service := NewService(writer)

	err := service.Write(context.Background(), Event{
		TeamID:     1,
		ActorID:    2,
		Action:     ActionUserPasswordReset,
		EntityType: "user",
		EntityID:   "3",
		After: map[string]any{
			"password": "new-password",
		},
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if strings.Contains(string(writer.event.AfterJSON), "new-password") {
		t.Fatalf("sensitive value leaked into audit json: %s", string(writer.event.AfterJSON))
	}
}

type captureWriter struct {
	event StoredEvent
}

func (w *captureWriter) Insert(ctx context.Context, event StoredEvent) error {
	w.event = event
	return nil
}
