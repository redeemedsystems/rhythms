package auth

import "crypto/subtle"

const CSRFHeaderName = "X-CSRF-Token"

func ValidCSRFToken(sessionToken, requestToken string) bool {
	if sessionToken == "" || requestToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(sessionToken), []byte(requestToken)) == 1
}
