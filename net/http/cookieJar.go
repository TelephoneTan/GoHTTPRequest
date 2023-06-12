package http

import (
	"github.com/TelephoneTan/GoHTTPRequest/util"
	"golang.org/x/net/publicsuffix"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type _CookieJar struct {
	Jar      http.CookieJar
	Readable bool
	Writable bool
	Tag      string
}

type CookieJar = *_CookieJar

func (c CookieJar) Clear() FlexibleCookieJar {
	return util.Copy(*c, func(c CookieJar) {
		c.Jar = selectJar(c.Tag, true)
	})
}

func (c CookieJar) SameTag(tag string) (res FlexibleCookieJar) {
	if tag != c.Tag {
		res = util.Copy(*c, func(c CookieJar) {
			c.Tag = tag
			c.Jar = selectJar(tag, false)
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

func (c CookieJar) SetCookiesManually(urlCookieList [][]string) {
	toSet := map[string][]*http.Cookie{}
	for _, urlCookie := range urlCookieList {
		u := ""
		cookie := ""
		if len(urlCookie) > 0 {
			u = urlCookie[0]
			if len(urlCookie) > 1 {
				cookie = urlCookie[1]
			}
		}
		_, err := url.Parse(u)
		if err == nil {
			header := http.Header{}
			header.Set("Set-Cookie", cookie)
			response := http.Response{Header: header}
			cookies := response.Cookies()
			if len(cookies) > 0 {
				toSet[u] = append(toSet[u], cookies[0])
			}
		}
	}
	for u, cookies := range toSet {
		uu, _ := url.Parse(u)
		c.SetCookies(uu, cookies)
	}
}

type _timeJar struct {
	lastAccessSecond atomic.Int64
	jar              http.CookieJar
}
type timeJar = *_timeJar

func (t timeJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	t.lastAccessSecond.Store(time.Now().Unix())
	t.jar.SetCookies(u, cookies)
}

func (t timeJar) Cookies(u *url.URL) []*http.Cookie {
	t.lastAccessSecond.Store(time.Now().Unix())
	return t.jar.Cookies(u)
}

var jarMap = map[string]timeJar{}
var jarMapLock = sync.Mutex{}

func cleanJarMap() {
	nowSecond := time.Now().Unix()
	for tag, jar := range jarMap {
		if nowSecond-jar.lastAccessSecond.Load() > 300 {
			delete(jarMap, tag)
		}
	}
}

func selectJar(tag string, clear bool) http.CookieJar {
	jarMapLock.Lock()
	defer jarMapLock.Unlock()
	if _, has := jarMap[tag]; !has {
		jarMap[tag] = &_timeJar{}
	}
	jar := jarMap[tag]
	jar.lastAccessSecond.Store(time.Now().Unix())
	if jar.jar == nil || clear {
		jar.jar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	}
	if len(jarMap) > 10_0000 {
		cleanJarMap()
	}
	return jar
}

func NewCookieJar(tag string, init ...func(CookieJar)) CookieJar {
	return util.New(&_CookieJar{
		Jar:      selectJar(tag, false),
		Readable: true,
		Writable: true,
		Tag:      tag,
	}, init...)
}
