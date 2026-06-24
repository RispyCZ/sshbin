package sharing

import "strings"

func SetupURL(baseURL, sharingID string) string {
	return strings.TrimRight(baseURL, "/") + "/setup/" + sharingID
}
