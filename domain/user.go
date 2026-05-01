package domain

import "time"

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  []byte    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	IsDefault bool      `json:"is_default"`
}

type PublicUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}
