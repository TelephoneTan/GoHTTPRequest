package http

import "net/http"

type FlexibleCookieJar interface {
	http.CookieJar
	AsReadOnlyJar() FlexibleCookieJar
	AsWriteOnlyJar() FlexibleCookieJar
	AsReadWriteJar() FlexibleCookieJar
	AsNoJar() FlexibleCookieJar
	WithRead(readable bool) FlexibleCookieJar
	WithWrite(writable bool) FlexibleCookieJar
	WithReadWrite(readable, writable bool) FlexibleCookieJar
	SameTag(tag string) FlexibleCookieJar
	Clear() FlexibleCookieJar
	SetCookiesManually(urlCookieList [][]string)
}
