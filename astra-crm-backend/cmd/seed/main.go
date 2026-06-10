package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

const demoPassword = "demo123"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer conn.Close(context.Background())

	passwordHash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer tx.Rollback(ctx)

	teamID, err := upsertTeam(ctx, tx)
	if err != nil {
		return err
	}
	teamleadID, err := upsertUser(ctx, tx, teamID, "teamlead", "teamlead", passwordHash)
	if err != nil {
		return err
	}

	traderIvanID, err := upsertTrader(ctx, tx, teamID, "trader_ivan", passwordHash, 75, "Bliss_OP2")
	if err != nil {
		return err
	}
	traderAnnaID, err := upsertTrader(ctx, tx, teamID, "trader_anna", passwordHash, 65, "Bliss_OP5")
	if err != nil {
		return err
	}
	traderOlegID, err := upsertTrader(ctx, tx, teamID, "trader_oleg", passwordHash, 50, "Bliss_OP7")
	if err != nil {
		return err
	}

	requisiteIvanID, err := upsertRequisite(ctx, tx, teamID, teamleadID, "+79991234567", "SBP", "192.168.1.1:8080")
	if err != nil {
		return err
	}
	requisiteAnnaID, err := upsertRequisite(ctx, tx, teamID, teamleadID, "+79997654321", "C2C", "10.2.0.14:9000")
	if err != nil {
		return err
	}
	requisiteOlegID, err := upsertRequisite(ctx, tx, teamID, teamleadID, "+79995554433", "SBP", "10.2.0.18:9000")
	if err != nil {
		return err
	}

	if err := ensureAssignment(ctx, tx, teamID, requisiteIvanID, traderIvanID, teamleadID, "Локальный seed: дневная смена"); err != nil {
		return err
	}
	if err := ensureAssignment(ctx, tx, teamID, requisiteAnnaID, traderAnnaID, teamleadID, "Локальный seed: дневная смена"); err != nil {
		return err
	}
	if err := ensureAssignment(ctx, tx, teamID, requisiteOlegID, traderOlegID, teamleadID, "Локальный seed: резерв"); err != nil {
		return err
	}

	if err := ensureAccountingPeriod(ctx, tx, teamID, teamleadID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}

	fmt.Println("seed applied")
	fmt.Println("demo users: teamlead / trader_ivan / trader_anna / trader_oleg")
	fmt.Println("demo password: demo123")
	return nil
}

func upsertTeam(ctx context.Context, tx pgx.Tx) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM teams WHERE name = 'Demo P2P Team' LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select team: %w", err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO teams(name, status)
VALUES ('Demo P2P Team', 'active')
RETURNING id`).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert team: %w", err)
	}
	return id, nil
}

func upsertUser(ctx context.Context, tx pgx.Tx, teamID int64, role string, login string, passwordHash string) (int64, error) {
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO users(team_id, role, login, password_hash, status)
VALUES ($1, $2, $3, $4, 'active')
ON CONFLICT (team_id, login)
DO UPDATE SET role = EXCLUDED.role, password_hash = EXCLUDED.password_hash, status = 'active', updated_at = now(), deleted_at = NULL
RETURNING id`, teamID, role, login, passwordHash).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert user %s: %w", login, err)
	}
	return id, nil
}

func upsertTrader(ctx context.Context, tx pgx.Tx, teamID int64, login string, passwordHash string, salaryRateBps int64, workerName string) (int64, error) {
	userID, err := upsertUser(ctx, tx, teamID, "trader", login, passwordHash)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO trader_profiles(user_id, salary_rate_bps, external_worker_name)
VALUES ($1, $2, $3)
ON CONFLICT (user_id)
DO UPDATE SET salary_rate_bps = EXCLUDED.salary_rate_bps, external_worker_name = EXCLUDED.external_worker_name, updated_at = now()`,
		userID, salaryRateBps, workerName); err != nil {
		return 0, fmt.Errorf("upsert trader profile %s: %w", login, err)
	}
	return userID, nil
}

func upsertRequisite(ctx context.Context, tx pgx.Tx, teamID int64, createdBy int64, phone string, methodType string, proxy string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
INSERT INTO requisites(team_id, phone, method_type, proxy, status, created_by)
SELECT $1, $2, $3, $4, 'active', $5
WHERE NOT EXISTS (SELECT 1 FROM requisites WHERE team_id = $1 AND phone = $2)
RETURNING id`, teamID, phone, methodType, proxy, createdBy).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return 0, fmt.Errorf("insert requisite %s: %w", phone, err)
	}
	if err := tx.QueryRow(ctx, `
UPDATE requisites
SET method_type = $3, proxy = $4, status = 'active', updated_at = now(), deleted_at = NULL
WHERE team_id = $1 AND phone = $2
RETURNING id`, teamID, phone, methodType, proxy).Scan(&id); err != nil {
		return 0, fmt.Errorf("update requisite %s: %w", phone, err)
	}
	return id, nil
}

func ensureAssignment(ctx context.Context, tx pgx.Tx, teamID int64, requisiteID int64, traderID int64, assignedBy int64, comment string) error {
	var activeTraderID int64
	err := tx.QueryRow(ctx, `
SELECT trader_id
FROM requisite_assignments
WHERE requisite_id = $1 AND unassigned_at IS NULL`, requisiteID).Scan(&activeTraderID)
	if err == nil && activeTraderID == traderID {
		return nil
	}
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("select active assignment for requisite %d: %w", requisiteID, err)
	}
	if err == nil {
		if _, err := tx.Exec(ctx, `
UPDATE requisite_assignments
SET unassigned_at = now()
WHERE requisite_id = $1 AND unassigned_at IS NULL`, requisiteID); err != nil {
			return fmt.Errorf("close old assignment for requisite %d: %w", requisiteID, err)
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO requisite_assignments(team_id, requisite_id, trader_id, assigned_by, comment)
VALUES ($1, $2, $3, $4, $5)`, teamID, requisiteID, traderID, assignedBy, comment); err != nil {
		return fmt.Errorf("insert assignment for requisite %d: %w", requisiteID, err)
	}
	return nil
}

func ensureAccountingPeriod(ctx context.Context, tx pgx.Tx, teamID int64, createdBy int64) error {
	_, err := tx.Exec(ctx, `
INSERT INTO accounting_periods(team_id, date_from, date_to, status, created_by)
SELECT $1, DATE '2026-06-08', DATE '2026-06-14', 'open', $2
WHERE NOT EXISTS (
	SELECT 1 FROM accounting_periods WHERE team_id = $1 AND date_from = DATE '2026-06-08' AND date_to = DATE '2026-06-14'
)`, teamID, createdBy)
	if err != nil {
		return fmt.Errorf("ensure accounting period: %w", err)
	}
	return nil
}
