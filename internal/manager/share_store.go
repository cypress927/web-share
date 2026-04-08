package manager

import (
	"errors"
	"strings"
	"sync"
)

var errShareNotFound = errors.New("share not found")

type ShareStore interface {
	Create(share *Share) error
	Update(share *Share) error
	DeleteByID(id string) (bool, error)
	GetByID(id string) (*Share, error)
	GetByCode(code string) (*Share, error)
	List() ([]*Share, error)
}

type memoryShareStore struct {
	mu     sync.RWMutex
	shares map[string]*Share
}

func newMemoryShareStore() *memoryShareStore {
	return &memoryShareStore{shares: make(map[string]*Share)}
}

func newMemoryShareStoreFromMap(seed map[string]*Share) *memoryShareStore {
	store := newMemoryShareStore()
	for id, share := range seed {
		store.shares[id] = copyShare(share)
	}
	return store
}

func (s *memoryShareStore) Create(share *Share) error {
	if share == nil {
		return errors.New("share is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shares[share.ID] = copyShare(share)
	return nil
}

func (s *memoryShareStore) Update(share *Share) error {
	if share == nil {
		return errors.New("share is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shares[share.ID]; !ok {
		return errShareNotFound
	}
	s.shares[share.ID] = copyShare(share)
	return nil
}

func (s *memoryShareStore) DeleteByID(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shares[id]; !ok {
		return false, nil
	}
	delete(s.shares, id)
	return true, nil
}

func (s *memoryShareStore) GetByID(id string) (*Share, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	share, ok := s.shares[id]
	if !ok {
		return nil, errShareNotFound
	}
	return copyShare(share), nil
}

func (s *memoryShareStore) GetByCode(code string) (*Share, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, share := range s.shares {
		if strings.EqualFold(share.Code, code) {
			return copyShare(share), nil
		}
	}
	return nil, errShareNotFound
}

func (s *memoryShareStore) List() ([]*Share, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Share, 0, len(s.shares))
	for _, share := range s.shares {
		result = append(result, copyShare(share))
	}
	return result, nil
}
