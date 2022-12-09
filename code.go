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

const mailbox = `MDDATA`
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

func (me *Type) PutObject(data []byte) []byte {
	if len(data) < sizelimit {
		return me.PutValue(data)
	}
	this, that := data[:sizelimit-64], data[sizelimit-64:]
	sep := len(that) / 2
	x, y := that[:sep], that[sep:]
	l, r := me.PutObject(x), me.PutObject(y)
	payload := make([]byte, sizelimit)
	copy(payload[0:32], l)
	copy(payload[32:64], r)
	copy(payload[64:], this)
	return me.PutValue(payload)
}

func (me *Type) PutValue(data []byte) []byte {
	if len(data) > sizelimit {
		panic(data)
	}
	digest := sha256.Sum256(data)
	subject := hex.EncodeToString(digest[:])
	sc := imap.NewSearchCriteria()
	sc.Header.Add("Subject", subject)
	if len(lazy.Unwrap(me.mail.Search(sc))) > 0 {
		return digest[:]
	}
	reader := strings.NewReader(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, base64.StdEncoding.EncodeToString(data)))
	lazy.Assert(me.mail.Append(mailbox, nil, time.Now(), reader))
	return digest[:]
}

func (me *Type) GetValue(digest []byte) []byte {
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
