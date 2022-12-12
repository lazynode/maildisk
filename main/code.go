package main

import (
	"encoding/hex"
	"log"
	"maildisk"
	"maildisk/lazy"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v3"
)

func main() {
	conf := lazy.Default(filepath.Join(lazy.Unwrap(os.UserHomeDir()), `.maildisk`))(os.LookupEnv(`CONF`))
	mail := lazy.JsonDecode[maildisk.Type](lazy.Unwrap(os.Open(filepath.Join(conf, `config.json`))))
	mail.Init()
	defer mail.Close()
	lf := lazy.Default(`/tmp/file`)(os.LookupEnv(`LF`))
	rf := lazy.Default(filepath.Join(lazy.Default(`/`)(os.LookupEnv(`PREFIX`)), lf))(os.LookupEnv(`RF`))
	lazy.Require(filepath.IsAbs(rf), `RF should be abs path`)
	opt := badger.DefaultOptions(filepath.Join(conf, `db`))
	opt.Logger = nil
	db := lazy.Unwrap(badger.Open(opt))
	defer db.Close()

	switch lazy.Default(`GET`)(os.LookupEnv(`CMD`)) {
	case `PUT`, `put`:
		hash := mail.Put(rf, lazy.Unwrap(os.ReadFile(lf)))
		lazy.Assert(db.Update(func(txn *badger.Txn) error { return txn.Set([]byte(rf), hash) }))
		log.Println(`PUT`, hex.EncodeToString(hash))
	case `GET`, `get`:
		lazy.Assert(db.View(func(txn *badger.Txn) error {
			return lazy.Unwrap(txn.Get([]byte(rf))).Value(func(val []byte) error {
				return os.WriteFile(lf, mail.Get(val), 0644)
			})
		}))
	default:
		panic(`invalid cmd`)
		// case "put-value":
		// 	fmt.Println(hex.EncodeToString(mail.PutValue(lazy.Unwrap(os.ReadFile(lazy.Default("/dev/stdin")(os.LookupEnv(`IF`)))))))
		// case "get-value":
		// 	lazy.Assert(os.WriteFile(lazy.Default("/dev/stdout")(os.LookupEnv(`OF`)), mail.GetValue(lazy.Unwrap(hex.DecodeString(lazy.Default("")(os.LookupEnv(`HASH`))))), 0644))
		// case "put-object":
		// 	fmt.Println(hex.EncodeToString(mail.PutObject(lazy.Unwrap(os.ReadFile(lazy.Default("/dev/stdin")(os.LookupEnv(`IF`)))))))
		// case "get-object":
		// 	lazy.Assert(os.WriteFile(lazy.Default("/dev/stdout")(os.LookupEnv(`OF`)), mail.GetObject(lazy.Unwrap(hex.DecodeString(lazy.Default("")(os.LookupEnv(`HASH`))))), 0644))
		// default:
		// 	panic("invalid command")
	}
}
