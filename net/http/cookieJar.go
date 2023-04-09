package http

import (
	"github.com/TelephoneTan/GoHTTPRequest/util"
	"golang.org/x/net/publicsuffix"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type _CookieJar struct {
	Jar      http.CookieJar
	Readable bool
	Writable bool
	Tag      string
}

type CookieJar = *_CookieJar

func (c CookieJar) SameTag(tag string) (res FlexibleCookieJar) {
	if tag != c.Tag {
		res = util.Copy(*c, func(c CookieJar) {
			c.Tag = tag
			c.Jar = newJar()
		})
	} else {
		res = c
	}
	return res
}

func (c CookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if !c.Writable {
		return
	}
	c.Jar.SetCookies(u, cookies)
}

func (c CookieJar) Cookies(u *url.URL) []*http.Cookie {
	if !c.Readable {
		return []*http.Cookie{}
	}
	return c.Jar.Cookies(u)
}

func (c CookieJar) require(readable, writable bool) CookieJar {
	if c.Readable == readable && c.Writable == writable {
		return c
	} else {
		return util.Copy(*c, func(c CookieJar) {
			c.Readable = readable
			c.Writable = writable
		})
	}
}

func (c CookieJar) AsReadOnlyJar() FlexibleCookieJar {
	return c.require(true, false)
}

func (c CookieJar) AsWriteOnlyJar() FlexibleCookieJar {
	return c.require(false, true)
}

func (c CookieJar) AsReadWriteJar() FlexibleCookieJar {
	return c.require(true, true)
}

func (c CookieJar) AsNoJar() FlexibleCookieJar {
	return c.require(false, false)
}

func (c CookieJar) WithRead(readable bool) FlexibleCookieJar {
	return c.require(readable, c.Writable)
}

func (c CookieJar) WithWrite(writable bool) FlexibleCookieJar {
	return c.require(c.Readable, writable)
}

func (c CookieJar) WithReadWrite(readable, writable bool) FlexibleCookieJar {
	return c.require(readable, writable)
}

func newJar() http.CookieJar {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return jar
}

func NewCookieJar(jar http.CookieJar, init ...func(CookieJar)) CookieJar {
	if jar == nil {
		jar = newJar()
	}
	return util.New(&_CookieJar{
		Jar:      jar,
		Readable: true,
		Writable: true,
	}, init...)
}
