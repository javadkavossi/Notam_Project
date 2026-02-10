package model

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	Id        int            `gorm:"primaryKey"`
	CreatedAt time.Time      `gorm:"type:TIMESTAMP with time zone;not null"`
	UpdatedAt time.Time      `gorm:"type:TIMESTAMP with time zone;not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	return nil
}

func (m *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now().UTC()
	return nil
}
