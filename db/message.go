package db

import (
	"fmt"

	"github.com/COAOX/zecrey_warrior/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type message db

func (m *message) Create(message *model.Message) error {
	return m.db.Create(message).Error
}

func (m *message) ListLatest(offset, size int) ([]model.Message, error) {
	var messages []model.Message
	err := m.db.Debug().Preload(clause.Associations).Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true}).Offset(offset).Limit(size).Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}

	fmt.Println("list message", messages)
	return messages, err
}
