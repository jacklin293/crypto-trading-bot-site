package model

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ContractStrategy struct {
	Id                    int64
	Uuid                  string
	UserUuid              string
	Symbol                string // e.g. BTC-PERP
	Margin                decimal.Decimal
	Side                  int64 // 0: short  1: long
	Params                datatypes.JSONMap
	Enabled               int64  // 0: disabled  1: enabled
	PositionStatus        int64  // 0: closed  1: opened  2: unknown
	Exchange              string // e.g. FTX
	ExchangeOrdersDetails datatypes.JSONMap
	LastPositionAt        time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (db *DB) GetContractStrategiesByUser(userUuid string) ([]ContractStrategy, int64, error) {
	var css []ContractStrategy
	result := db.GormDB.Where("user_uuid = ?", userUuid).Order("enabled DESC, position_status DESC").Find(&css)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return css, 0, result.Error
	}
	return css, result.RowsAffected, result.Error
}
