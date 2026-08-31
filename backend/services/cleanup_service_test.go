package services

import (
	"testing"

	"github.com/ctwj/urldb/db/entity"
)

func cleanupUint(value uint) *uint { return &value }

func TestGroupCleanupRecordsDeduplicatesByAccountAndFile(t *testing.T) {
	records := []entity.TransferRecord{
		{ID: 1, AccountID: cleanupUint(8), FileID: "same-file", ResourceID: cleanupUint(10)},
		{ID: 2, AccountID: cleanupUint(8), FileID: "same-file", ResourceID: cleanupUint(10)},
		{ID: 3, AccountID: cleanupUint(9), FileID: "same-file", ResourceID: cleanupUint(11)},
		{ID: 4},
		{ID: 5},
	}

	groups := groupCleanupRecords(records)
	if len(groups) != 4 {
		t.Fatalf("group count = %d, want 4", len(groups))
	}
	if len(groups[0]) != 2 || groups[0][0].ID != 1 || groups[0][1].ID != 2 {
		t.Fatalf("first group should contain both operations for the same file: %+v", groups[0])
	}
}

func TestCleanupResourceIDsAreUnique(t *testing.T) {
	records := []entity.TransferRecord{
		{ResourceID: cleanupUint(3)},
		{ResourceID: cleanupUint(3)},
		{ResourceID: cleanupUint(5)},
		{},
	}
	ids := cleanupResourceIDs(records)
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 5 {
		t.Fatalf("resource ids = %v, want [3 5]", ids)
	}
}
