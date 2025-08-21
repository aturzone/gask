package main

import (
    "fmt"
    "time"
)

// Simple Shamsi (Persian) calendar conversion
// Note: This is a simplified implementation. For production use, consider using a proper Jalali library

var (
    // Days in each month for Persian calendar (non-leap year)
    persianMonthDays = []int{31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30, 29}
    // Days in each month for Persian calendar (leap year) 
    persianMonthDaysLeap = []int{31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30, 30}
)

// isPersianLeapYear checks if a Persian year is a leap year
func isPersianLeapYear(year int) bool {
    // Simplified leap year calculation for Persian calendar
    cycle := year % 128
    if cycle <= 29 {
        return cycle%4 == 1
    } else if cycle <= 62 {
        return (cycle-29)%4 == 1
    } else if cycle <= 95 {
        return (cycle-62)%4 == 1
    } else {
        return (cycle-95)%4 == 1
    }
}

// ToShamsi converts Gregorian time to Shamsi string (YYYY-MM-DD)
func ToShamsi(t time.Time) string {
    // Simplified conversion - for production use proper library like github.com/yaa110/go-persian-calendar
    year, month, day := gregorianToJalali(t.Year(), int(t.Month()), t.Day())
    return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

// FromShamsi parses Shamsi string to Gregorian time
func FromShamsi(shamsiDate string) (time.Time, error) {
    var y, m, d int
    _, err := fmt.Sscanf(shamsiDate, "%d-%d-%d", &y, &m, &d)
    if err != nil {
        return time.Time{}, fmt.Errorf("invalid Shamsi date format: %v", err)
    }
    
    gYear, gMonth, gDay := jalaliToGregorian(y, m, d)
    return time.Date(gYear, time.Month(gMonth), gDay, 0, 0, 0, 0, time.UTC), nil
}

// Simplified Gregorian to Jalali conversion
func gregorianToJalali(gYear, gMonth, gDay int) (int, int, int) {
    // This is a simplified implementation
    // For accurate conversion, use a proper library
    
    // Persian calendar epoch: March 22, 622 CE
    const persianEpoch = 1948321 // Julian day number for Persian epoch
    
    jd := gregorianToJulianDay(gYear, gMonth, gDay)
    persianDay := jd - persianEpoch
    
    // Approximate conversion (simplified)
    pYear := 1 + int(persianDay/365.24)
    remainingDays := persianDay - int(float64(pYear-1)*365.24)
    
    if remainingDays <= 0 {
        pYear--
        if isPersianLeapYear(pYear) {
            remainingDays += 366
        } else {
            remainingDays += 365
        }
    }
    
    // Find month and day
    monthDays := persianMonthDays
    if isPersianLeapYear(pYear) {
        monthDays = persianMonthDaysLeap
    }
    
    pMonth := 1
    for i, days := range monthDays {
        if remainingDays <= days {
            pMonth = i + 1
            break
        }
        remainingDays -= days
    }
    
    pDay := remainingDays
    if pDay <= 0 {
        pDay = 1
    }
    
    return pYear, pMonth, int(pDay)
}

// Simplified Jalali to Gregorian conversion
func jalaliToGregorian(pYear, pMonth, pDay int) (int, int, int) {
    // This is a simplified implementation
    // For accurate conversion, use a proper library
    
    const persianEpoch = 1948321
    
    // Calculate days from Persian epoch
    totalDays := 0
    
    // Add days for complete years
    for y := 1; y < pYear; y++ {
        if isPersianLeapYear(y) {
            totalDays += 366
        } else {
            totalDays += 365
        }
    }
    
    // Add days for complete months in current year
    monthDays := persianMonthDays
    if isPersianLeapYear(pYear) {
        monthDays = persianMonthDaysLeap
    }
    
    for i := 0; i < pMonth-1; i++ {
        totalDays += monthDays[i]
    }
    
    // Add remaining days
    totalDays += pDay
    
    // Convert to Julian day
    jd := persianEpoch + totalDays
    
    return julianDayToGregorian(jd)
}

// Helper function to convert Gregorian date to Julian day number
func gregorianToJulianDay(year, month, day int) int {
    if month <= 2 {
        year--
        month += 12
    }
    
    a := year / 100
    b := 2 - a + a/4
    
    jd := int(365.25*float64(year+4716)) + int(30.6001*float64(month+1)) + day + b - 1524
    return jd
}

// Helper function to convert Julian day number to Gregorian date
func julianDayToGregorian(jd int) (int, int, int) {
    a := jd + 32044
    b := (4*a + 3) / 146097
    c := a - (146097*b)/4
    d := (4*c + 3) / 1461
    e := c - (1461*d)/4
    m := (5*e + 2) / 153
    
    day := e - (153*m+2)/5 + 1
    month := m + 3 - 12*(m/10)
    year := 100*b + d - 4800 + m/10
    
    return year, month, day
}
