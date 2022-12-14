package maildisk

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"maildisk/lazy"
	"maildisk/type/conf"
	"maildisk/type/exception/content_not_found"
	"maildisk/type/exception/init_failed"
	"maildisk/type/exception/login_failed"
	"maildisk/type/exception/mail_box_already_exists"
	"maildisk/type/exception/maxconn_is_zero"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func Put(config *conf.Type, path string, data []byte) []byte {
	pool := createpool(config)
	hash := puts(pool, config, []byte(TAGDATA), data)
	puts(pool, config, []byte(TAGATTR), bytes.Join([][]byte{hash, []byte(path)}, nil))
	return hash
}

func Get(config *conf.Type, hash []byte) []byte {
	return gets(createpool(config), config, []byte(TAGDATA), hash)
}

func Init(config *conf.Type) {
	defer lazy.Catch(init_failed.CatchError)
	mail := lazy.Unwrap(client.DialTLS(config.Address, nil))
	defer lazy.Catch(login_failed.CatchError)
	lazy.Assert(mail.Login(config.Username, config.Password))
	defer lazy.Catch(mail_box_already_exists.CatchError)
	lazy.Assert(mail.Create(MAILBOX))
}

func createpool(config *conf.Type) chan *client.Client {
	lazy.Require(config.MaxConn > 0, &maxconn_is_zero.Type{})
	pool := make(chan *client.Client, config.MaxConn)
	for i := 0; i < config.MaxConn; i++ {
		pool <- nil
	}
	return pool
}

func pickmail(pool chan *client.Client, config *conf.Type) *client.Client {
	mail := <-pool
	if mail == nil {
		mail = lazy.Unwrap(client.DialTLS(config.Address, nil))
		lazy.Assert(mail.Login(config.Username, config.Password))
		lazy.Unwrap(mail.Select(MAILBOX, false))
	}
	return mail
}

func gets(pool chan *client.Client, config *conf.Type, tag []byte, hash []byte) []byte {
	data := get(pool, config, tag, hash)
	if len(data) < HARDLIMIT {
		return data
	}
	l, r, this := data[0:32], data[32:64], data[64:]
	return bytes.Join(append([][]byte{this}, lazy.ParallelReturn(func() []byte { return gets(pool, config, tag, l) }, func() []byte { return gets(pool, config, tag, r) })...), nil)
}

func get(pool chan *client.Client, config *conf.Type, tag []byte, hash []byte) []byte {
	lazy.Require(len(hash) == 32, `invalid hash`)
	mail := pickmail(pool, config)
	defer func() { pool <- mail }()
	subject, to := hex.EncodeToString(hash), hex.EncodeToString(tag)
	sc := imap.NewSearchCriteria()
	sc.Header.Add("Subject", subject)
	sc.Header.Add("To", to)
	uids := lazy.Unwrap(mail.UidSearch(sc))
	for _, uid := range uids {
		ch := make(chan *imap.Message, 1)
		ss := new(imap.SeqSet)
		ss.AddNum(uid)
		lazy.Assert(mail.UidFetch(ss, []imap.FetchItem{imap.FetchRFC822Text}, ch))
		for msg := range ch {
			for _, text := range msg.Body {
				data := lazy.Unwrap(io.ReadAll(base64.NewDecoder(base64.StdEncoding, text)))
				dig := sha256.Sum256(data)
				if bytes.Equal(dig[:], hash) {
					return data
				}
			}
		}
	}
	panic(&content_not_found.Type{Hash: hash})
}

func put(pool chan *client.Client, config *conf.Type, tag []byte, data []byte) []byte {
	lazy.Require(len(data) <= HARDLIMIT, `size error`)
	mail := pickmail(pool, config)
	defer func() { pool <- mail }()
	hash := sha256.Sum256(data)
	subject, to := hex.EncodeToString(hash[:]), hex.EncodeToString(tag)
	sc := imap.NewSearchCriteria()
	sc.Header.Add("Subject", subject)
	sc.Header.Add("To", to)
	if len(lazy.Unwrap(mail.Search(sc))) == 0 {
		lazy.Assert(mail.Append(MAILBOX, nil, time.Now(), strings.NewReader(fmt.Sprintf("Subject: %s\r\nTo: %s\r\n\r\n%s", subject, to, regexp.MustCompile(`(.{1,78})`).ReplaceAllString(base64.StdEncoding.EncodeToString(data), "$1\r\n")))))
	}
	return hash[:]
}

func puts(pool chan *client.Client, config *conf.Type, tag []byte, data []byte) []byte {
	if len(data) < HARDLIMIT {
		return put(pool, config, tag, data)
	}
	sx, sy := SOFTLIMIT, (len(data)/SOFTLIMIT+1)/2*SOFTLIMIT
	this, l, r := data[:sx], data[sx:sy], data[sy:]
	return put(pool, config, tag, bytes.Join(append(lazy.ParallelReturn(func() []byte { return puts(pool, config, tag, l) }, func() []byte { return puts(pool, config, tag, r) }), this), nil))
}

const (
	MAILBOX   = `MDDATA`
	HARDLIMIT = 64 * 1024
	SOFTLIMIT = HARDLIMIT - 64

	TAGATTR = `ATTR`
	TAGDATA = `DATA`
)
