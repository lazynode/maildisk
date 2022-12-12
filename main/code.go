package main

import (
	"encoding/hex"
	"fmt"
	"maildisk"
	"maildisk/lazy"
	"os"
	"path/filepath"
)

func main() {
	mail := lazy.JsonDecode[maildisk.Type](lazy.Unwrap(os.Open(lazy.Default(filepath.Join(lazy.Unwrap(os.UserHomeDir()), `.maildisk`, `config.json`))(os.LookupEnv(`CONF`)))))
	mail.Init()
	defer mail.Close()
	file := lazy.Default(``)(os.LookupEnv(`LF`))
	target := lazy.Default(file)(os.LookupEnv(`RF`))

	switch lazy.Default(`GET`)(os.LookupEnv(`CMD`)) {
	case `PUT`, `put`:
		fmt.Println(hex.EncodeToString(mail.Put(target, lazy.Unwrap(os.ReadFile(file)))))

	case `GET`, `get`:
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
