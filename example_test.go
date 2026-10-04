package workcal_test

import (
	"fmt"
	"time"

	"github.com/kinreply/workcal"
)

func ExampleAddWorkingDays() {
	received := time.Date(2026, 4, 9, 10, 0, 0, 0, workcal.Location()) // Thursday before Easter
	fmt.Println(workcal.AddWorkingDays(received, 3, true).Format(time.DateTime))
	fmt.Println(workcal.AddWorkingDays(received, 3, false).Format(time.DateTime))
	// Output:
	// 2026-04-16 23:59:59
	// 2026-04-14 23:59:59
}

func ExampleDate_AddMonthsClamped() {
	r := workcal.Date{Year: 2026, Month: time.January, Day: 31}
	fmt.Println(r.AddMonthsClamped(1))
	fmt.Println(workcal.Earlier(r.AddDays(30), r.AddMonthsClamped(1)).EndOfDay().Format(time.DateTime))
	// Output:
	// {2026 February 28}
	// 2026-02-28 23:59:59
}

func ExampleSerbianHolidays() {
	for _, d := range workcal.SerbianHolidays(2026) {
		fmt.Printf("%s %d ", d.Month.String()[:3], d.Day)
	}
	fmt.Println()
	// Output:
	// Jan 1 Jan 2 Jan 7 Feb 15 Feb 16 Feb 17 Apr 10 Apr 11 Apr 12 Apr 13 May 1 May 2 Nov 11
}
