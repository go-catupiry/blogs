package blog

import "time"

type BlogEditorsModel struct {
	BlogID    int64     `gorm:"primaryKey;index:PRIMARY,unique;column:blog_id;type:int(11);not null" json:"blogId"`
	UserID    int64     `gorm:"primaryKey;index:PRIMARY,unique;column:user_id;type:int(11);not null" json:"userId"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null" json:"updatedAt"`
}

func (m *BlogEditorsModel) TableName() string {
	return "blogs-editors"
}
