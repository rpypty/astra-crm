package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const defaultTeamName = "Default P2P Team"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	login := strings.TrimSpace(os.Getenv("TEAMLEAD_LOGIN"))
	if login == "" {
		return errors.New("TEAMLEAD_LOGIN is required")
	}

	password := os.Getenv("TEAMLEAD_PASSWORD")
	if password == "" {
		return errors.New("TEAMLEAD_PASSWORD is required")
	}

	teamName := strings.TrimSpace(os.Getenv("TEAM_NAME"))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer conn.Close(context.Background())

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create teamlead user: %w", err)
	}
	defer tx.Rollback(ctx)

	teamID, resolvedTeamName, err := resolveTeam(ctx, tx, teamName)
	if err != nil {
		return err
	}

	var userID int64
	err = tx.QueryRow(ctx, `
INSERT INTO users(team_id, role, login, password_hash, status)
VALUES ($1, 'teamlead', $2, $3, 'active')
RETURNING id`, teamID, login, passwordHash).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("teamlead login %q already exists", login)
		}
		return fmt.Errorf("insert teamlead user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create teamlead user: %w", err)
	}

	fmt.Printf("created teamlead user: login=%s id=%d team_id=%d team_name=%q\n", login, userID, teamID, resolvedTeamName)
	return nil
}

func resolveTeam(ctx context.Context, tx pgx.Tx, teamName string) (int64, string, error) {
	if teamName != "" {
		return findOrCreateTeamByName(ctx, tx, teamName)
	}

	rows, err := tx.Query(ctx, `
SELECT id, name
FROM teams
WHERE status = 'active'
ORDER BY id`)
	if err != nil {
		return 0, "", fmt.Errorf("select active teams: %w", err)
	}
	defer rows.Close()

	type team struct {
		id   int64
		name string
	}
	var teams []team
	for rows.Next() {
		var current team
		if err := rows.Scan(&current.id, &current.name); err != nil {
			return 0, "", fmt.Errorf("scan active team: %w", err)
		}
		teams = append(teams, current)
	}
	if err := rows.Err(); err != nil {
		return 0, "", fmt.Errorf("read active teams: %w", err)
	}

	switch len(teams) {
	case 0:
		return insertTeam(ctx, tx, defaultTeamName)
	case 1:
		return teams[0].id, teams[0].name, nil
	default:
		return 0, "", errors.New("multiple active teams found; pass TEAM_NAME or use create-user-teamlead -t")
	}
}

func findOrCreateTeamByName(ctx context.Context, tx pgx.Tx, teamName string) (int64, string, error) {
	rows, err := tx.Query(ctx, `
SELECT id
FROM teams
WHERE name = $1 AND status = 'active'
ORDER BY id`, teamName)
	if err != nil {
		return 0, "", fmt.Errorf("select team %q: %w", teamName, err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, "", fmt.Errorf("scan team %q: %w", teamName, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, "", fmt.Errorf("read team %q: %w", teamName, err)
	}

	switch len(ids) {
	case 0:
		return insertTeam(ctx, tx, teamName)
	case 1:
		return ids[0], teamName, nil
	default:
		return 0, "", fmt.Errorf("multiple active teams named %q found", teamName)
	}
}

func insertTeam(ctx context.Context, tx pgx.Tx, teamName string) (int64, string, error) {
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO teams(name, status)
VALUES ($1, 'active')
RETURNING id`, teamName).Scan(&id); err != nil {
		return 0, "", fmt.Errorf("insert team %q: %w", teamName, err)
	}
	return id, teamName, nil
}
