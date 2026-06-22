package main

import (
	"testing"
	"time"

	"temu-tools/backend/internal/models"
)

func TestDeliveryExtractExportFilenameUsesSameDayBeforeFourPM(t *testing.T) {
	batch := models.DeliveryExtractBatch{
		Rows: []models.DeliveryExtractRow{{EuRepresentative: "刘典松"}},
	}
	now := time.Date(2026, time.June, 21, 15, 59, 0, 0, time.Local)

	got := deliveryExtractExportFilename(batch, now)
	want := "刘典松6-21.xlsx"
	if got != want {
		t.Fatalf("deliveryExtractExportFilename() = %q, want %q", got, want)
	}
}

func TestDeliveryExtractExportFilenameUsesNextDayAtFourPM(t *testing.T) {
	batch := models.DeliveryExtractBatch{
		Rows: []models.DeliveryExtractRow{{EuRepresentative: "刘典松"}},
	}
	now := time.Date(2026, time.June, 21, 16, 0, 0, 0, time.Local)

	got := deliveryExtractExportFilename(batch, now)
	want := "刘典松6-22.xlsx"
	if got != want {
		t.Fatalf("deliveryExtractExportFilename() = %q, want %q", got, want)
	}
}

func TestDeliveryExtractExportFilenameFallsBackAndSanitizesName(t *testing.T) {
	batch := models.DeliveryExtractBatch{
		Rows: []models.DeliveryExtractRow{{ShopName: `店铺/A:01`}},
	}
	now := time.Date(2026, time.June, 21, 8, 0, 0, 0, time.Local)

	got := deliveryExtractExportFilename(batch, now)
	want := "店铺-A-016-21.xlsx"
	if got != want {
		t.Fatalf("deliveryExtractExportFilename() = %q, want %q", got, want)
	}
}
