package workcal

import (
	"slices"
	"testing"
	"time"
)

// bg is a Belgrade wall-clock instant.
func bg(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, Location())
}

// eod is the expected deadline: the last millisecond of a Belgrade date.
func eod(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 23, 59, 59, 999_000_000, Location())
}

func same(t *testing.T, name string, got, want time.Time) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s: got %s, want %s", name, got.In(Location()), want.In(Location()))
	}
}

func TestLocationIsBelgrade(t *testing.T) {
	if Location().String() != "Europe/Belgrade" {
		t.Fatalf("location %s", Location())
	}
}

func TestDateOfIsBelgradeDate(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
		want Date
	}{
		// 23:30 UTC on Dec 31 is 00:30 on Jan 1 in Belgrade (CET, +1).
		{"23:30 UTC is tomorrow in winter", time.Date(2026, 12, 31, 23, 30, 0, 0, time.UTC), Date{2027, 1, 1}},
		// 22:30 UTC in summer is 00:30 CEST (+2) the next day.
		{"22:30 UTC is tomorrow in summer", time.Date(2026, 3, 31, 22, 30, 0, 0, time.UTC), Date{2026, 4, 1}},
		{"21:59 UTC in summer is still today", time.Date(2026, 3, 31, 21, 59, 0, 0, time.UTC), Date{2026, 3, 31}},
		{"last ns of a day", time.Date(2026, 6, 10, 23, 59, 59, 999_999_999, Location()), Date{2026, 6, 10}},
	}
	for _, c := range cases {
		if got := DateOf(c.at); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAddDaysAcrossDST(t *testing.T) {
	// Receipt on the spring-forward day (Mar 29 2026, 02:00 -> 03:00), 30 days on.
	same(t, "spring", DateOf(time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC)).AddDays(30).EndOfDay(), eod(2026, 4, 28))
	// Landing on the fall-back day (Oct 25 2026, 03:00 -> 02:00).
	same(t, "fall", DateOf(bg(2026, 9, 25, 9, 0)).AddDays(30).EndOfDay(), eod(2026, 10, 25))
	// Every day of a year that crosses both DST changes moves by exactly one date.
	d := Date{2026, 1, 1}
	for i := 0; i < 365; i++ {
		next := d.AddDays(1)
		if DaysBetween(d, next) != 1 || !d.Before(next) || next.AddDays(-1) != d {
			t.Fatalf("%v -> %v is not one day", d, next)
		}
		d = next
	}
	if d != (Date{2027, 1, 1}) {
		t.Fatalf("365 days from 2026-01-01 is %v", d)
	}
}

func TestEndOfDayInstants(t *testing.T) {
	// Pin the instants, not only the wall clock: CEST is +2, CET is +1.
	same(t, "spring", Date{2026, 4, 28}.EndOfDay(), time.Date(2026, 4, 28, 21, 59, 59, 999_000_000, time.UTC))
	same(t, "fall-back day is CET", Date{2026, 10, 25}.EndOfDay(), time.Date(2026, 10, 25, 22, 59, 59, 999_000_000, time.UTC))
	same(t, "winter", Date{2026, 1, 31}.EndOfDay(), time.Date(2026, 1, 31, 22, 59, 59, 999_000_000, time.UTC))
}

func TestAddMonthsClamped(t *testing.T) {
	cases := []struct {
		from Date
		n    int
		want Date
	}{
		{Date{2026, 1, 31}, 1, Date{2026, 2, 28}},
		{Date{2028, 1, 31}, 1, Date{2028, 2, 29}},
		{Date{2026, 3, 31}, 1, Date{2026, 4, 30}},
		{Date{2026, 12, 31}, 2, Date{2027, 2, 28}},
		{Date{2026, 11, 15}, 3, Date{2027, 2, 15}},
		{Date{2026, 1, 28}, 1, Date{2026, 2, 28}},
		{Date{2026, 1, 31}, 3, Date{2026, 4, 30}},
		{Date{2027, 11, 30}, 3, Date{2028, 2, 29}},
		{Date{2026, 2, 1}, 1, Date{2026, 3, 1}},
	}
	for _, c := range cases {
		if got := c.from.AddMonthsClamped(c.n); got != c.want {
			t.Errorf("%v + %d months = %v, want %v", c.from, c.n, got, c.want)
		}
	}
}

func TestEarlier(t *testing.T) {
	a, b := Date{2026, 1, 31}, Date{2026, 2, 1}
	if Earlier(a, b) != a || Earlier(b, a) != a || Earlier(a, a) != a {
		t.Fatal("Earlier picked the later date")
	}
}

func TestDaysBetween(t *testing.T) {
	cases := []struct {
		from, to Date
		want     int
	}{
		{Date{2026, 10, 20}, Date{2026, 10, 25}, 5}, // across the fall-back day
		{Date{2026, 3, 28}, Date{2026, 3, 30}, 2},   // across the 23-hour spring day
		{Date{2026, 10, 28}, Date{2026, 10, 25}, -3},
		{Date{2026, 1, 1}, Date{2026, 1, 1}, 0},
		{Date{2027, 12, 31}, Date{2028, 12, 31}, 366},
	}
	for _, c := range cases {
		if got := DaysBetween(c.from, c.to); got != c.want {
			t.Errorf("%v -> %v: got %d, want %d", c.from, c.to, got, c.want)
		}
	}
}

func TestOrthodoxEaster(t *testing.T) {
	known := map[int]time.Time{
		1900: bg(1900, 4, 22, 0, 0),
		2018: bg(2018, 4, 8, 0, 0),
		2021: bg(2021, 5, 2, 0, 0),
		2022: bg(2022, 4, 24, 0, 0),
		2023: bg(2023, 4, 16, 0, 0),
		2024: bg(2024, 5, 5, 0, 0),
		2025: bg(2025, 4, 20, 0, 0),
		2026: bg(2026, 4, 12, 0, 0),
		2027: bg(2027, 5, 2, 0, 0),
	}
	for y, want := range known {
		same(t, "easter", OrthodoxEaster(y), want)
		if OrthodoxEaster(y).Weekday() != time.Sunday {
			t.Errorf("%d: easter is not a Sunday", y)
		}
	}
	for y := EasterMinYear; y <= EasterMaxYear; y++ {
		if wd := OrthodoxEaster(y).Weekday(); wd != time.Sunday {
			t.Fatalf("%d: easter on %s", y, wd)
		}
	}
}

func mustPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", name)
		}
	}()
	f()
}

func TestOrthodoxEasterRangeGuard(t *testing.T) {
	for _, y := range []int{EasterMinYear - 1, EasterMaxYear + 1, 1} {
		mustPanic(t, "OrthodoxEaster", func() { OrthodoxEaster(y) })
	}
	if EasterMinYear != 1900 || EasterMaxYear != 2099 {
		t.Fatalf("range moved to %d..%d; the +13 offset is only valid 1900..2099", EasterMinYear, EasterMaxYear)
	}
	mustPanic(t, "IsSerbianHoliday(zero time)", func() { IsSerbianHoliday(time.Time{}) })
	mustPanic(t, "SerbianHolidays(2100)", func() { SerbianHolidays(2100) })
	mustPanic(t, "AddWorkingDays into 2100", func() { AddWorkingDays(bg(2099, 12, 31, 10, 0), 1, true) })
	// Weekends only never needs Easter, so it works past the bound.
	same(t, "weekends only in 2100", AddWorkingDays(bg(2099, 12, 31, 10, 0), 1, false), eod(2100, 1, 1))
}

func TestIsSerbianHoliday(t *testing.T) {
	cases := []struct {
		name string
		day  time.Time
		want bool
	}{
		{"new year 1", bg(2026, 1, 1, 12, 0), true},
		{"new year 2", bg(2026, 1, 2, 12, 0), true},
		{"jan 3 2026 (sat, no carry)", bg(2026, 1, 3, 12, 0), false},
		{"christmas", bg(2026, 1, 7, 12, 0), true},
		{"jan 6", bg(2026, 1, 6, 12, 0), false},
		{"jan 8", bg(2026, 1, 8, 12, 0), false},
		// 2024: Christmas on a Sunday does NOT carry (čl. 3a is state holidays only).
		{"christmas on sunday: no carry", bg(2024, 1, 8, 12, 0), false},
		{"sretenje 15", bg(2027, 2, 15, 12, 0), true},
		{"sretenje 16", bg(2027, 2, 16, 12, 0), true},
		{"feb 14", bg(2027, 2, 14, 12, 0), false},
		{"feb 17 2027 (wed)", bg(2027, 2, 17, 12, 0), false},
		// 2026: Feb 15 Sunday, Feb 16 Monday is a holiday anyway -> Tue 17.
		{"2026 feb 15 on sunday carries past feb 16", bg(2026, 2, 17, 12, 0), true},
		{"2026 feb 18", bg(2026, 2, 18, 12, 0), false},
		// 2025: Feb 16 Sunday -> Mon 17.
		{"2025 feb 16 on sunday carries to monday", bg(2025, 2, 17, 12, 0), true},
		{"2025 feb 18", bg(2025, 2, 18, 12, 0), false},
		{"labour 1", bg(2026, 5, 1, 12, 0), true},
		{"labour 2", bg(2026, 5, 2, 12, 0), true},
		{"may 3 2026 (sun)", bg(2026, 5, 3, 12, 0), false},
		{"may 4 2026 (mon, no carry: may 2 is sat)", bg(2026, 5, 4, 12, 0), false},
		// 2022: May 1 Sunday, May 2 Monday a holiday -> Tue 3.
		{"2022 may 3 carried", bg(2022, 5, 3, 12, 0), true},
		// 2021: May 2 Sunday and Easter; May 3 Easter Monday -> Tue 4.
		{"2021 good friday", bg(2021, 4, 30, 12, 0), true},
		{"2021 may 4 carried past easter monday", bg(2021, 5, 4, 12, 0), true},
		{"2021 may 5", bg(2021, 5, 5, 12, 0), false},
		// 2023: Jan 1 Sunday, Jan 2 Monday a holiday -> Tue 3.
		{"2023 jan 3 carried", bg(2023, 1, 3, 12, 0), true},
		{"2023 jan 4", bg(2023, 1, 4, 12, 0), false},
		// 2022: Jan 2 Sunday -> Mon 3.
		{"2022 jan 3 carried", bg(2022, 1, 3, 12, 0), true},
		{"2022 jan 4", bg(2022, 1, 4, 12, 0), false},
		{"armistice", bg(2026, 11, 11, 12, 0), true},
		{"nov 10", bg(2026, 11, 10, 12, 0), false},
		{"nov 12 2026 (thu)", bg(2026, 11, 12, 12, 0), false},
		// 2018: Nov 11 Sunday -> Mon 12.
		{"2018 nov 12 carried", bg(2018, 11, 12, 12, 0), true},
		// 2023: Nov 11 Saturday: no carry to Monday.
		{"2023 nov 13 not carried from saturday", bg(2023, 11, 13, 12, 0), false},
		{"victory day is a working holiday", bg(2026, 5, 9, 12, 0), false},
		{"victory day on a weekday (2025, fri)", bg(2025, 5, 9, 12, 0), false},
		{"monday after victory day on saturday (2026)", bg(2026, 5, 11, 12, 0), false},
		// Easter 2026: Apr 12.
		{"2026 holy thursday", bg(2026, 4, 9, 12, 0), false},
		{"2026 good friday", bg(2026, 4, 10, 12, 0), true},
		{"2026 holy saturday", bg(2026, 4, 11, 12, 0), true},
		{"2026 easter", bg(2026, 4, 12, 12, 0), true},
		{"2026 easter monday", bg(2026, 4, 13, 12, 0), true},
		{"2026 easter tuesday", bg(2026, 4, 14, 12, 0), false},
		{"ordinary day", bg(2026, 9, 28, 12, 0), false},
		{"catholic christmas is not in the set", bg(2026, 12, 25, 12, 0), false},
		// Belgrade date, not UTC: 23:30 UTC Dec 31 is Jan 1.
		{"utc dec 31 23:30 is belgrade jan 1", time.Date(2025, 12, 31, 23, 30, 0, 0, time.UTC), true},
		{"utc jan 7 23:30 is belgrade jan 8", time.Date(2026, 1, 7, 23, 30, 0, 0, time.UTC), false},
	}
	for _, c := range cases {
		if got := IsSerbianHoliday(c.day); got != c.want {
			t.Errorf("%s (%s): got %v, want %v", c.name, c.day.In(Location()).Format("Mon 2006-01-02"), got, c.want)
		}
	}
}

// TestSerbianHolidaysPinned pins each verified year's full set, so a holiday
// that goes missing or appears is noticed even on a date no other case names.
// The lists are written out from the law (see IsSerbianHoliday) and checked
// against the published non-working calendars, not taken from the code.
func TestSerbianHolidaysPinned(t *testing.T) {
	d := func(m time.Month, day int) func(int) Date {
		return func(y int) Date { return Date{y, m, day} }
	}
	jan1, jan2, jan3, jan7 := d(1, 1), d(1, 2), d(1, 3), d(1, 7)
	feb15, feb16, feb17 := d(2, 15), d(2, 16), d(2, 17)
	may1, may2, may3, may4 := d(5, 1), d(5, 2), d(5, 3), d(5, 4)
	nov11, nov12 := d(11, 11), d(11, 12)
	want := map[int][]func(int) Date{
		// Easter Apr 8; Nov 11 Sunday -> Mon 12.
		2018: {jan1, jan2, jan7, feb15, feb16, d(4, 6), d(4, 7), d(4, 8), d(4, 9), may1, may2, nov11, nov12},
		// Easter May 2 (also May 2, a Sunday): Apr 30 - May 3, carry to May 4.
		2021: {jan1, jan2, jan7, feb15, feb16, d(4, 30), may1, may2, may3, may4, nov11},
		// Easter Apr 24; Jan 2 Sunday -> Jan 3; May 1 Sunday -> past May 2 to May 3.
		2022: {jan1, jan2, jan3, jan7, feb15, feb16, d(4, 22), d(4, 23), d(4, 24), d(4, 25), may1, may2, may3, nov11},
		// Easter Apr 16; Jan 1 Sunday -> past Jan 2 to Jan 3.
		2023: {jan1, jan2, jan3, jan7, feb15, feb16, d(4, 14), d(4, 15), d(4, 16), d(4, 17), may1, may2, nov11},
		// Easter May 5: May 1 - May 6 off in one run; Jan 7 Sunday does not carry.
		2024: {jan1, jan2, jan7, feb15, feb16, may1, may2, d(5, 3), d(5, 4), d(5, 5), d(5, 6), nov11},
		// Easter Apr 20; Feb 16 Sunday -> Feb 17.
		2025: {jan1, jan2, jan7, feb15, feb16, feb17, d(4, 18), d(4, 19), d(4, 20), d(4, 21), may1, may2, nov11},
		// Easter Apr 12; Feb 15 Sunday -> past Feb 16 to Feb 17.
		2026: {jan1, jan2, jan7, feb15, feb16, feb17, d(4, 10), d(4, 11), d(4, 12), d(4, 13), may1, may2, nov11},
		// Easter May 2 (Sunday), same shape as 2021.
		2027: {jan1, jan2, jan7, feb15, feb16, d(4, 30), may1, may2, may3, may4, nov11},
	}
	for y, fs := range want {
		var exp []Date
		for _, f := range fs {
			exp = append(exp, f(y))
		}
		if got := SerbianHolidays(y); !slices.Equal(got, exp) {
			t.Errorf("%d:\n got  %v\n want %v", y, got, exp)
		}
	}
}

func TestIsWorkingDay(t *testing.T) {
	cases := []struct {
		d                      Date
		withHols, weekendsOnly bool
	}{
		{Date{2026, 9, 28}, true, true},   // Monday
		{Date{2026, 9, 26}, false, false}, // Saturday
		{Date{2026, 9, 27}, false, false}, // Sunday
		{Date{2026, 1, 7}, false, true},   // Christmas, Wednesday
		{Date{2026, 2, 17}, false, true},  // carried Sretenje, Tuesday
	}
	for _, c := range cases {
		if got := IsWorkingDay(c.d, true); got != c.withHols {
			t.Errorf("%v with holidays: got %v", c.d, got)
		}
		if got := IsWorkingDay(c.d, false); got != c.weekendsOnly {
			t.Errorf("%v weekends only: got %v", c.d, got)
		}
	}
}

func TestAddWorkingDays(t *testing.T) {
	cases := []struct {
		name     string
		received time.Time
		n        int
		want     time.Time
		weekends time.Time // skipHolidays=false
	}{
		{"monday", bg(2026, 9, 21, 10, 0), 3, eod(2026, 9, 24), eod(2026, 9, 24)},
		{"thursday crosses a weekend", bg(2026, 9, 24, 10, 0), 3, eod(2026, 9, 29), eod(2026, 9, 29)},
		{"friday", bg(2026, 9, 25, 10, 0), 3, eod(2026, 9, 30), eod(2026, 9, 30)},
		{"saturday", bg(2026, 9, 26, 10, 0), 3, eod(2026, 9, 30), eod(2026, 9, 30)},
		{"sunday", bg(2026, 9, 27, 10, 0), 3, eod(2026, 9, 30), eod(2026, 9, 30)},
		{"22:30 UTC friday is saturday in belgrade", time.Date(2026, 9, 25, 22, 30, 0, 0, time.UTC), 3, eod(2026, 9, 30), eod(2026, 9, 30)},
		{"wednesday late evening", bg(2026, 9, 23, 23, 59), 3, eod(2026, 9, 28), eod(2026, 9, 28)},
		{"one working day from friday", bg(2026, 9, 25, 10, 0), 1, eod(2026, 9, 28), eod(2026, 9, 28)},
		{"ten working days spans two weekends", bg(2026, 9, 21, 10, 0), 10, eod(2026, 10, 5), eod(2026, 10, 5)},
		// Easter 2026: Fri Apr 10 to Mon Apr 13 off.
		{"thursday before easter 2026", bg(2026, 4, 9, 10, 0), 3, eod(2026, 4, 16), eod(2026, 4, 14)},
		{"good friday 2026", bg(2026, 4, 10, 10, 0), 3, eod(2026, 4, 16), eod(2026, 4, 15)},
		{"easter monday 2026", bg(2026, 4, 13, 10, 0), 3, eod(2026, 4, 16), eod(2026, 4, 16)},
		// 2023: Jan 1 Sun, 2 Mon, 3 Tue (carried) off; Jan 7 Christmas is a Saturday.
		{"fri dec 30 2022 into new year 2023", bg(2022, 12, 30, 10, 0), 3, eod(2023, 1, 6), eod(2023, 1, 4)},
		// 2023: Nov 11 Saturday, no carry.
		{"fri nov 10 2023", bg(2023, 11, 10, 10, 0), 3, eod(2023, 11, 15), eod(2023, 11, 15)},
		// 2018: Nov 11 Sunday -> Mon 12 off.
		{"fri nov 9 2018", bg(2018, 11, 9, 10, 0), 3, eod(2018, 11, 15), eod(2018, 11, 14)},
		// 2021: Apr 30 to May 4 off.
		{"thu apr 29 2021", bg(2021, 4, 29, 10, 0), 3, eod(2021, 5, 7), eod(2021, 5, 4)},
		// 2026: Feb 16 Mon and 17 Tue off.
		{"fri feb 13 2026", bg(2026, 2, 13, 10, 0), 3, eod(2026, 2, 20), eod(2026, 2, 18)},
		// Christmas on a weekday.
		{"tue jan 6 2026", bg(2026, 1, 6, 10, 0), 3, eod(2026, 1, 12), eod(2026, 1, 9)},
		// Labour day 2026: Fri May 1 off, Sat May 2 no carry.
		{"thu apr 30 2026", bg(2026, 4, 30, 10, 0), 3, eod(2026, 5, 6), eod(2026, 5, 5)},

		// Orthodox Christmas. 2027: Jan 7 is a Thursday.
		{"wed jan 6 2027, 1 day", bg(2027, 1, 6, 10, 0), 1, eod(2027, 1, 8), eod(2027, 1, 7)},
		{"wed jan 6 2027, 3 days", bg(2027, 1, 6, 10, 0), 3, eod(2027, 1, 12), eod(2027, 1, 11)},
		// 2024: Jan 7 is a Sunday and, being religious, does not carry to Mon 8.
		{"fri jan 5 2024: christmas on sunday", bg(2024, 1, 5, 10, 0), 1, eod(2024, 1, 8), eod(2024, 1, 8)},
		// 2028: Jan 1 Sat, Jan 2 Sun -> Mon 3 off; Jan 7 Fri off.
		{"fri dec 31 2027, 3 days", bg(2027, 12, 31, 10, 0), 3, eod(2028, 1, 6), eod(2028, 1, 5)},
		{"fri dec 31 2027, 5 days", bg(2027, 12, 31, 10, 0), 5, eod(2028, 1, 11), eod(2028, 1, 7)},

		// Sretenje. 2025: Feb 15 Sat, Feb 16 Sun -> Mon 17 off.
		{"fri feb 14 2025, 1 day", bg(2025, 2, 14, 10, 0), 1, eod(2025, 2, 18), eod(2025, 2, 17)},
		{"fri feb 14 2025, 3 days", bg(2025, 2, 14, 10, 0), 3, eod(2025, 2, 20), eod(2025, 2, 19)},
		// 2026: received on Sunday Feb 15 itself; Mon 16 and Tue 17 off.
		{"sun feb 15 2026, 1 day", bg(2026, 2, 15, 10, 0), 1, eod(2026, 2, 18), eod(2026, 2, 16)},

		// Easter Friday to Monday. 2025: Apr 18-21.
		{"thu apr 17 2025, 1 day", bg(2025, 4, 17, 10, 0), 1, eod(2025, 4, 22), eod(2025, 4, 18)},
		{"thu apr 17 2025, 2 days", bg(2025, 4, 17, 10, 0), 2, eod(2025, 4, 23), eod(2025, 4, 21)},
		// 2024: May 1 Wed, 2 Thu, Good Friday 3, Easter 5, Easter Monday 6: one run.
		{"tue apr 30 2024, 1 day", bg(2024, 4, 30, 10, 0), 1, eod(2024, 5, 7), eod(2024, 5, 1)},
		{"tue apr 30 2024, 3 days (lands on victory day, a working day)", bg(2024, 4, 30, 10, 0), 3, eod(2024, 5, 9), eod(2024, 5, 3)},

		// Praznik rada. 2022: May 1 Sun, May 2 Mon off -> Tue 3 carried.
		{"fri apr 29 2022, 1 day", bg(2022, 4, 29, 10, 0), 1, eod(2022, 5, 4), eod(2022, 5, 2)},
		{"fri apr 29 2022, 3 days", bg(2022, 4, 29, 10, 0), 3, eod(2022, 5, 6), eod(2022, 5, 4)},

		// Dan primirja. 2029: Nov 11 Sun -> Mon 12 off.
		{"fri nov 9 2029, 1 day", bg(2029, 11, 9, 10, 0), 1, eod(2029, 11, 13), eod(2029, 11, 12)},
		{"fri nov 9 2029, 3 days", bg(2029, 11, 9, 10, 0), 3, eod(2029, 11, 15), eod(2029, 11, 14)},
		{"sun nov 11 2029, 1 day", bg(2029, 11, 11, 10, 0), 1, eod(2029, 11, 13), eod(2029, 11, 12)},
	}
	for _, c := range cases {
		same(t, c.name+" (holidays)", AddWorkingDays(c.received, c.n, true), c.want)
		same(t, c.name+" (weekends only)", AddWorkingDays(c.received, c.n, false), c.weekends)
	}
}

func TestAddWorkingDaysRejectsNonPositive(t *testing.T) {
	for _, n := range []int{0, -1} {
		mustPanic(t, "AddWorkingDays(n<1)", func() { AddWorkingDays(bg(2026, 9, 21, 10, 0), n, true) })
	}
}

// The three carry-over rulings, each named, so a change to any one fails by
// name. Weekdays are asserted too, so a wrong premise fails loudly rather than
// passing for the wrong reason.

func TestBozicOnSundayIsNotCarried(t *testing.T) {
	// Jan 7 2018 is a Sunday. Božić is a čl. 2 religious holiday; čl. 3a
	// carries state holidays only, so Mon Jan 8 is a working day.
	if wd := (Date{2018, 1, 7}).Weekday(); wd != time.Sunday {
		t.Fatalf("premise: 2018-01-07 is %s", wd)
	}
	if !IsSerbianHoliday(bg(2018, 1, 7, 12, 0)) {
		t.Error("2018-01-07 Božić is not a holiday")
	}
	if IsSerbianHoliday(bg(2018, 1, 8, 12, 0)) {
		t.Error("2018-01-08 is a holiday: Božić was carried")
	}
	same(t, "one working day from Sat Jan 6 2018", AddWorkingDays(bg(2018, 1, 6, 10, 0), 1, true), eod(2018, 1, 8))
}

func TestNewYearOnSundayCarriesToTuesday(t *testing.T) {
	// Jan 1 2023 is a Sunday. čl. 3a: the first following working day is off;
	// Mon Jan 2 is already a holiday, so that is Tue Jan 3. Wed Jan 4 works.
	if wd := (Date{2023, 1, 1}).Weekday(); wd != time.Sunday {
		t.Fatalf("premise: 2023-01-01 is %s", wd)
	}
	for _, c := range []struct {
		d    int
		want bool
	}{{1, true}, {2, true}, {3, true}, {4, false}} {
		if got := IsSerbianHoliday(bg(2023, 1, c.d, 12, 0)); got != c.want {
			t.Errorf("2023-01-%02d: got %v, want %v", c.d, got, c.want)
		}
	}
}

func TestDanPobedeOnSundayIsNotAHolidayAndNotCarried(t *testing.T) {
	// May 9 2021 is a Sunday. Dan pobede is worked (čl. 3), so it is not a
	// non-working day and, by interpretation, nothing carries to Mon May 10.
	if wd := (Date{2021, 5, 9}).Weekday(); wd != time.Sunday {
		t.Fatalf("premise: 2021-05-09 is %s", wd)
	}
	if IsSerbianHoliday(bg(2021, 5, 9, 12, 0)) {
		t.Error("2021-05-09 Dan pobede is a non-working holiday")
	}
	if IsSerbianHoliday(bg(2021, 5, 10, 12, 0)) {
		t.Error("2021-05-10 is a holiday: Dan pobede was carried")
	}
	same(t, "one working day from Fri May 7 2021", AddWorkingDays(bg(2021, 5, 7, 10, 0), 1, true), eod(2021, 5, 10))
}
