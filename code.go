package maildisk

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"maildisk/lazy"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

const sizelimit = 64 * 1024

type Type struct {
	Address  string
	Username string
	Password string
	mail     *client.Client
}

func (me *Type) Init() {
	me.mail = lazy.Unwrap(client.DialTLS(me.Address, nil))
	lazy.Assert(me.mail.Login(me.Username, me.Password))
	defer lazy.Catch(func(error) { lazy.Assert(me.mail.Create(mailbox)); lazy.Unwrap(me.mail.Select(mailbox, false)) })
	lazy.Unwrap(me.mail.Select(mailbox, false))
}

func (me *Type) put_single(tag []byte, data []byte) []byte {
	lazy.Require(len(data) <= hardlimit, `size error`)
	digest := sha256.Sum256(data)
	subject, to := hex.EncodeToString(digest[:]), hex.EncodeToString(tag)
	if len(lazy.Unwrap(me.mail.Search(lazy.With(imap.NewSearchCriteria(), func(sc *imap.SearchCriteria) { sc.Header.Add("Subject", subject) }, func(sc *imap.SearchCriteria) { sc.Header.Add("To", to) })))) == 0 {
		lazy.Assert(me.mail.Append(mailbox, nil, time.Now(), strings.NewReader(fmt.Sprintf("Subject: %s\r\nTo: %s\r\n\r\n%s", subject, to, base64.StdEncoding.EncodeToString(data)))))
	}
	return digest[:]
}

func (me *Type) put_multiple(tag []byte, data []byte) []byte {
	if len(data) < hardlimit {
		return me.put_single(tag, data)
	}
	sx, sy := softlimit, (len(data)/softlimit+1)/2*softlimit
	p, px, py := data[:sx], data[sx:sy], data[sy:]
	hx, hy := me.put_single(tag, px), me.put_single(tag, py)
	return me.put_single(tag, bytes.Join([][]byte{p, hx, hy}, nil))
}

func (me *Type) put(path string, data []byte) []byte {
	return me.put_multiple([]byte(TAGATTR), bytes.Join([][]byte{me.put_multiple([]byte(TAGDATA), data), []byte(path)}, nil))
}

func (me *Type) Put(path string, data []byte) []byte {
	return me.put_multiple([]byte(TAGATTR), bytes.Join([][]byte{me.put_multiple([]byte(TAGDATA), data), []byte(path)}, nil))
}

// func (me *Type) PutObject(data []byte) []byte {
// 	if len(data) < sizelimit {
// 		return me.PutValue(data)
// 	}
// 	this, that := data[:sizelimit-64], data[sizelimit-64:]
// 	sep := len(that) / 2
// 	x, y := that[:sep], that[sep:]
// 	l, r := me.PutObject(x), me.PutObject(y)
// 	payload := make([]byte, sizelimit)
// 	copy(payload[0:32], l)
// 	copy(payload[32:64], r)
// 	copy(payload[64:], this)
// 	return me.PutValue(payload)
// }

// func (me *Type) PutValue(data []byte) []byte {
// 	if len(data) > sizelimit {
// 		panic(data)
// 	}
// 	digest := sha256.Sum256(data)
// 	subject := hex.EncodeToString(digest[:])
// 	sc := imap.NewSearchCriteria()
// 	sc.Header.Add("Subject", subject)
// 	if len(lazy.Unwrap(me.mail.Search(sc))) > 0 {
// 		return digest[:]
// 	}
// 	reader := strings.NewReader(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, base64.StdEncoding.EncodeToString(data)))
// 	lazy.Assert(me.mail.Append(mailbox, nil, time.Now(), reader))
// 	return digest[:]
// }

func (me *Type) GetValue(digest []byte) []byte {
	lazy.Require(len(digest) == 32, `invalid hash`)
	subject := hex.EncodeToString(digest)
	sc := imap.NewSearchCriteria()
	sc.Header.Add("Subject", subject)
	uids := lazy.Unwrap(me.mail.UidSearch(sc))
	for _, uid := range uids {
		ch := make(chan *imap.Message, 1)
		ss := new(imap.SeqSet)
		ss.AddNum(uid)
		lazy.Assert(me.mail.UidFetch(ss, []imap.FetchItem{imap.FetchRFC822Text}, ch))
		for msg := range ch {
			for _, text := range msg.Body {
				data := lazy.Unwrap(io.ReadAll(base64.NewDecoder(base64.StdEncoding, text)))
				dig := sha256.Sum256(data)
				if bytes.Equal(dig[:], digest) {
					return data
				}
			}
		}
	}
	panic(`not found`)
}

func (me *Type) GetObject(digest []byte) []byte {
	data := me.GetValue(digest)
	if len(data) < sizelimit {
		return data
	}
	l, r, this := data[0:32], data[32:64], data[64:]
	x, y := me.GetObject(l), me.GetObject(r)
	payload := make([]byte, len(x)+len(y)+len(data)-64)
	copy(payload[0:len(this)], this)
	copy(payload[len(this):len(this)+len(x)], x)
	copy(payload[len(this)+len(x):], y)
	return payload
}

func (me *Type) Close() {
	me.mail.Logout()
}

const (
	mailbox   = `MDDATA`
	hardlimit = 64 * 1024
	softlimit = hardlimit - 64

	TAGATTR = `ATTR`
	TAGDATA = `DATA`
)
