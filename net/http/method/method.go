package method

type Method string

var (
	_GET     Method = "GET"
	GET             = &_GET
	_POST    Method = "POST"
	POST            = &_POST
	_PUT     Method = "PUT"
	PUT             = &_PUT
	_DELETE  Method = "DELETE"
	DELETE          = &_DELETE
	_HEAD    Method = "HEAD"
	HEAD            = &_HEAD
	_OPTIONS Method = "OPTIONS"
	OPTIONS         = &_OPTIONS
	_TRACE   Method = "TRACE"
	TRACE           = &_TRACE
	_PATCH   Method = "PATCH"
	PATCH           = &_PATCH
)
