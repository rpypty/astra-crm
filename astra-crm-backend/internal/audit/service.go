package audit

import (
	"context"
	"fmt"
)

type Writer interface {
	Insert(ctx context.Context, event StoredEvent) error
}

type Service struct {
	writer Writer
}

func NewService(writer Writer) *Service {
	return &Service{writer: writer}
}

type Event struct {
	TeamID        int64
	ActorID       int64
	Action        string
	EntityType    string
	EntityID      string
	Before        any
	After         any
	ChangedFields any
	Comment       *string
}

type StoredEvent struct {
	TeamID            int64
	ActorID           int64
	Action            string
	EntityType        string
	EntityID          string
	BeforeJSON        []byte
	AfterJSON         []byte
	ChangedFieldsJSON []byte
	Comment           *string
}

func (s *Service) Write(ctx context.Context, event Event) error {
	before, err := MarshalRedacted(event.Before)
	if err != nil {
		return fmt.Errorf("audit: marshal before json: %w", err)
	}

	after, err := MarshalRedacted(event.After)
	if err != nil {
		return fmt.Errorf("audit: marshal after json: %w", err)
	}

	changedFields, err := MarshalRedacted(event.ChangedFields)
	if err != nil {
		return fmt.Errorf("audit: marshal changed fields json: %w", err)
	}

	return s.writer.Insert(ctx, StoredEvent{
		TeamID:            event.TeamID,
		ActorID:           event.ActorID,
		Action:            event.Action,
		EntityType:        event.EntityType,
		EntityID:          event.EntityID,
		BeforeJSON:        before,
		AfterJSON:         after,
		ChangedFieldsJSON: changedFields,
		Comment:           event.Comment,
	})
}
