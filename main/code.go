package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"maildisk"
	"maildisk/lazy"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v3"
)

func main() {
	conf := lazy.Default(filepath.Join(lazy.Unwrap(os.UserHomeDir()), `.maildisk`))(os.LookupEnv(`CONF`))
	file := lazy.Unwrap(os.Open(filepath.Join(conf, `config.json`)))
	defer file.Close()
	mail := lazy.JsonDecode[maildisk.Type](file)
	mail.Init()
	defer mail.Close()
	opt := badger.DefaultOptions(filepath.Join(conf, `db`))
	opt.Logger = nil
	db := lazy.Unwrap(badger.Open(opt))
	defer db.Close()
	switch lazy.Default(`GET`)(os.LookupEnv(`CMD`)) {
	case `PUT`, `put`:
		lf := lazy.Default(``)(os.LookupEnv(`LF`))
		rf := lazy.Default(``)(os.LookupEnv(`RF`))
		lazy.Require(len(lf)*len(rf) > 0, `LF & RF required`)
		lazy.Require(filepath.IsAbs(rf), `RF should be abs path`)
		hash := mail.Put(rf, lazy.Unwrap(os.ReadFile(lf)))
		lazy.Assert(db.Update(func(txn *badger.Txn) error { return txn.Set([]byte(rf), hash) }))
		log.Println(`PUT`, lf, `->`, rf, hex.EncodeToString(hash))
	case `GET`, `get`:
		lf := lazy.Default(``)(os.LookupEnv(`LF`))
		rf := lazy.Default(``)(os.LookupEnv(`RF`))
		lazy.Require(len(lf)*len(rf) > 0, `LF & RF required`)
		lazy.Require(filepath.IsAbs(rf), `RF should be abs path`)
		lazy.Assert(db.View(func(txn *badger.Txn) error {
			return lazy.Unwrap(txn.Get([]byte(rf))).Value(func(val []byte) error {
				return os.WriteFile(lf, mail.Get(val), 0644)
			})
		}))
	case `LIST`, `list`:
		lazy.Assert(db.View(func(txn *badger.Txn) error {
			opt := badger.DefaultIteratorOptions
			opt.Prefix = []byte(lazy.Default(`/`)(os.LookupEnv(`PREFIX`)))
			iter := txn.NewIterator(opt)
			defer iter.Close()
			for iter.Rewind(); iter.Valid(); iter.Next() {
				item := iter.Item()
				lazy.Assert(item.Value(func(val []byte) error {
					fmt.Println(hex.EncodeToString(val), "\t", string(item.Key()))
					return nil
				}))
			}
			return nil
		}))
	default:
		panic(`invalid cmd`)
	}
}
