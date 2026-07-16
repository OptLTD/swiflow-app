package browser

import "strings"

// LooksLikeBotBlock reports common anti-bot interstitial phrases.
func LooksLikeBotBlock(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"unusual traffic from your computer",
		"unusual traffic from your network",
		"our systems have detected unusual traffic",
		"not a robot",
		"/sorry/index",
		"detected unusual traffic",
		"verify you are human",
		"automated queries",
		"enable javascript to continue",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
