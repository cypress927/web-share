package manager

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type sqliteShareStore struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

type shareRecord struct {
	ID          string `gorm:"primaryKey;size:64"`
	Code        string `gorm:"index;size:32"`
	Kind        string `gorm:"size:32"`
	Path        string `gorm:"type:text"`
	Name        string `gorm:"type:text"`
	IsDir       bool
	Visible     bool
	Password    string    `gorm:"type:text"`
	TextContent string    `gorm:"type:text"`
	BinaryData  []byte    `gorm:"type:blob"`
	MimeType    string    `gorm:"size:128"`
	CreatedAt   time.Time `gorm:"index"`
	LastUpdated time.Time `gorm:"index"`
}

func newSQLiteShareStore(path string) (ShareStore, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&shareRecord{}); err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	return &sqliteShareStore{db: db, sqlDB: sqlDB}, nil
}

func (s *sqliteShareStore) Create(share *Share) error {
	if share == nil {
		return errors.New("share is nil")
	}
	return s.db.Create(toShareRecord(share)).Error
}

func (s *sqliteShareStore) Update(share *Share) error {
	if share == nil {
		return errors.New("share is nil")
	}
	record := toShareRecord(share)
	tx := s.db.Model(&shareRecord{}).Where("id = ?", share.ID).Updates(record)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return errShareNotFound
	}
	return nil
}

func (s *sqliteShareStore) DeleteByID(id string) (bool, error) {
	tx := s.db.Delete(&shareRecord{}, "id = ?", id)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (s *sqliteShareStore) GetByID(id string) (*Share, error) {
	var record shareRecord
	if err := s.db.Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errShareNotFound
		}
		return nil, err
	}
	return fromShareRecord(&record), nil
}

func (s *sqliteShareStore) GetByCode(code string) (*Share, error) {
	var record shareRecord
	if err := s.db.Where("lower(code) = ?", strings.ToLower(code)).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errShareNotFound
		}
		return nil, err
	}
	return fromShareRecord(&record), nil
}

func (s *sqliteShareStore) List() ([]*Share, error) {
	var records []shareRecord
	if err := s.db.Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]*Share, 0, len(records))
	for i := range records {
		result = append(result, fromShareRecord(&records[i]))
	}
	return result, nil
}

func (s *sqliteShareStore) Close() error {
	if s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func toShareRecord(share *Share) *shareRecord {
	record := &shareRecord{
		ID:          share.ID,
		Code:        share.Code,
		Kind:        share.Kind,
		Path:        share.Path,
		Name:        share.Name,
		IsDir:       share.IsDir,
		Visible:     share.Visible,
		Password:    share.Password,
		TextContent: share.TextContent,
		MimeType:    share.MimeType,
		CreatedAt:   share.CreatedAt,
		LastUpdated: share.LastUpdated,
	}
	if share.BinaryData != nil {
		record.BinaryData = append([]byte(nil), share.BinaryData...)
	}
	return record
}

func fromShareRecord(record *shareRecord) *Share {
	share := &Share{
		ID:          record.ID,
		Code:        record.Code,
		Kind:        record.Kind,
		Path:        record.Path,
		Name:        record.Name,
		IsDir:       record.IsDir,
		Visible:     record.Visible,
		Password:    record.Password,
		TextContent: record.TextContent,
		MimeType:    record.MimeType,
		CreatedAt:   record.CreatedAt,
		LastUpdated: record.LastUpdated,
	}
	if record.BinaryData != nil {
		share.BinaryData = append([]byte(nil), record.BinaryData...)
	}
	return share
}
