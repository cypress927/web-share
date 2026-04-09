package manager

import (
	"errors"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type SettingsStore interface {
	GetDefaultLanguage() (string, error)
	SetDefaultLanguage(lang string) error
	GetSetupCompleted() (bool, error)
	SetSetupCompleted(done bool) error
}

type memorySettingsStore struct {
	mu          sync.RWMutex
	defaultLang string
}

func newMemorySettingsStore(defaultLang string) *memorySettingsStore {
	if !isSupportedLanguage(defaultLang) {
		defaultLang = langEN
	}
	return &memorySettingsStore{defaultLang: defaultLang}
}

func (s *memorySettingsStore) GetDefaultLanguage() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultLang, nil
}

func (s *memorySettingsStore) SetDefaultLanguage(lang string) error {
	if !isSupportedLanguage(lang) {
		return errors.New("unsupported language")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultLang = lang
	return nil
}

func (s *memorySettingsStore) GetSetupCompleted() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return false, nil
}

func (s *memorySettingsStore) SetSetupCompleted(bool) error {
	return nil
}

type sqliteSettingsStore struct {
	db *gorm.DB
}

type appSetting struct {
	Key   string `gorm:"primaryKey;size:64"`
	Value string `gorm:"type:text"`
}

func newSQLiteSettingsStore(path string) (SettingsStore, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&appSetting{}); err != nil {
		return nil, err
	}
	store := &sqliteSettingsStore{db: db}
	if _, err := store.get("setup_completed", "false"); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *sqliteSettingsStore) GetDefaultLanguage() (string, error) {
	value, err := s.get("default_lang", "")
	if err != nil {
		return "", err
	}
	lang := normalizeLanguage(value)
	if !isSupportedLanguage(lang) {
		return "", nil
	}
	return lang, nil
}

func (s *sqliteSettingsStore) SetDefaultLanguage(lang string) error {
	if !isSupportedLanguage(lang) {
		return errors.New("unsupported language")
	}
	return s.set("default_lang", lang)
}

func (s *sqliteSettingsStore) GetSetupCompleted() (bool, error) {
	value, err := s.get("setup_completed", "false")
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

func (s *sqliteSettingsStore) SetSetupCompleted(done bool) error {
	if done {
		return s.set("setup_completed", "true")
	}
	return s.set("setup_completed", "false")
}

func (s *sqliteSettingsStore) get(key, fallback string) (string, error) {
	var record appSetting
	err := s.db.Where("key = ?", key).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := s.db.Create(&appSetting{Key: key, Value: fallback}).Error; err != nil {
			return "", err
		}
		return fallback, nil
	}
	if err != nil {
		return "", err
	}
	if record.Value == "" {
		return fallback, nil
	}
	return record.Value, nil
}

func (s *sqliteSettingsStore) set(key, value string) error {
	return s.db.Save(&appSetting{Key: key, Value: value}).Error
}
