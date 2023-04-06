package header

type Header = string

const (
	ContentType     Header = "Content-Type"
	ContentLength   Header = "Content-Length"
	ContentEncoding Header = "Content-Encoding"
	Referer         Header = "Referer"
)
