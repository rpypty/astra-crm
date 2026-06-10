package audit

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Insert(ctx context.Context, event StoredEvent) error {
	comment := pgtype.Text{}
	if event.Comment != nil {
		comment = pgtype.Text{String: *event.Comment, Valid: true}
	}

	_, err := r.queries.InsertAuditLog(ctx, db.InsertAuditLogParams{
		TeamID:            event.TeamID,
		ActorID:           event.ActorID,
		Action:            event.Action,
		EntityType:        event.EntityType,
		EntityID:          event.EntityID,
		BeforeJson:        event.BeforeJSON,
		AfterJson:         event.AfterJSON,
		ChangedFieldsJson: event.ChangedFieldsJSON,
		Comment:           comment,
	})
	return err
}
