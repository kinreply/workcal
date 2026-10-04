# workcal

A Go library for the Serbian calendar: dates in Europe/Belgrade, calendar-day
and month arithmetic, Orthodox Easter, the public non-working holidays, and
"N working days after receipt" deadlines.

It has no dependencies outside the Go standard library and holds no policy: it
does not know which law gives which period, only how to count one.

## How it works

- **Dates are Belgrade dates.** `DateOf(t)` takes the calendar date of `t` in
  Europe/Belgrade, so 23:30 UTC on Dec 31 is Jan 1. The zone database is
  embedded (`time/tzdata`), so it works on images with no zoneinfo.
- **Deadlines are the end of a day**: 23:59:59.999 Belgrade time.
- **The day of receipt is never counted** (Regulation (EEC, Euratom) No
  1182/71 Art. 3(1)). 30 days from the 1st ends at the end of the 31st; 3
  working days from a Thursday ends at the end of the following Tuesday.
- **Arithmetic is DST-safe.** A `Date` has no time and no zone; adding days goes
  through UTC midnight, where every day is 24 hours, so the 23- and 25-hour days
  in March and October cannot shift a date. `EndOfDay` then puts the result back
  in Belgrade, at +1 or +2 as the date requires.
- **Months clamp** (Regulation 1182/71 Art. 3(2)(c)): Jan 31 + 1 month is Feb 28,
  or Feb 29 in a leap year. Go's `time.AddDate` would give Mar 3.
- **Working days** are Monday to Friday, optionally excluding Serbian public
  non-working holidays. `AddWorkingDays(received, n, skipHolidays)` steps forward
  one date at a time from the day after receipt and returns the end of the n-th
  working day.

### The holiday rules and their sources

From the Zakon o državnim i drugim praznicima u Republici Srbiji ("Sl. glasnik
RS", br. 43/2001, 101/2007 i 92/2011):

| Rule | Days off |
|---|---|
| čl. 1 and 1a, state holidays | Nova godina Jan 1-2; Sretenje (Dan državnosti) Feb 15-16; Praznik rada May 1-2; Dan primirja Nov 11 |
| čl. 2, religious holidays for all | prvi dan Božića Jan 7; Vaskrs from Good Friday to Easter Monday (Orthodox) |
| čl. 3a, Sunday carry-over | a **state** holiday on a Sunday makes the first following working day off. This applies **only** to the state holidays (Nova godina, Sretenje, Praznik rada, Dan primirja), **not** to the religious ones in čl. 2: Božić on a Sunday (Jan 7 2018, Jan 7 2024) is not carried, and neither is any Easter day. "First following working day" skips days already off: Jan 1 2023 (Sunday) moved to Tue Jan 3, past the Jan 2 holiday; May 2 2021 (Sunday) moved to Tue May 4, past Easter Monday. |
| čl. 3, working holidays | Dan pobede (May 9) is in the čl. 1a list of state holidays, but čl. 3 says state holidays are not worked "osim na Dan pobede koji se praznuje radno", so it is not a day off. Not carrying it when it falls on a Sunday (May 9 2021) is an **interpretation**: čl. 3a speaks of state holidays generally, but a holiday that is worked anyway gives no day off to move. |

Sources: the law's text at
paragraf.rs/propisi/zakon_o_drzavnim_i_drugim_praznicima_u_republici_srbiji.html
and propisi.net/zakon-o-drzavnim-i-drugim-praznicima-u-republici-srbiji/ (read
2026-09-27); the carry-over outcomes checked against published non-working-day
calendars, e.g. ipc.rs/statisticki_podaci/2023/praznici-neradni-dani-u-2023-godini
(2023) and unija.com/sr/koji-su-to-neradni-dani-za-prvomajske-i-vaskrsnje-praznike/
(2021).

Orthodox Easter is Meeus's Julian Easter algorithm plus 13 days.

### Verified versus computed

| Range | Status |
|---|---|
| 2018, 2021-2027 (8 years) | Full holiday set pinned in tests, written out from the law and checked against published calendars |
| 1900, 2018, 2021-2027 | Orthodox Easter date pinned in tests |
| 1900-2099 (200 years) | Computed. Tests check every Easter falls on a Sunday; the holiday set follows from the same rules but no year outside the pinned ones has been compared to a published calendar |
| before 1900, after 2099 | `OrthodoxEaster` panics, and so does everything that needs the holiday set: `IsSerbianHoliday`, `SerbianHolidays`, `IsWorkingDay(d, true)`, `AddWorkingDays(t, n, true)`. Weekend-only counting (`skipHolidays=false`) works for any year |

## Install

```sh
go get github.com/kinreply/workcal
```

Requires Go 1.27 or later (the `go` directive in go.mod).

## Usage

```go
import "github.com/kinreply/workcal"

received := time.Date(2026, 4, 9, 10, 0, 0, 0, workcal.Location()) // Thursday before Easter

// 3 working days, skipping Good Friday to Easter Monday: end of Thu Apr 16.
due := workcal.AddWorkingDays(received, 3, true)

// Weekends only: end of Tue Apr 14.
due = workcal.AddWorkingDays(received, 3, false)

// 30 days or one calendar month from receipt, whichever ends first.
r := workcal.DateOf(received)
due = workcal.Earlier(r.AddDays(30), r.AddMonthsClamped(1)).EndOfDay()

// Whole calendar days left.
left := workcal.DaysBetween(workcal.DateOf(time.Now()), workcal.DateOf(due))

workcal.IsSerbianHoliday(time.Date(2026, 2, 17, 12, 0, 0, 0, workcal.Location())) // true: Feb 15 was a Sunday
workcal.SerbianHolidays(2026) // 13 dates, in order
```

### API

| Name | What |
|---|---|
| `Location() *time.Location` | Europe/Belgrade |
| `type Date struct{ Year int; Month time.Month; Day int }` | a calendar date, no time, no zone |
| `DateOf(t time.Time) Date` | t's Belgrade date |
| `(Date) AddDays(n int) Date` | n calendar days later |
| `(Date) AddMonthsClamped(n int) Date` | same date n months later, clamped to the month's last day |
| `(Date) Before(o Date) bool`, `(Date) Weekday() time.Weekday` | |
| `(Date) EndOfDay() time.Time` | 23:59:59.999 Belgrade |
| `Earlier(a, b Date) Date` | the first of two dates |
| `DaysBetween(from, to Date) int` | calendar days from `from` to `to` |
| `IsWorkingDay(d Date, skipHolidays bool) bool` | Monday-Friday, optionally not a holiday |
| `AddWorkingDays(received time.Time, n int, skipHolidays bool) time.Time` | end of the n-th working day after receipt |
| `IsSerbianHoliday(t time.Time) bool` | t's Belgrade date is a non-working holiday |
| `SerbianHolidays(year int) []Date` | the year's non-working holidays, in order |
| `OrthodoxEaster(year int) time.Time` | Easter Sunday, midnight Belgrade |
| `EasterMinYear`, `EasterMaxYear` | 1900, 2099 |

### Cost

The holiday set is rebuilt on every call rather than cached. On a Ryzen 5 5600:
`IsSerbianHoliday` 1.1 µs and 5 allocations; `AddWorkingDays(t, 3, true)` across
Easter 6.0 µs and 25 allocations. Fine for deadlines on requests; measure before
putting it in a loop over millions of dates.

## Development

```sh
make check   # gofmt -l, go vet, go test -race
```

## Limitations

- **Year bounds.** Orthodox Easter is computed as the Julian date plus 13 days,
  which holds only from 1900 to 2099: in 2100 the Julian-Gregorian gap becomes
  14 days. Outside those years `OrthodoxEaster` and everything that needs the
  holiday set panic, on purpose: a zero `time.Time` is year 1, and a calendar
  silently off by a day is worse than a crash. Weekend-only counting is not
  bounded.
- **Holidays of other faiths are not counted.** Under čl. 2 of the law, holidays
  of other faiths (Catholic Christmas, Bajram, Yom Kippur and others) are days
  off only for employees of that faith. They are not general
  non-working days, so they are not in the set.
- **No end-of-period roll-forward.** Regulation 1182/71 Art. 3(4), and similar
  Serbian procedural rules, move a period that ends on a weekend or holiday to the
  next working day. `AddDays` and `AddMonthsClamped` do not apply it, so a date
  they give is never later than the law's and at worst a day or two early. A
  caller that needs it can step forward with `IsWorkingDay`. (`AddWorkingDays`
  always ends on a working day by construction.)
- **The law as of "Sl. glasnik RS" br. 92/2011.** The rules are code, not data.
  A future amendment, or a one-off government decision adding a non-working day,
  needs a code change and a new release. The same rules are applied to every year
  1900-2099, including years before the law or its amendments took effect, so
  results for those years are not historical fact.
- **One country, one zone.** Other countries' holidays and other time zones are
  out of scope.
- Only the years in the table above are checked against published calendars.

## License

MIT. See [LICENSE](LICENSE).
