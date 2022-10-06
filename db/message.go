package db

import (
	"github.com/COAOX/zecrey_warrior/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type message db

func (m *message) Create(message *model.Message) error {
	return m.DB.Create(message).Error
}

func (m *message) ListLatest(offset, size int) ([]model.Message, error) {
	var messages []model.Message
	err := m.DB.Preload(clause.Associations).Model(&model.Message{}).Order("CreatedAt DESC").Offset(offset).Limit(size).Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}
