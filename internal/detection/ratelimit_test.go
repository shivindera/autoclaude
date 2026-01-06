package detection

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return string(data)
}

func TestCheckRateLimit_NewFormat(t *testing.T) {
	content := loadFixture(t, "rate_limit_new_format.txt")
	status := CheckRateLimit(content)

	if !status.IsLimited {
		t.Error("expected IsLimited to be true")
	}
	if status.ResetsAt != "10pm" {
		t.Errorf("expected ResetsAt to be '10pm', got '%s'", status.ResetsAt)
	}
	if status.ResetTime.IsZero() {
		t.Error("expected ResetTime to be set")
	}
}

func TestCheckRateLimit_OldFormat(t *testing.T) {
	content := loadFixture(t, "rate_limit_old_format.txt")
	status := CheckRateLimit(content)

	if !status.IsLimited {
		t.Error("expected IsLimited to be true")
	}
	if status.ResetsAt != "2pm" {
		t.Errorf("expected ResetsAt to be '2pm', got '%s'", status.ResetsAt)
	}
	if status.ResetTime.IsZero() {
		t.Error("expected ResetTime to be set")
	}
}

func TestCheckRateLimit_NoMatch(t *testing.T) {
	content := loadFixture(t, "not_claude_code.txt")
	status := CheckRateLimit(content)

	if status.IsLimited {
		t.Error("expected IsLimited to be false")
	}
}

func TestCheckRateLimit_TimeFormats(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		wantTime string
	}{
		{
			name:     "simple pm",
			content:  "You've hit your limit · resets 2pm",
			wantTime: "2pm",
		},
		{
			name:     "simple am",
			content:  "You've hit your limit · resets 9am",
			wantTime: "9am",
		},
		{
			name:     "with minutes",
			content:  "limit reached ∙ resets 10:30am",
			wantTime: "10:30am",
		},
		{
			name:     "with space before am/pm",
			content:  "limit reached ∙ resets 3 pm",
			wantTime: "3 pm",
		},
		{
			name:     "double digit hour",
			content:  "You've hit your limit · resets 11pm (Europe/London)",
			wantTime: "11pm",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := CheckRateLimit(tc.content)
			if !status.IsLimited {
				t.Error("expected IsLimited to be true")
			}
			if status.ResetsAt != tc.wantTime {
				t.Errorf("expected ResetsAt to be '%s', got '%s'", tc.wantTime, status.ResetsAt)
			}
		})
	}
}

func TestHasReset(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name   string
		status RateLimitStatus
		want   bool
	}{
		{
			name:   "not limited",
			status: RateLimitStatus{IsLimited: false},
			want:   false,
		},
		{
			name:   "limited but no reset time",
			status: RateLimitStatus{IsLimited: true},
			want:   false,
		},
		{
			name: "limited, reset time in future",
			status: RateLimitStatus{
				IsLimited: true,
				ResetTime: now.Add(1 * time.Hour),
			},
			want: false,
		},
		{
			name: "limited, reset time in past",
			status: RateLimitStatus{
				IsLimited: true,
				ResetTime: now.Add(-1 * time.Hour),
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.HasReset()
			if got != tc.want {
				t.Errorf("HasReset() = %v, want %v", got, tc.want)
			}
		})
	}
}
