package model

type User struct {
	ID             int     `json:"id"`
	Name           string  `json:"name" validate:"required"`
	Age            int     `json:"age" validate:"gte=0,lte=130"`
	Email          string  `json:"email" validate:"required, email"`
	Role           string  `json:"role"`
	HashedPassword string  // не указывать json:"..." — не придет снаружи
	Balance        float64 `json:"balance" validate:"min=0"`
}

// PartialUser — используется для PATCH-запросов
type PartialUser struct {
	ID             int      `json:"id"`
	Name           *string  `json:"name,omitempty" validate:"required"`
	Age            *int     `json:"age,omitempty" validate:"gte=0,lte=130"`
	Email          *string  `json:"email,omitempty" validate:"required, email"`
	HashedPassword *string  // не указывать json:"..." — не придет снаружи
	Balance        *float64 `json:"balance" validate:"min=0"`
}

type RegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Age      int    `json:"age" validate:"gte=0,lte=130"`
	Email    string `json:"email" validate:"required, email"`
	Password string `json:"password" validate:"required, min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required, email"`
	Password string `json:"password" validate:"required"`
}

var UpdatedUserPut = User{
	ID: 2, Name: "Marina", Age: 23, Email: "marina@gmail.com"}

// UpdatedUserPatch — тестовые данные для PATCH-запроса
var UpdatedUserPatch = PartialUser{
	ID:    1,
	Email: StrPtr("new@gmail.com"),
}

// StrPtr - вспомогательная функция для удобного создания указателей
func StrPtr(s string) *string {
	return &s
}

var DeletedUser = User{ID: 1}

var GetUserByID = User{ID: 3}
