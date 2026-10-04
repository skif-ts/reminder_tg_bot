package main

import (
	"log"
	"os"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type RepeatType string

const (
	RepeatNone    RepeatType = ""
	RepeatHour    RepeatType = "hour"
	RepeatDay     RepeatType = "day"
	RepeatWeek    RepeatType = "week"
	RepeatMonth   RepeatType = "month"
	RepeatYear    RepeatType = "year"
	RepeatWeekday RepeatType = "weekdays"
	RepeatWeekend RepeatType = "weekend"
)

type Reminder struct {
	ChatID   int64
	RemindAt time.Time
	Text     string
	Repeat   RepeatType
}

var reminders []Reminder

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, mess string) {
	msg := tgbotapi.NewMessage(
		chatID,
		mess,
	)

	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Ошибка отправки:", err)
	}

}

func reminderPrint(bot *tgbotapi.BotAPI) {
	for i := 0; i < len(reminders); i++ {
		reminder := &reminders[i]

		if time.Now().Before(reminder.RemindAt) {
			continue
		}

		// Отправляем напоминание
		sendMessage(
			bot,
			reminder.ChatID,
			"Напоминание: "+reminder.Text,
		)

		// Одноразовое напоминание
		if reminder.Repeat == RepeatNone {
			reminders = append(
				reminders[:i],
				reminders[i+1:]...,
			)

			return
		}

		// Повторяющееся напоминание
		next, ok := nextReminderTime(*reminder)

		if !ok {
			// На всякий случай удаляем,
			// если повтор неизвестного типа.
			reminders = append(
				reminders[:i],
				reminders[i+1:]...,
			)

			return
		}

		// Переносим текущее напоминание
		// на следующее срабатывание.
		reminder.RemindAt = next

		return
	}
}

func scheduler(bot *tgbotapi.BotAPI) {
	for {
		log.Println("scheduler tick")

		reminderPrint(bot)

		time.Sleep(time.Second)
	}
}

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	/*
		log.Printf("MessageID: %d", update.Message.MessageID)
		log.Printf("Text: %s", update.Message.Text)
		log.Printf("ChatID: %d", update.Message.Chat.ID)
		log.Printf("User: %s", update.Message.From.UserName)
		//log.Printf("%v", aText) // массив
	*/

	if update.Message == nil {
		return
	}

	// получаем просто текст
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// получаем голосовое сообщение
	if update.Message.Voice != nil {
		file, err := bot.GetFile(tgbotapi.FileConfig{
			FileID: update.Message.Voice.FileID,
		})

		if err != nil {
			log.Println("Ошибка получения файла:", err)
			return
		}

		fileURL := file.Link(bot.Token)

		err = downloadFile(fileURL, "voice.oga")
		if err != nil {
			log.Println("Ошибка скачивания:", err)
			return
		}

		log.Println("Голосовое сообщение скачано")

		text, err = processVoice("voice.oga", "voice.wav")
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Распознано:", text)

		//return
	}

	text = normalizeRussianClockTime(text)

	aText := strings.Fields(text)
	if len(aText) == 0 {
		return
	}

	var mess string

	switch aText[0] {

	case "/start":
		mess = "Привет! Я бот-напоминалка."

	case "/help":
		mess = `Доступные команды: 
				/start — запустить бота 
				/help — показать помощь
				/list или /l "лист" — список моих напоминаний
				/remind или /r — напомнить`

	case "/list", "/l", "лист":
		mess = "Список напоминаний:\n"
		var aMess []string

		var userReminders []Reminder

		for _, reminder := range reminders {
			if reminder.ChatID == chatID {
				userReminders = append(userReminders, reminder)
			}
		}

		sort.Slice(userReminders, func(i, j int) bool {
			return userReminders[i].RemindAt.Before(userReminders[j].RemindAt)
		})

		for _, reminder := range userReminders {
			aMess = append(
				aMess,
				reminder.RemindAt.Format("02.01.2006 15:04")+" "+reminder.Text+". "+repeatText(reminder.Repeat),
			)
		}

		if len(aMess) == 0 {
			mess += "Напоминаний пока нет."
		} else {
			mess += strings.Join(aMess, "\n")
		}

	default:
		//mess = "Я получил: " + text

		if len(aText) < 2 {
			mess = "Напиши, когда и что нужно напомнить."
		} else {
			remindAt, timeParts, err := parseRemindTime(aText)

			if err != nil {
				mess = "Не понимаю формат времени. Сообщение: " + text
			} else {
				mess = strings.Join(aText[0+timeParts:], " ")

				reminderText, repeat := parseRepeat(mess)

				reminder := Reminder{
					ChatID:   update.Message.Chat.ID,
					RemindAt: remindAt,
					Text:     reminderText,
					Repeat:   repeat,
				}

				reminders = append(reminders, reminder)

				mess = "Принял, напомню в " + remindAt.Format("02.01.2006 15:04")
			}
		}

	}

	// выводим то, что ввели сейчас
	sendMessage(bot, update.Message.Chat.ID, mess)
}

func main() {
	token := os.Getenv("REMINDER_BOT_TOKEN")

	log.Println("Token exists:", token != "")

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Бот запущен: @%s", bot.Self.UserName)

	now := time.Now()
	later := now.Add(1 * time.Second)
	reminders = []Reminder{
		{410459541, later, "Рестарт", RepeatNone},
	}

	go scheduler(bot)

	config := tgbotapi.NewUpdate(0)
	config.Timeout = 30

	updates := bot.GetUpdatesChan(config)

	for update := range updates {
		handleUpdate(bot, update)
	}
}
