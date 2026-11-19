package model

import (
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCanvasCreditMigrationConflict = errors.New("canvas credit transfer does not match the original receipt")
var ErrCanvasCreditMigrationInvalid = errors.New("invalid canvas credit transfer")

// The receipt and wallet increment commit together, so delivery can be retried
// even when the previous HTTP response was lost.
type CanvasCreditMigration struct {
	ID        string    `json:"id" gorm:"primaryKey;size:64"`
	UserID    int       `json:"user_id" gorm:"index;not null"`
	CNYMicros int64     `json:"cny_micros" gorm:"not null"`
	Quota     int       `json:"-" gorm:"not null"`
	CreatedAt time.Time `json:"-"`
}

func ApplyCanvasCreditMigration(request CanvasCreditMigration) error {
	if len(request.ID) != 64 || request.UserID <= 0 || request.CNYMicros <= 0 {
		return ErrCanvasCreditMigrationInvalid
	}
	if _, err := hex.DecodeString(request.ID); err != nil {
		return ErrCanvasCreditMigrationInvalid
	}
	receipt := CanvasCreditMigration{ID: request.ID, UserID: request.UserID, CNYMicros: request.CNYMicros}
	credited := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).Create(&receipt)
		if result.Error != nil {
			return result.Error
		}
		var stored CanvasCreditMigration
		if err := lockForUpdate(tx).First(&stored, "id = ?", request.ID).Error; err != nil {
			return err
		}
		if stored.UserID != request.UserID || stored.CNYMicros != request.CNYMicros {
			return ErrCanvasCreditMigrationConflict
		}
		if stored.Quota > 0 {
			return nil
		}
		var user User
		if err := lockForUpdate(tx).Select("id", "status").First(&user, request.UserID).Error; err != nil {
			return err
		}
		if user.Status != common.UserStatusEnabled {
			return ErrUserSessionInactive
		}
		rate, unit := operation_setting.USDExchangeRate, common.QuotaPerUnit
		if rate <= 0 || unit <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || math.IsNaN(unit) || math.IsInf(unit, 0) {
			return ErrCanvasCreditMigrationInvalid
		}
		quota, remainder := decimal.New(request.CNYMicros, -6).Mul(decimal.NewFromFloat(unit)).QuoRem(decimal.NewFromFloat(rate), 0)
		if remainder.IsPositive() {
			quota = quota.Add(decimal.NewFromInt(1))
		}
		creditedQuota, err := common.WalletQuotaFromDecimalStrict(quota)
		if err != nil {
			return ErrTopUpQuotaLimitExceeded
		}
		receipt.Quota = creditedQuota
		if err := creditTopUpQuota(tx, user.Id, receipt.Quota, nil); err != nil {
			return err
		}
		if err := tx.Model(&receipt).UpdateColumn("quota", receipt.Quota).Error; err != nil {
			return err
		}
		credited = true
		return nil
	})
	if err == nil && credited {
		syncCreditUserQuotaCache(receipt.UserID, receipt.Quota, common.LogText("canvas credit migration"))
	}
	return err
}
