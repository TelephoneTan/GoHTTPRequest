package mime

type Type string

var (
	ImageJPEG              Type = "image/jpeg"
	ImagePNG               Type = "image/png"
	ApplicationJSON        Type = "application/json"
	XWWWFormURLEncoded     Type = "application/x-www-form-urlencoded"
	TextPlainUTF8          Type = "text/plain;charset=utf-8"
	ApplicationOctetStream Type = "application/octet-stream"
)
