package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type Task struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null"`
	Status      bool      `json:"status" gorm:"default:false"`
	Priority    int       `json:"priority" gorm:"default:1"`
	Deadline    string    `json:"deadline"`
	Information string    `json:"information"`
	UserID      int       `json:"user_id" gorm:"not null;index"`
	GroupID     int       `json:"group_id" gorm:"not null;index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Group struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null;uniqueIndex"`
	AdminID   int       `json:"admin_id" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type IntSlice []int

func (is IntSlice) Value() (driver.Value, error) {
	if is == nil {
		return json.Marshal([]int{})
	}
	return json.Marshal(is)
}

func (is *IntSlice) Scan(value interface{}) error {
	if value == nil {
		*is = []int{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan into IntSlice")
	}

	var result []int
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	if result == nil {
		*is = []int{}
	} else {
		*is = result
	}
	return nil
}

func (is IntSlice) MarshalJSON() ([]byte, error) {
	if is == nil {
		return json.Marshal([]int{})
	}
	return json.Marshal([]int(is))
}

type WorkTimes map[string]float64

func (wt WorkTimes) Value() (driver.Value, error) {
	if wt == nil {
		return json.Marshal(map[string]float64{})
	}
	return json.Marshal(wt)
}

func (wt *WorkTimes) Scan(value interface{}) error {
	if value == nil {
		*wt = make(map[string]float64)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan into WorkTimes")
	}

	result := make(map[string]float64)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*wt = result
	return nil
}

func (wt WorkTimes) MarshalJSON() ([]byte, error) {
	if wt == nil {
		return json.Marshal(map[string]float64{})
	}
	return json.Marshal(map[string]float64(wt))
}

type User struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	FullName  string    `json:"full_name" gorm:"not null"`
	Role      string    `json:"role" gorm:"not null;default:'user'"`
	GroupIDs  IntSlice  `json:"group_ids" gorm:"type:json"`
	Number    string    `json:"number"`
	Email     string    `json:"email" gorm:"not null;uniqueIndex"`
	Password  string    `json:"password,omitempty" gorm:"not null"`
	WorkTimes WorkTimes `json:"work_times" gorm:"type:json"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type UserGroup struct {
	UserID  int `json:"user_id" gorm:"primaryKey"`
	GroupID int `json:"group_id" gorm:"primaryKey"`
}

type SearchTask struct {
	UserID int  `json:"user_id"`
	Task   Task `json:"task"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Count   int         `json:"count"`
	Page    int         `json:"page,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Total   int64       `json:"total,omitempty"`
}

type CreateUserRequest struct {
	FullName  string             `json:"full_name" binding:"required"`
	Role      string             `json:"role"`
	GroupIDs  []int              `json:"group_ids"`
	Number    string             `json:"number"`
	Email     string             `json:"email" binding:"required,email"`
	Password  string             `json:"password" binding:"required,min=6"`
	WorkTimes map[string]float64 `json:"work_times"`
}

type UpdateUserRequest struct {
	FullName  string             `json:"full_name,omitempty"`
	Role      string             `json:"role,omitempty"`
	GroupIDs  []int              `json:"group_ids,omitempty"`
	Number    string             `json:"number,omitempty"`
	Email     string             `json:"email,omitempty"`
	Password  string             `json:"password,omitempty"`
	WorkTimes map[string]float64 `json:"work_times,omitempty"`
}

type CreateTaskRequest struct {
	Title       string `json:"title" binding:"required"`
	Priority    int    `json:"priority"`
	Deadline    string `json:"deadline"`
	Information string `json:"information"`
	GroupID     int    `json:"group_id" binding:"required"`
}

type UpdateTaskRequest struct {
	Title       string `json:"title,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Deadline    string `json:"deadline,omitempty"`
	Information string `json:"information,omitempty"`
	Status      *bool  `json:"status,omitempty"`
	GroupID     int    `json:"group_id,omitempty"`
}

type CreateGroupRequest struct {
	Name    string `json:"name" binding:"required"`
	AdminID int    `json:"admin_id" binding:"required"`
}

type UpdateGroupRequest struct {
	Name    string `json:"name,omitempty"`
	AdminID int    `json:"admin_id,omitempty"`
}

type WorkTimesRequest struct {
	WorkTimes map[string]float64 `json:"work_times" binding:"required"`
}
