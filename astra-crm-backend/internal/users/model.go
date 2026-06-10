package users

const (
	RoleTeamlead = "teamlead"
	RoleTrader   = "trader"

	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusDeleted  = "deleted"
)

type User struct {
	ID           int64
	TeamID       int64
	Role         string
	Login        string
	PasswordHash string
	Status       string
}

func (u User) IsActive() bool {
	return u.Status == StatusActive
}

type PublicUser struct {
	ID     int64  `json:"id"`
	TeamID int64  `json:"teamId"`
	Role   string `json:"role"`
	Login  string `json:"login"`
	Status string `json:"status"`
}

func ToPublic(u User) PublicUser {
	return PublicUser{
		ID:     u.ID,
		TeamID: u.TeamID,
		Role:   u.Role,
		Login:  u.Login,
		Status: u.Status,
	}
}
