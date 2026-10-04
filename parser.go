package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseRemindTime(parts []string) (time.Time, int, error) {
	if len(parts) == 0 {
		return time.Time{}, 0, fmt.Errorf("время не указано")
	}

	// 1. 2h30m, 10m, 1h10m10s
	if duration, ok := parseDuration(parts[0]); ok {
		return time.Now().Add(duration), 1, nil
	}

	// 2. 15:40
	if result, ok := parseClockTime(parts[0]); ok {
		return result, 1, nil
	}

	// 3. "через 10 минут", "через 2 часа", "через неделю"
	if result, count, ok := parseRelativeTime(parts); ok {
		return result, count, nil
	}

	// 4. "сегодня", "завтра", "послезавтра"
	if result, count, ok := parseRelativeDay(parts); ok {
		return result, count, nil
	}

	// 5. 02.10.2026 19:40
	if len(parts) >= 2 {
		if result, ok := parseDateTime(parts[0], parts[1]); ok {
			return result, 2, nil
		}
	}

	// 6. 02.10.2026
	if result, ok := parseDate(parts[0]); ok {
		return result, 1, nil
	}

	return time.Time{}, 0, fmt.Errorf("неизвестный формат времени")
}

func parseDuration(text string) (time.Duration, bool) {
	duration, err := time.ParseDuration(text)

	if err != nil {
		return 0, false
	}

	return duration, true
}

func parseClockTime(text string) (time.Time, bool) {
	now := time.Now()

	layouts := []string{
		"15:04",
		"15.04",
	}

	var parsed time.Time
	var err error

	for _, layout := range layouts {
		parsed, err = time.Parse(layout, text)
		if err == nil {
			break
		}
	}

	if err != nil {
		return time.Time{}, false
	}

	result := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		parsed.Hour(),
		parsed.Minute(),
		0,
		0,
		now.Location(),
	)

	if now.After(result) {
		result = result.AddDate(0, 0, 1)
	}

	return result, true
}

// parseRelativeTime разбирает:
//
// через 10 минут
// через 2 часа
// через 3 дня
// через неделю
// через полчаса
// через полтора часа
func parseRelativeTime(parts []string) (time.Time, int, bool) {
	if len(parts) < 2 {
		return time.Time{}, 0, false
	}

	if strings.ToLower(parts[0]) != "через" {
		return time.Time{}, 0, false
	}

	// Особые случаи:
	// "через полчаса"
	// "через полтора часа"
	// "через полторы минуты"
	if duration, ok := parseRussianFraction(parts[1:]); ok {
		return time.Now().Add(duration), 1 + fractionParts(parts[1:]), true
	}

	if len(parts) < 3 {
		return time.Time{}, 0, false
	}

	number, err := strconv.Atoi(parts[1])
	if err != nil {
		var ok bool
		number, ok = parseRussianNumber(strings.ToLower(parts[1]))

		if !ok {
			return time.Time{}, 0, false
		}
	}

	unit := strings.ToLower(parts[2])

	var duration time.Duration

	switch unit {
	case "минуту", "минута", "минуты", "минут", "мин":
		duration = time.Duration(number) * time.Minute

	case "час", "часа", "часов", "ч":
		duration = time.Duration(number) * time.Hour

	case "день", "дня", "дней":
		duration = time.Duration(number) * 24 * time.Hour

	case "неделю", "неделя", "недели", "недель":
		duration = time.Duration(number) * 7 * 24 * time.Hour

	default:
		return time.Time{}, 0, false
	}

	return time.Now().Add(duration), 3, true
}

// parseRussianFraction разбирает:
//
// полчаса
// полтора часа
// полторы минуты
func parseRussianFraction(parts []string) (time.Duration, bool) {
	if len(parts) == 0 {
		return 0, false
	}

	switch strings.ToLower(parts[0]) {
	case "полчаса":
		return 30 * time.Minute, true

	case "полтора":
		if len(parts) >= 2 {
			switch strings.ToLower(parts[1]) {
			case "часа", "час":
				return 90 * time.Minute, true

			case "дня":
				return 36 * time.Hour, true
			}
		}

	case "полторы":
		if len(parts) >= 2 {
			switch strings.ToLower(parts[1]) {
			case "минуты", "минут":
				return 90 * time.Second, true

			case "часа", "час":
				return 90 * time.Minute, true
			}
		}
	}

	return 0, false
}

// fractionParts возвращает количество использованных слов.
func fractionParts(parts []string) int {
	if len(parts) == 0 {
		return 0
	}

	switch strings.ToLower(parts[0]) {
	case "полчаса":
		return 1

	case "полтора", "полторы":
		if len(parts) >= 2 {
			return 2
		}
	}

	return 0
}

// parseRelativeDay разбирает:
//
// сегодня
// завтра
// послезавтра
func parseRelativeDay(parts []string) (time.Time, int, bool) {
	if len(parts) == 0 {
		return time.Time{}, 0, false
	}

	now := time.Now()

	var result time.Time

	switch strings.ToLower(parts[0]) {
	case "сегодня":
		result = now

	case "завтра":
		result = now.AddDate(0, 0, 1)

	case "послезавтра":
		result = now.AddDate(0, 0, 2)

	default:
		return time.Time{}, 0, false
	}

	// Проверяем, есть ли после даты часть суток.
	if len(parts) >= 2 {
		timeOfDay, ok := parseTimeOfDay(strings.ToLower(parts[1]))

		if ok {
			result = time.Date(
				result.Year(),
				result.Month(),
				result.Day(),
				timeOfDay.Hour(),
				timeOfDay.Minute(),
				0,
				0,
				result.Location(),
			)

			return result, 2, true
		}
	}

	return result, 1, true
}

func parseDateTime(dateText, timeText string) (time.Time, bool) {
	now := time.Now()

	layouts := []string{
		"02.01.06 15:04",
		"02/01/06 15:04",
		"02.01.2006 15:04",
		"02/01/2006 15:04",
	}

	text := dateText + " " + timeText

	for _, layout := range layouts {
		result, err := time.ParseInLocation(
			layout,
			text,
			now.Location(),
		)

		if err == nil {
			return result, true
		}
	}

	return time.Time{}, false
}

func parseDate(text string) (time.Time, bool) {
	now := time.Now()

	layouts := []string{
		"02.01.06",
		"02/01/06",
		"02.01.2006",
		"02/01/2006",
	}

	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(
			layout,
			text,
			now.Location(),
		)

		if err == nil {
			result := time.Date(
				parsed.Year(),
				parsed.Month(),
				parsed.Day(),
				now.Hour(),
				now.Minute(),
				now.Second(),
				0,
				now.Location(),
			)

			return result, true
		}
	}

	return time.Time{}, false
}

var russianNumbers = map[string]int{
	"ноль":         0,
	"один":         1,
	"одна":         1,
	"одно":         1,
	"одну":         1,
	"два":          2,
	"две":          2,
	"три":          3,
	"четыре":       4,
	"пять":         5,
	"шесть":        6,
	"семь":         7,
	"восемь":       8,
	"девять":       9,
	"десять":       10,
	"одиннадцать":  11,
	"двенадцать":   12,
	"тринадцать":   13,
	"четырнадцать": 14,
	"пятнадцать":   15,
	"шестнадцать":  16,
	"семнадцать":   17,
	"восемнадцать": 18,
	"девятнадцать": 19,
	"двадцать":     20,
	"тридцать":     30,
	"сорок":        40,
	"пятьдесят":    50,
}

func parseRussianNumber(text string) (int, bool) {
	value, ok := russianNumbers[text]

	if !ok {
		return 0, false
	}

	return value, true
}

func parseRussianNumberWords(parts []string) (int, int, bool) {
	if len(parts) == 0 {
		return 0, 0, false
	}

	first := strings.ToLower(strings.Trim(parts[0], ".,!?"))

	// Число цифрами: "9", "42"
	if number, err := strconv.Atoi(first); err == nil {
		return number, 1, true
	}

	// Простое число словами: "девять", "двенадцать"
	if number, ok := parseRussianNumber(first); ok {
		return number, 1, true
	}

	// Десятки: "тридцать", "сорок", "пятьдесят"
	tens := map[string]int{
		"двадцать":  20,
		"тридцать":  30,
		"сорок":     40,
		"пятьдесят": 50,
	}

	value, ok := tens[first]
	if !ok {
		return 0, 0, false
	}

	// Например: "тридцать две"
	if len(parts) >= 2 {
		second := strings.ToLower(strings.Trim(parts[1], ".,!?"))

		if number, ok := parseRussianNumber(second); ok {
			return value + number, 2, true
		}
	}

	return value, 1, true
}

/*func parseRussianNumberWords(parts []string) (int, int, bool) {
	if len(parts) == 0 {
		return 0, 0, false
	}

	first := strings.ToLower(
		strings.Trim(parts[0], ".,!?"),
	)

	// Простое число: "двенадцать", "пятнадцать"
	if number, ok := parseRussianNumber(first); ok {
		return number, 1, true
	}

	// Десятки: "тридцать", "сорок", "пятьдесят"
	tens := map[string]int{
		"двадцать":  20,
		"тридцать":  30,
		"сорок":     40,
		"пятьдесят": 50,
	}

	value, ok := tens[first]
	if !ok {
		return 0, 0, false
	}

	// Например: "тридцать две"
	if len(parts) >= 2 {
		second := strings.ToLower(
			strings.Trim(parts[1], ".,!?"),
		)

		if number, ok := parseRussianNumber(second); ok {
			return value + number, 2, true
		}
	}

	return value, 1, true
}*/

var timeOfDay = map[string]string{
	"утром":           "08:00",
	"днём":            "13:00",
	"днем":            "13:00",
	"вечером":         "19:00",
	"ночью":           "23:00",
	"рано утром":      "06:00",
	"поздно вечером":  "22:00",
	"полдень":         "12:00",
	"полночь":         "00:00",
	"обед":            "13:00",
	"после обеда":     "15:00",
	"перед обедом":    "11:30",
	"после завтрака":  "10:00",
	"перед завтраком": "07:30",
	"после ужина":     "20:30",
	"перед ужином":    "19:00",
}

func parseTimeOfDay(text string) (time.Time, bool) {
	timeText, ok := timeOfDay[text]
	if !ok {
		return time.Time{}, false
	}

	now := time.Now()

	parsed, err := time.ParseInLocation(
		"15:04",
		timeText,
		now.Location(),
	)

	if err != nil {
		return time.Time{}, false
	}

	result := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		parsed.Hour(),
		parsed.Minute(),
		0,
		0,
		now.Location(),
	)

	return result, true
}

func normalizeRussianClockTime(text string) string {
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return text
	}

	// Если время записано через "-" или ".",
	// превращаем его в два отдельных слова.
	for i, part := range parts {
		part = strings.TrimSpace(part)

		part = strings.ReplaceAll(part, "-", " ")
		part = strings.ReplaceAll(part, ".", " ")

		parts[i] = part
	}

	// После замены разделителей снова собираем слова.
	text = strings.Join(parts, " ")
	parts = strings.Fields(text)

	if len(parts) < 2 {
		return text
	}

	hour, hourParts, ok := parseRussianNumberWords(parts)
	if !ok || hour < 0 || hour > 23 {
		return text
	}

	minute, minuteParts, ok := parseRussianNumberWords(parts[hourParts:])
	if !ok || minute < 0 || minute > 59 {
		return text
	}

	result := fmt.Sprintf("%02d:%02d", hour, minute)

	remaining := parts[hourParts+minuteParts:]

	if len(remaining) > 0 {
		remaining[0] = strings.TrimLeft(remaining[0], ".,!?")
	}

	if len(remaining) == 0 {
		return result
	}

	return strings.Join(
		append([]string{result}, remaining...),
		" ",
	)
}

func parseRepeat(text string) (string, RepeatType) {
	text = strings.TrimSpace(text)

	lower := strings.ToLower(text)

	repeats := []struct {
		suffix string
		repeat RepeatType
	}{
		{"каждый час", RepeatHour},
		{"каждый день", RepeatDay},
		{"каждую неделю", RepeatWeek},
		{"каждый месяц", RepeatMonth},
		{"каждый год", RepeatYear},
		{"по будням", RepeatWeekday},
		{"в выходные", RepeatWeekend},
		{"на выходных", RepeatWeekend},
	}

	for _, item := range repeats {
		if strings.HasSuffix(lower, item.suffix) {
			text = strings.TrimSpace(
				text[:len(text)-len(item.suffix)],
			)

			return text, item.repeat
		}
	}

	return text, RepeatNone
}

func nextReminderTime(reminder Reminder) (time.Time, bool) {
	switch reminder.Repeat {

	case RepeatHour:
		return reminder.RemindAt.Add(time.Hour), true

	case RepeatDay:
		return reminder.RemindAt.AddDate(0, 0, 1), true

	case RepeatWeek:
		return reminder.RemindAt.AddDate(0, 0, 7), true

	case RepeatMonth:
		return reminder.RemindAt.AddDate(0, 1, 0), true

	case RepeatYear:
		return reminder.RemindAt.AddDate(1, 0, 0), true

	case RepeatWeekday:
		next := reminder.RemindAt.AddDate(0, 0, 1)

		for {
			weekday := next.Weekday()

			if weekday != time.Saturday &&
				weekday != time.Sunday {
				return next, true
			}

			next = next.AddDate(0, 0, 1)
		}

	case RepeatWeekend:
		next := reminder.RemindAt.AddDate(0, 0, 1)

		for {
			weekday := next.Weekday()

			if weekday == time.Saturday ||
				weekday == time.Sunday {
				return next, true
			}

			next = next.AddDate(0, 0, 1)
		}

	default:
		return time.Time{}, false
	}
}

func repeatText(repeat RepeatType) string {
	switch repeat {
	case RepeatHour:
		return "Каждый час"
	case RepeatDay:
		return "Каждый день"
	case RepeatWeek:
		return "Каждую неделю"
	case RepeatMonth:
		return "Каждый месяц"
	case RepeatYear:
		return "Каждый год"
	case RepeatWeekday:
		return "По будням"
	case RepeatWeekend:
		return "В выходные"
	default:
		return ""
	}
}
