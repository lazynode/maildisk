package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"maildisk"
	"maildisk/lazy"
	"maildisk/type/conf"
	"maildisk/type/exception/content_not_found"
	"maildisk/type/exception/init_failed"
	"maildisk/type/exception/login_failed"
	"maildisk/type/exception/mail_box_already_exists"
	"maildisk/type/exception/maxconn_is_zero"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger/v3"
)

func main() {
	defer lazy.Catch(func(err *mail_box_already_exists.Type) { log.Println(`MAILBOX ALREADY EXISTS; YOU DON'T NEED TO INIT`) })
	defer lazy.Catch(func(err *login_failed.Type) { log.Println(`LOGIN FAILED; CHECK YOUR CONFIG FILE`) })
	defer lazy.Catch(func(err *init_failed.Type) { log.Println(`INIT FAILED`) })
	defer lazy.Catch(func(err *maxconn_is_zero.Type) { log.Println(`MAXCONN IS 0; CHECK YOUR CONFIG FILE`) })
	defer lazy.Catch(func(err *content_not_found.Type) { log.Println(`CONTENT NOT FOUND:`, err.Hash) })

	dir := lazy.Default(filepath.Join(lazy.Unwrap(os.UserHomeDir()), `.maildisk`))(os.LookupEnv(`CONF`))
	file := lazy.Unwrap(os.Open(filepath.Join(dir, `config.json`)))
	defer file.Close()
	conf := lazy.JsonDecodePtr[conf.Type](file)
	opt := badger.DefaultOptions(filepath.Join(dir, `db`))
	opt.Logger = nil
	db := lazy.Unwrap(badger.Open(opt))
	defer db.Close()
	switch strings.ToUpper(lazy.Default(`GET`)(os.LookupEnv(`CMD`))) {
	case `PUT`:
		lf := lazy.Default(``)(os.LookupEnv(`LF`))
		rf := lazy.Default(``)(os.LookupEnv(`RF`))
		lazy.Require(len(lf)*len(rf) > 0, `LF & RF required`)
		lazy.Require(path.IsAbs(rf), `RF should be abs path`)
		hash := maildisk.Put(conf, rf, lazy.Unwrap(os.ReadFile(lf)))
		log.Println(`PUT`, lf, `->`, rf, hex.EncodeToString(hash))
		lazy.Assert(db.Update(func(txn *badger.Txn) error { return txn.Set([]byte(rf), hash) }))
	case `GET`:
		lf := lazy.Default(``)(os.LookupEnv(`LF`))
		rf := lazy.Default(``)(os.LookupEnv(`RF`))
		lazy.Require(len(lf)*len(rf) > 0, `LF & RF required`)
		lazy.Require(path.IsAbs(rf), `RF should be abs path`)
		lazy.Assert(db.View(func(txn *badger.Txn) error {
			return lazy.Unwrap(txn.Get([]byte(rf))).Value(func(hash []byte) error {
				return os.WriteFile(lf, maildisk.Get(conf, hash), 0644)
			})
		}))
	case `LIST`:
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
	case `INIT`:
		maildisk.Init(conf)
		log.Println(`DONE`)
	default:
		panic(`invalid cmd`)
	}
}
