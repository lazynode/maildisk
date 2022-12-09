package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"maildisk"
	"maildisk/lazy"
	"os"
	"path/filepath"
)

func main() {
	flag.StringVar(&cmd, "cmd", "", "command")
	flag.StringVar(&conf, "conf", filepath.Join(lazy.Unwrap(os.UserHomeDir()), ".maildisk.json"), "path of config")
	flag.StringVar(&infile, "infile", "", "path of input file")
	flag.StringVar(&outfile, "outfile", "", "path of output file")
	flag.StringVar(&digest, "digest", "", "hex of digest")
	flag.Parse()
	var mail maildisk.Type
	lazy.Assert(json.NewDecoder(lazy.Unwrap(os.Open(conf))).Decode(&mail))
	mail.Init()
	defer mail.Close()
	switch cmd {
	case "put-value":
		fmt.Println(hex.EncodeToString(mail.PutValue(lazy.Unwrap(ioutil.ReadFile(infile)))))
	case "get-value":
		lazy.Assert(ioutil.WriteFile(outfile, mail.GetValue(lazy.Unwrap(hex.DecodeString(digest))), 0644))
	case "put-object":
		fmt.Println(hex.EncodeToString(mail.PutObject(lazy.Unwrap(ioutil.ReadFile(infile)))))
	case "get-object":
		lazy.Assert(ioutil.WriteFile(outfile, mail.GetObject(lazy.Unwrap(hex.DecodeString(digest))), 0644))
	default:
		panic("invalid command")
	}
}

var (
	cmd     string
	conf    string
	infile  string
	outfile string
	digest  string
)
