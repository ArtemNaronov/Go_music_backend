package connectqr

import "encoding/json"

// Payload is encoded into a QR code for the iOS app.
type Payload struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

// Encode returns compact JSON for a QR code.
func Encode(serverURL, token string) string {
	b, _ := json.Marshal(Payload{
		Server: serverURL,
		Token:  token,
	})
	return string(b)
}
