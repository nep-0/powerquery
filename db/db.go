package db

import (
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
}

type BadgerCache struct {
	db *badger.DB
}

func NewBadgerCache(path string) (*BadgerCache, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}
	return &BadgerCache{db: db}, nil
}

func (c *BadgerCache) Get(key string) ([]byte, error) {
	var value []byte
	err := c.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	return value, err
}

func (c *BadgerCache) Set(key string, value []byte, ttl time.Duration) error {
	return c.db.Update(func(tx *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return tx.SetEntry(e)
	})
}

func (c *BadgerCache) Delete(key string) error {
	return c.db.Update(func(tx *badger.Txn) error {
		return tx.Delete([]byte(key))
	})
}
