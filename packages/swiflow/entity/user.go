package entity

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type UserEntity struct {
	ID uint64 `json:"id" gorm:"primarykey"`

	Avatar   string `json:"avatar" gorm:"column:avatar;type:varchar(256)"`
	Email    string `json:"email" gorm:"type:varchar(256);index;not null"`
	Username string `json:"username" gorm:"type:varchar(128);index;not null"`
	Password string `json:"password" gorm:"type:varchar(64);not null"`
	APIKey   string `json:"apiKey" gorm:"column:api_key;type:varchar(68);index"`
	IsActive bool   `json:"isActive" gorm:"column:is_active;default:true"`
	Verified bool   `json:"verified" gorm:"column:verified;default:false"`
	UserRole string `json:"userRole" gorm:"column:user_role;default:user"`

	// 当前套餐、过期时间
	ExpireAt *time.Time  `json:"expireAt" gorm:"column:expire_at;default:null"`
	UserPlan string      `json:"userPlan" gorm:"column:user_plan;type:varchar(20)"`
	ApiUsage interface{} `json:"apiUsage" gorm:"column:api_usage;serializer:json"`
	ApiLimit interface{} `json:"apiLimit" gorm:"column:api_limit;serializer:json"`

	gorm.Model
}

func (m *UserEntity) TableName() string {
	return "llm_user"
}

func (r *UserEntity) ToMap() map[string]any {
	return map[string]any{
		"email":  r.Email,
		"avatar": r.Avatar,
		"apiKey": r.APIKey,

		"username": r.Username,
		"userPlan": r.UserPlan,
		"expireAt": r.ExpireAt,
	}
}

// FromMap 从 map 转换为 UserEntity
func (r *UserEntity) FromMap(data map[string]any) error {
	if val, ok := data["email"].(string); ok {
		r.Email = val
	}
	if val, ok := data["avatar"].(string); ok {
		r.Avatar = val
	}
	if val, ok := data["apiKey"].(string); ok {
		r.APIKey = val
	}
	if val, ok := data["username"].(string); ok {
		r.Username = val
	}
	if val, ok := data["userRole"].(string); ok {
		r.UserRole = val
	}
	if val, ok := data["userPlan"].(string); ok {
		r.UserPlan = val
	}
	if val, ok := data["isActive"].(bool); ok {
		r.IsActive = val
	}
	if val, ok := data["verified"].(bool); ok {
		r.Verified = val
	}
	if val, ok := data["createdAt"].(string); ok {
		r.CreatedAt, _ = time.Parse(time.RFC3339, val)
	}
	if val, ok := data["updatedAt"].(string); ok {
		r.UpdatedAt, _ = time.Parse(time.RFC3339, val)
	}
	if val, ok := data["apiUsage"].(map[string]any); ok {
		r.ApiUsage = val
	}
	if val, ok := data["apiLimit"].(map[string]any); ok {
		r.ApiLimit = val
	}
	if val, ok := data["expireAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			r.ExpireAt = &t
		}
	}
	if r.Email == "" || r.APIKey == "" {
		return fmt.Errorf("lose email or apiKey")
	}
	return nil
}
