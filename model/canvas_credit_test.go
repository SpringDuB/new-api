package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestCanvasCreditMigrationAddsWalletQuotaOnce(t *testing.T) {
	setupUserSessionTest(t)
	if err := DB.AutoMigrate(&CanvasCreditMigration{}); err != nil {
		t.Fatal(err)
	}
	oldRate, oldUnit := operation_setting.USDExchangeRate, common.QuotaPerUnit
	t.Cleanup(func() { operation_setting.USDExchangeRate, common.QuotaPerUnit = oldRate, oldUnit })
	operation_setting.USDExchangeRate, common.QuotaPerUnit = 7.3, 500000
	owner := User{Username: "migration-owner", AffCode: "origin", Status: common.UserStatusEnabled, Quota: 20}
	other := User{Username: "migration-other", AffCode: "other", Status: common.UserStatusEnabled}
	if err := DB.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	request := CanvasCreditMigration{ID: strings.Repeat("a", 64), UserID: owner.Id, CNYMicros: 7_300_000}
	if err := ApplyCanvasCreditMigration(request); err != nil {
		t.Fatal(err)
	}
	operation_setting.USDExchangeRate = 9
	if err := ApplyCanvasCreditMigration(request); err != nil {
		t.Fatal(err)
	}
	request.UserID = other.Id
	if err := ApplyCanvasCreditMigration(request); !errors.Is(err, ErrCanvasCreditMigrationConflict) {
		t.Fatalf("destination change was accepted: %v", err)
	}
	if err := DB.First(&owner, owner.Id).Error; err != nil {
		t.Fatal(err)
	}
	if owner.Quota != 500020 {
		t.Fatalf("expected exactly one 500000 quota credit; got %d", owner.Quota)
	}
	if err := DB.First(&other, other.Id).Error; err != nil {
		t.Fatal(err)
	}
	if other.Quota != 0 {
		t.Fatalf("credited wrong wallet: %d", other.Quota)
	}
}
