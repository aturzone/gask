package main

import (
    "time"

    "github.com/ilius/go-jalali"
)

// ToShamsi converts Miladi time to Shamsi string (YYYY-MM-DD).
func ToShamsi(t time.Time) string {
    jYear, jMonth, jDay := jalali.GregorianToJalali(t.Year(), int(t.Month()), t.Day())
    return fmt.Sprintf("%04d-%02d-%02d", jYear, jMonth, jDay)
}

// FromShamsi parses Shamsi string to Miladi time.
func FromShamsi(shamsiDate string) (time.Time, error) {
    var y, m, d int
    _, err := fmt.Sscanf(shamsiDate, "%d-%d-%d", &y, &m, &d)
    if err != nil {
        return time.Time{}, err
    }
    gYear, gMonth, gDay := jalali.JalaliToGregorian(y, m, d)
    return time.Date(gYear, time.Month(gMonth), gDay, 0, 0, 0, 0, time.UTC), nil
}
