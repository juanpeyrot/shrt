package shortcode

import "crypto/rand"

const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const length = 7

func Generate() (string, error) {
	result := make([]byte, length)
	for i := range result {
		for {
			var b [1]byte
			if _, err := rand.Read(b[:]); err != nil {
				return "", err
			}
			if b[0] < 248 {
				result[i] = charset[b[0]%62]
				break
			}
		}
	}
	return string(result), nil
}
