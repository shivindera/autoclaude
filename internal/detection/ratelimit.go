package detection

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RateLimitStatus represents the rate limit state of a pane
type RateLimitStatus struct {
	IsLimited  bool
	ResetsAt   string    // Original string like "2pm" or "10:30am"
	ResetTime  time.Time // Parsed reset time
	TimeUntil  time.Duration
}

// Rate limit patterns - multiple formats Claude Code uses
// Examples: "limit reached ∙ resets 2pm", "limit reached ∙ resets 10:30am"
//           "You've hit your limit · resets 10pm (Europe/London)"
var rateLimitPatterns = []*regexp.Regexp{
	// New format: "You've hit your limit · resets 10pm (Europe/London)"
	regexp.MustCompile(`(?i)hit\s+your\s+limit.*resets?\s+(\d{1,2}(?::\d{2})?\s*[ap]m)`),
	// Original format: "limit reached ∙ resets 2pm"
	regexp.MustCompile(`(?i)limit\s+reached.*resets?\s+(\d{1,2}(?::\d{2})?\s*[ap]m)`),
}

// CheckRateLimit checks pane content for rate limit messages
func CheckRateLimit(content string) RateLimitStatus {
	// Try each pattern until one matches
	var match []string
	for _, pattern := range rateLimitPatterns {
		match = pattern.FindStringSubmatch(content)
		if match != nil {
			break
		}
	}
	if match == nil {
		return RateLimitStatus{IsLimited: false}
	}

	resetStr := match[1]
	resetTime, err := parseResetTime(resetStr)
	if err != nil {
		// Pattern matched but couldn't parse time - still rate limited
		return RateLimitStatus{
			IsLimited: true,
			ResetsAt:  resetStr,
		}
	}

	now := time.Now()
	timeUntil := resetTime.Sub(now)

	// If the time is in the past, it might be for tomorrow
	if timeUntil < 0 {
		resetTime = resetTime.Add(24 * time.Hour)
		timeUntil = resetTime.Sub(now)
	}

	return RateLimitStatus{
		IsLimited: true,
		ResetsAt:  resetStr,
		ResetTime: resetTime,
		TimeUntil: timeUntil,
	}
}

// parseResetTime parses a time string like "2pm" or "10:30am" into a time.Time for today
func parseResetTime(s string) (time.Time, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	now := time.Now()
	loc := now.Location()

	// Try parsing with minutes first: "10:30am"
	formats := []string{
		"3:04pm",
		"3:04 pm",
		"3pm",
		"3 pm",
	}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, s, loc)
		if err == nil {
			// Combine parsed time with today's date
			return time.Date(now.Year(), now.Month(), now.Day(),
				t.Hour(), t.Minute(), 0, 0, loc), nil
		}
	}

	// Manual parsing as fallback
	isPM := strings.Contains(s, "pm")
	s = strings.ReplaceAll(s, "am", "")
	s = strings.ReplaceAll(s, "pm", "")
	s = strings.TrimSpace(s)

	var hour, minute int
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		hour, _ = strconv.Atoi(parts[0])
		minute, _ = strconv.Atoi(parts[1])
	} else {
		hour, _ = strconv.Atoi(s)
		minute = 0
	}

	// Convert to 24-hour format
	if isPM && hour != 12 {
		hour += 12
	} else if !isPM && hour == 12 {
		hour = 0
	}

	return time.Date(now.Year(), now.Month(), now.Day(),
		hour, minute, 0, 0, loc), nil
}

// HasReset checks if the rate limit has reset (time has passed)
func (r RateLimitStatus) HasReset() bool {
	if !r.IsLimited {
		return false
	}
	if r.ResetTime.IsZero() {
		return false
	}
	return time.Now().After(r.ResetTime)
}
