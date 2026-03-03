package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// rangeRe matches "9-10pm", "9am-10am", "9:30-10:30pm", "9 - 10pm", etc.
var rangeRe = regexp.MustCompile(`(?i)(\d{1,2}(?::\d{2})?\s*(?:am|pm)?)\s*-\s*(\d{1,2}(?::\d{2})?\s*(?:am|pm))`)

// byRe matches "by 8pm", "by 8:30pm".
var byRe = regexp.MustCompile(`(?i)^by\s+(\d{1,2}(?::\d{2})?\s*(?:am|pm))$`)

// singleRe matches a standalone time like "9pm".
var singleRe = regexp.MustCompile(`(?i)^(\d{1,2}(?::\d{2})?\s*(?:am|pm))$`)

// ParseTimeWindow parses a time-window string relative to now's date.
// Supported formats:
//   "9-10pm"       → 9:00pm – 10:00pm today (or tomorrow if that window has passed)
//   "9am-10am"     → 9:00am – 10:00am today (or tomorrow if that window has passed)
//   "by 8pm"       → now – 8:00pm today (or tomorrow 8pm if 8pm has already passed)
//   "9pm"          → 9:00pm – 10:00pm today (or tomorrow if that window has passed)
//
// If the end time has already passed relative to now, the entire window is rolled
// forward by 24 hours so that late-evening texts don't immediately expire.
func ParseTimeWindow(s string, now time.Time) (start, end time.Time, ok bool) {
	s = strings.TrimSpace(s)

	if m := byRe.FindStringSubmatch(s); m != nil {
		endT, err := parseHHMM(m[1], now)
		if err != nil {
			return
		}
		// "by X" where X has already passed → assume tomorrow.
		if !endT.After(now) {
			endT = endT.Add(24 * time.Hour)
		}
		return now, endT, true
	}

	if m := rangeRe.FindStringSubmatch(s); m != nil {
		s2, err1 := parseHHMM(m[1], now)
		e2, err2 := parseHHMM(m[2], now)
		if err1 == nil && err2 == nil {
			// Entire window is in the past → roll both forward.
			if !e2.After(now) {
				s2 = s2.Add(24 * time.Hour)
				e2 = e2.Add(24 * time.Hour)
			}
			return s2, e2, true
		}
	}

	if m := singleRe.FindStringSubmatch(s); m != nil {
		t, err := parseHHMM(m[1], now)
		if err == nil {
			endT := t.Add(time.Hour)
			if !endT.After(now) {
				t = t.Add(24 * time.Hour)
				endT = endT.Add(24 * time.Hour)
			}
			return t, endT, true
		}
	}

	return
}

// FindTimeWindow searches text for a time-window token and returns it plus
// the text with that token removed.
func FindTimeWindow(text string, now time.Time) (start, end time.Time, remainder string, ok bool) {
	// Try "by Xpm" first
	byFull := regexp.MustCompile(`(?i)\bby\s+\d{1,2}(?::\d{2})?\s*(?:am|pm)\b`)
	if loc := byFull.FindStringIndex(text); loc != nil {
		token := text[loc[0]:loc[1]]
		s, e, parsed := ParseTimeWindow(token, now)
		if parsed {
			remainder = strings.TrimSpace(text[:loc[0]] + " " + text[loc[1]:])
			return s, e, remainder, true
		}
	}

	// Try "X-Ypm"
	rangeFull := regexp.MustCompile(`(?i)\d{1,2}(?::\d{2})?\s*(?:am|pm)?\s*-\s*\d{1,2}(?::\d{2})?\s*(?:am|pm)`)
	if loc := rangeFull.FindStringIndex(text); loc != nil {
		token := text[loc[0]:loc[1]]
		s, e, parsed := ParseTimeWindow(token, now)
		if parsed {
			remainder = strings.TrimSpace(text[:loc[0]] + " " + text[loc[1]:])
			return s, e, remainder, true
		}
	}

	return
}

// WindowsOverlap returns true when [aStart,aEnd) and [bStart,bEnd) overlap.
func WindowsOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

// OverlapDuration returns the length of overlap between two windows.
func OverlapDuration(aStart, aEnd, bStart, bEnd time.Time) time.Duration {
	lo := aStart
	if bStart.After(lo) {
		lo = bStart
	}
	hi := aEnd
	if bEnd.Before(hi) {
		hi = bEnd
	}
	if hi.Before(lo) {
		return 0
	}
	return hi.Sub(lo)
}

// parseHHMM parses "9pm", "9:30am", "21:00" etc. relative to now's date.
func parseHHMM(s string, now time.Time) (time.Time, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	isPM := strings.HasSuffix(s, "pm")
	isAM := strings.HasSuffix(s, "am")
	s = strings.TrimSuffix(strings.TrimSuffix(s, "pm"), "am")
	s = strings.TrimSpace(s)

	var hour, min int
	parts := strings.SplitN(s, ":", 2)
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}, fmt.Errorf("bad hour %q", parts[0])
	}
	hour = h
	if len(parts) == 2 {
		m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return time.Time{}, fmt.Errorf("bad minute %q", parts[1])
		}
		min = m
	}

	if isPM && hour != 12 {
		hour += 12
	}
	if isAM && hour == 12 {
		hour = 0
	}
	if hour > 23 || min > 59 {
		return time.Time{}, fmt.Errorf("time out of range %d:%02d", hour, min)
	}

	loc := now.Location()
	y, mo, d := now.Date()
	return time.Date(y, mo, d, hour, min, 0, 0, loc), nil
}
