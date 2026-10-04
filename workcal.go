// Package workcal is the Serbian civil and working-day calendar: dates in
// Europe/Belgrade, calendar-day and month arithmetic, Orthodox Easter, the
// public non-working holidays, and working-day counting.
//
// It holds calendar mechanism only. Which period a deadline uses, and which
// law sets it, is the caller's business.
//
// A deadline here is the END OF A DAY in Europe/Belgrade (23:59:59.999 local).
// The day of the triggering event is never counted: a period of 30 days from
// receipt on the 1st ends at the end of the 31st. That is Regulation (EEC,
// Euratom) No 1182/71 Art. 3(1), and the ordinary reading of Serbian law.
//
// # What this deliberately does NOT do
//
// Regulation 1182/71 Art. 3(4) moves a period that ends on a Saturday, Sunday
// or public holiday to the end of the next working day, and Serbian procedural
// law has a similar rule. Both only ever make a deadline LATER. Nothing here
// applies them: AddDays and AddMonthsClamped are purely calendar, so a date
// computed from them is never later than the law's; at worst it is a day or two
// early. That is the cheap direction to be wrong in. A caller that wants the
// roll-forward can apply it with IsWorkingDay.
package workcal

import (
	"fmt"
	"slices"
	"time"
	_ "time/tzdata" // minimal container images have no zoneinfo; without this LoadLocation fails there
)

var belgrade = mustLoad("Europe/Belgrade")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(fmt.Sprintf("workcal: loading %s: %v", name, err))
	}
	return loc
}

// Location is Europe/Belgrade, the zone every date here is taken in.
func Location() *time.Location { return belgrade }

// Date is a calendar date with no time and no zone. Arithmetic on it goes
// through UTC midnight, where every day is 24 hours, so a DST change in
// Belgrade cannot shift a date by one.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// DateOf is t's calendar date in Belgrade.
func DateOf(t time.Time) Date {
	y, m, d := t.In(belgrade).Date()
	return Date{y, m, d}
}

func (d Date) utc() time.Time { return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC) }

// AddDays is the date n calendar days later (earlier for negative n).
func (d Date) AddDays(n int) Date {
	y, m, dd := d.utc().AddDate(0, 0, n).Date()
	return Date{y, m, dd}
}

// Before reports whether d is an earlier date than o.
func (d Date) Before(o Date) bool { return d.utc().Before(o.utc()) }

// Weekday is d's day of the week.
func (d Date) Weekday() time.Weekday { return d.utc().Weekday() }

// EndOfDay is the last millisecond of the date in Belgrade.
func (d Date) EndOfDay() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 23, 59, 59, 999_000_000, belgrade)
}

// AddMonthsClamped is "the same date n months later", and when the target
// month has no such date, its last day. That is Regulation 1182/71 Art.
// 3(2)(c): a period in months ends on the day "which is the same date as the
// day during which the event ... occurred", and "if, in a period expressed in
// months ..., the day on which it should expire does not occur in the last
// month, the period shall end with the expiry of the last hour of the last day
// of that month". So Jan 31 + 1 month is Feb 28 (Feb 29 in a leap year).
//
// time.AddDate(0, n, 0) is NOT this: it normalises Feb 31 to Mar 3, which
// would show a deadline three days after the law's.
func (d Date) AddMonthsClamped(n int) Date {
	first := time.Date(d.Year, d.Month+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	y, m, _ := first.Date()
	last := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := d.Day
	if day > last {
		day = last
	}
	return Date{y, m, day}
}

// Earlier is whichever of a and b comes first.
func Earlier(a, b Date) Date {
	if b.Before(a) {
		return b
	}
	return a
}

// DaysBetween is the number of calendar days from `from` to `to`: positive
// when to is later, negative when it is earlier, 0 on the same date. It counts
// dates, not 24-hour spans, so a 23- or 25-hour DST day is still one day.
func DaysBetween(from, to Date) int {
	return int((to.utc().Unix() - from.utc().Unix()) / 86400)
}

// IsWorkingDay reports whether d is Monday to Friday and, when skipHolidays is
// true, not a Serbian public non-working holiday (SerbianHolidays).
//
// With skipHolidays it panics outside EasterMinYear..EasterMaxYear (see
// OrthodoxEaster).
func IsWorkingDay(d Date, skipHolidays bool) bool {
	if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !skipHolidays || !holidaySet(d.Year)[d]
}

// AddWorkingDays is the end (in Belgrade) of the n-th working day after the
// date of received, the day of receipt not counted, whatever day it is. A
// working day is as IsWorkingDay defines it: skipHolidays=true skips Serbian
// non-working holidays as well as weekends, false skips weekends only.
//
// "Within 3 working days" of a report received on a Thursday ends at the end
// of the following Tuesday; received on a Saturday, at the end of Wednesday.
//
// It panics if n < 1, and, with skipHolidays, if the count reaches a year
// outside EasterMinYear..EasterMaxYear.
func AddWorkingDays(received time.Time, n int, skipHolidays bool) time.Time {
	if n < 1 {
		panic(fmt.Sprintf("workcal: AddWorkingDays(n=%d): n must be at least 1", n))
	}
	d := DateOf(received)
	for counted := 0; counted < n; {
		d = d.AddDays(1)
		if IsWorkingDay(d, skipHolidays) {
			counted++
		}
	}
	return d.EndOfDay()
}

// EasterMinYear and EasterMaxYear bound OrthodoxEaster: in these years the
// Julian calendar runs exactly 13 days behind the Gregorian for every date
// Easter can fall on. 1900 is inside because its Julian Feb 29 precedes the
// earliest Easter; 2100 is outside because the gap becomes 14 on its (Julian)
// Mar 1.
const (
	EasterMinYear = 1900
	EasterMaxYear = 2099
)

// OrthodoxEaster is the Orthodox Easter Sunday of year as a Gregorian date
// (midnight, Belgrade): Meeus's Julian Easter algorithm, plus 13 days.
//
// It panics outside EasterMinYear..EasterMaxYear. A date in those years is a
// bug in the caller (a zero time.Time is year 1), and silently computing a
// holiday calendar off by a day is worse than failing loudly.
func OrthodoxEaster(year int) time.Time {
	if year < EasterMinYear || year > EasterMaxYear {
		panic(fmt.Sprintf("workcal: OrthodoxEaster(%d) is outside %d..%d", year, EasterMinYear, EasterMaxYear))
	}
	a := year % 4
	b := year % 7
	c := year % 19
	d := (19*c + 15) % 30
	e := (2*a + 4*b - d + 34) % 7
	month := (d + e + 114) / 31
	day := (d+e+114)%31 + 1
	// time.Date normalises the +13 across the month end (Apr 30 + 13 = May 13).
	return time.Date(year, time.Month(month), day+13, 0, 0, 0, 0, belgrade)
}

// IsSerbianHoliday reports whether t's Belgrade date is a public non-working
// day in Serbia, under the Zakon o državnim i drugim praznicima u Republici
// Srbiji ("Sl. glasnik RS", br. 43/2001, 101/2007 i 92/2011):
//
//   - čl. 1 and 1a, state holidays: Nova godina Jan 1-2, Sretenje (Dan
//     državnosti) Feb 15-16, Praznik rada May 1-2, Dan primirja Nov 11.
//     Dan pobede (May 9) is also in the čl. 1a list of state holidays, but
//     čl. 3 makes it a working one: state holidays are not worked "osim na
//     Dan pobede koji se praznuje radno".
//   - čl. 2, religious: prvi dan Božića Jan 7, and Vaskrs "počev od Velikog
//     petka zaključno sa drugim danom Vaskrsa", Good Friday to Easter Monday.
//   - čl. 3a: "Ako jedan od datuma kada se praznuju državni praznici Republike
//     Srbije padne u nedelju, ne radi se prvog narednog radnog dana." It applies
//     ONLY to the state holidays (Nova godina, Sretenje, Praznik rada, Dan
//     primirja), NOT to the religious ones in čl. 2: Božić on a Sunday (as in
//     2018 and 2024) is not carried, and neither is any Easter day. "Prvi naredni
//     radni dan" skips days that are already off: in 2023 Jan 1 was a Sunday,
//     Jan 2 a holiday anyway, so Jan 3 (Tuesday) was off; in 2021 May 2 was a
//     Sunday and May 3 Easter Monday, so May 4 was off. Both match the
//     non-working calendars published for those years.
//
// Sources:
//   - the law's text: paragraf.rs/propisi/zakon_o_drzavnim_i_drugim_praznicima_u_republici_srbiji.html
//     and propisi.net/zakon-o-drzavnim-i-drugim-praznicima-u-republici-srbiji/,
//     read 2026-09-27;
//   - the carry-over outcomes: the published non-working-day calendars, e.g.
//     ipc.rs/statisticki_podaci/2023/praznici-neradni-dani-u-2023-godini (2023)
//     and unija.com/sr/koji-su-to-neradni-dani-za-prvomajske-i-vaskrsnje-praznike/
//     (2021); the tests pin the holiday set for 2018 and 2021-2027 against them.
//
// Dan pobede (May 9) is a WORKING holiday (čl. 3): it never makes a day off.
// Not carrying it when it falls on a Sunday (as in 2021) is an interpretation:
// čl. 3a speaks of state holidays generally, but a holiday that is worked
// anyway gives no day off to move. Holidays of other faiths (Catholic Christmas, Bajram, Yom Kippur...)
// are days off only for employees of that faith (čl. 2) and are not in this
// set.
//
// It panics outside EasterMinYear..EasterMaxYear (see OrthodoxEaster).
func IsSerbianHoliday(t time.Time) bool {
	d := DateOf(t)
	return holidaySet(d.Year)[d]
}

// SerbianHolidays is every public non-working day of year, as IsSerbianHoliday
// defines them, in date order. Days that coincide (May 2 that is also Easter)
// appear once. It panics outside EasterMinYear..EasterMaxYear.
func SerbianHolidays(year int) []Date {
	set := holidaySet(year)
	out := make([]Date, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b Date) int { return DaysBetween(b, a) })
	return out
}

// carries are the non-working čl. 1/1a dates that čl. 3a moves off a Sunday.
// Dan pobede is deliberately absent (see IsSerbianHoliday).
var carries = []struct {
	m time.Month
	d int
}{
	{time.January, 1}, {time.January, 2},
	{time.February, 15}, {time.February, 16},
	{time.May, 1}, {time.May, 2},
	{time.November, 11},
}

func holidaySet(year int) map[Date]bool {
	set := map[Date]bool{{year, time.January, 7}: true}
	for _, c := range carries {
		set[Date{year, c.m, c.d}] = true
	}
	easter := DateOf(OrthodoxEaster(year))
	for off := -2; off <= 1; off++ {
		set[easter.AddDays(off)] = true
	}
	// Carries are applied after every base day is in the set, so "the first
	// following working day" skips Easter Monday as well as the second day of
	// the same holiday. Processed in date order, so a carried day is itself
	// skipped by a later carry.
	for _, c := range carries {
		h := Date{year, c.m, c.d}
		if h.Weekday() != time.Sunday {
			continue
		}
		// Starts on a Monday and skips at most a few holidays, so it never
		// reaches a weekend; only the holiday set needs checking.
		next := h.AddDays(1)
		for set[next] {
			next = next.AddDays(1)
		}
		set[next] = true
	}
	return set
}
