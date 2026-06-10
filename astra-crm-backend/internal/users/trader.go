package users

import "time"

type TraderProfile struct {
	ID                 int64
	UserID             int64
	SalaryRateBps      int64
	ExternalWorkerName string
}

type Trader struct {
	ID                 int64
	TeamID             int64
	Role               string
	Login              string
	Status             string
	SalaryRateBps      int64
	ExternalWorkerName string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type PublicTrader struct {
	ID                 int64     `json:"id"`
	TeamID             int64     `json:"teamId"`
	Role               string    `json:"role"`
	Login              string    `json:"login"`
	Status             string    `json:"status"`
	SalaryRateBps      int64     `json:"salaryRateBps"`
	ExternalWorkerName string    `json:"externalWorkerName"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func ToPublicTrader(trader Trader) PublicTrader {
	return PublicTrader{
		ID:                 trader.ID,
		TeamID:             trader.TeamID,
		Role:               trader.Role,
		Login:              trader.Login,
		Status:             trader.Status,
		SalaryRateBps:      trader.SalaryRateBps,
		ExternalWorkerName: trader.ExternalWorkerName,
		CreatedAt:          trader.CreatedAt,
		UpdatedAt:          trader.UpdatedAt,
	}
}

func ToPublicTraders(traders []Trader) []PublicTrader {
	items := make([]PublicTrader, 0, len(traders))
	for _, trader := range traders {
		items = append(items, ToPublicTrader(trader))
	}

	return items
}
