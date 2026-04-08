package manager

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteShareStoreCRUD(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "shares.db")
	store, err := newSQLiteShareStore(dbPath)
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	sqliteStore := store.(*sqliteShareStore)
	defer func() {
		_ = sqliteStore.Close()
	}()

	now := time.Now().UTC().Round(time.Second)
	created := &Share{
		ID:          "id-1",
		Code:        "abc123",
		Kind:        shareKindClipboardImage,
		Name:        "clip image",
		Visible:     true,
		BinaryData:  []byte{1, 2, 3},
		MimeType:    "image/png",
		CreatedAt:   now,
		LastUpdated: now,
	}
	if err := store.Create(created); err != nil {
		t.Fatalf("create share: %v", err)
	}

	gotByCode, err := store.GetByCode("ABC123")
	if err != nil {
		t.Fatalf("get by code: %v", err)
	}
	if gotByCode.Name != created.Name {
		t.Fatalf("expected name %q, got %q", created.Name, gotByCode.Name)
	}
	if !bytes.Equal(gotByCode.BinaryData, created.BinaryData) {
		t.Fatalf("binary data mismatch")
	}

	gotByCode.Name = "renamed"
	gotByCode.LastUpdated = now.Add(time.Minute)
	if err := store.Update(gotByCode); err != nil {
		t.Fatalf("update share: %v", err)
	}

	gotByID, err := store.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if gotByID.Name != "renamed" {
		t.Fatalf("expected updated name, got %q", gotByID.Name)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 share, got %d", len(list))
	}

	deleted, err := store.DeleteByID(created.ID)
	if err != nil {
		t.Fatalf("delete share: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete true")
	}

	_, err = store.GetByID(created.ID)
	if !errors.Is(err, errShareNotFound) {
		t.Fatalf("expected errShareNotFound, got %v", err)
	}
}

func TestSQLiteShareStorePersistsAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "shares.db")
	store1, err := newSQLiteShareStore(dbPath)
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	sqlite1 := store1.(*sqliteShareStore)

	share := &Share{
		ID:          "persist-1",
		Code:        "persist",
		Kind:        shareKindClipboardText,
		Name:        "persist-name",
		TextContent: "hello",
		CreatedAt:   time.Now().UTC().Round(time.Second),
		LastUpdated: time.Now().UTC().Round(time.Second),
	}
	if err := store1.Create(share); err != nil {
		t.Fatalf("create share: %v", err)
	}
	_ = sqlite1.Close()

	store2, err := newSQLiteShareStore(dbPath)
	if err != nil {
		t.Fatalf("reopen sqlite store: %v", err)
	}
	sqlite2 := store2.(*sqliteShareStore)
	defer func() { _ = sqlite2.Close() }()

	got, err := store2.GetByCode("PERSIST")
	if err != nil {
		t.Fatalf("get persisted share: %v", err)
	}
	if got.ID != share.ID || got.TextContent != "hello" {
		t.Fatalf("unexpected persisted share: %+v", got)
	}
}
