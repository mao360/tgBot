package main

// сюда писать код

import (
	"context"
	"fmt"
	tgbotapi "github.com/skinass/telegram-bot-api/v5"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var (
	// @BotFather в телеграме даст вам это
	BotToken = "test_token" // мой

	// Урл выдаст вам нгрок или хероку
	WebhookURL = "test_hook"
)

var counter int64

type Task struct {
	Creator  *tgbotapi.User
	Executor *tgbotapi.User
	TaskName string
}

func startTaskBot(ctx context.Context) error {

	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Fatalf("NewBotAPI failed: %s", err)
	}
	bot.Debug = true

	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Fatalf("NewWebhook failed: %s", err)
	}

	if _, err = bot.Request(wh); err != nil {
		log.Fatalf("SetWebhook failed: %s", err)
	}

	upd := bot.ListenForWebhook("/")

	port := "8080"

	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":"+port, nil))
	}()

	currTasks := make(map[int64]*Task)

	for data := range upd {
		command := data.Message.Text
		user := data.Message.From
		mapResponse := Logic(command, user, currTasks)
		for key, val := range mapResponse {
			bot.Send(tgbotapi.NewMessage(key, val)) //nolint:errcheck
		}
	}
	return nil
}

func Tasks(user *tgbotapi.User, currTasks map[int64]*Task, retter map[int64]string) {
	if len(currTasks) == 0 {
		retter[user.ID] = "Нет задач"
	} else {
		resString := ""
		for idx, val := range currTasks {
			resString += fmt.Sprintf("%d. %s by @%s\n", idx, val.TaskName, val.Creator.UserName)
			if val.Executor == nil {
				resString += fmt.Sprintf("/assign_%d\n", idx)
			} else {
				if user.ID == val.Executor.ID {
					resString += fmt.Sprintf("assignee: я\n/unassign_%d /resolve_%d\n", idx, idx)
				} else {
					resString += fmt.Sprintf("assignee: @%s\n", val.Executor.UserName)
				}
			}
			resString += "\n"
		}
		resString = strings.TrimSuffix(resString, "\n")
		resString = strings.TrimSuffix(resString, "\n")
		retter[user.ID] = resString
	}
}

func New(command string, user *tgbotapi.User, currTasks map[int64]*Task, retter map[int64]string) {
	counter++
	fragmentedCommand := strings.Split(command, " ")
	taskName := ""
	for i := 1; i < len(fragmentedCommand); i++ {
		taskName += fragmentedCommand[i] + " "
	}
	taskName = strings.TrimSuffix(taskName, " ")

	task := &Task{
		Creator:  user,
		Executor: nil,
		TaskName: taskName,
	}
	currTasks[counter] = task
	retter[user.ID] = fmt.Sprintf(`Задача "%s" создана, id=%d`, taskName, counter)
}
func Assign(command string, user *tgbotapi.User, currTasks map[int64]*Task, retter map[int64]string) {
	fragmentedCommand := strings.Split(command, "_")
	taskID, _ := strconv.Atoi(fragmentedCommand[1]) //nolint:errcheck
	taskID64 := int64(taskID)

	switch currTasks[taskID64].Executor {
	case nil:
		if creatorID := currTasks[taskID64].Creator.ID; creatorID != user.ID {
			retter[creatorID] = fmt.Sprintf(`Задача "%s" назначена на @%s`, currTasks[taskID64].TaskName, user.UserName)
		}
		currTasks[taskID64].Executor = user

		retter[user.ID] = fmt.Sprintf(`Задача "%s" назначена на вас`, currTasks[taskID64].TaskName)
	default:
		prevExecutor := currTasks[taskID64].Executor
		currTasks[taskID64].Executor = user

		retter[user.ID] = fmt.Sprintf(`Задача "%s" назначена на вас`, currTasks[taskID64].TaskName)
		retter[prevExecutor.ID] = fmt.Sprintf(`Задача "%s" назначена на @%s`, currTasks[taskID64].TaskName, user.UserName)
	}
}
func UnAssign(command string, user *tgbotapi.User, currTasks map[int64]*Task, retter map[int64]string) {
	fragmentedCommand := strings.Split(command, "_")
	taskID, _ := strconv.Atoi(fragmentedCommand[1]) //nolint:errcheck
	taskID64 := int64(taskID)
	if *user == *currTasks[taskID64].Executor {
		currTasks[taskID64].Executor = nil

		retter[user.ID] = "Принято"
		retter[currTasks[taskID64].Creator.ID] = fmt.Sprintf(`Задача "%s" осталась без исполнителя`, currTasks[taskID64].TaskName)
	} else {
		retter[user.ID] = "Задача не на вас"
	}
}
func Resolve(command string, user *tgbotapi.User, currTasks map[int64]*Task, retter map[int64]string) {
	fragmentedCommand := strings.Split(command, "_")
	taskID, _ := strconv.Atoi(fragmentedCommand[1]) //nolint:errcheck
	taskID64 := int64(taskID)
	retter[user.ID] = fmt.Sprintf(`Задача "%s" выполнена`, currTasks[taskID64].TaskName)
	retter[currTasks[taskID64].Creator.ID] = fmt.Sprintf(`Задача "%s" выполнена @%s`, currTasks[taskID64].TaskName, user.UserName)
	delete(currTasks, taskID64)
}
func My(user *tgbotapi.User, currTasks map[int64]*Task, retter map[int64]string) {
	resString := ""
	for key, val := range currTasks {
		if val.Executor != nil && val.Executor.ID == user.ID {
			resString += fmt.Sprintf("%d. %s by @%s\n/unassign_%d /resolve_%d\n", key, val.TaskName, val.Creator.UserName, key, key)
		}
	}
	retter[user.ID] = strings.TrimSuffix(resString, "\n")
}
func Owner(user *tgbotapi.User, currTasks map[int64]*Task, retter map[int64]string) {
	resString := ""
	for key, val := range currTasks {
		if val.Creator != nil && val.Creator.ID == user.ID {
			resString += fmt.Sprintf("%d. %s by @%s\n/assign_%d\n", key, val.TaskName, val.Creator.UserName, key)
		}
	}
	retter[user.ID] = strings.TrimSuffix(resString, "\n")
}

func Logic(command string, user *tgbotapi.User, currTasks map[int64]*Task) map[int64]string {
	ret := make(map[int64]string)
	switch {

	case strings.HasPrefix(command, "/tasks"):
		Tasks(user, currTasks, ret)

	case strings.HasPrefix(command, "/new"):
		New(command, user, currTasks, ret)

	case strings.HasPrefix(command, "/assign"):
		Assign(command, user, currTasks, ret)

	case strings.HasPrefix(command, "/unassign"):
		UnAssign(command, user, currTasks, ret)

	case strings.HasPrefix(command, "/resolve"):
		Resolve(command, user, currTasks, ret)

	case strings.HasPrefix(command, "/my"):
		My(user, currTasks, ret)

	case strings.HasPrefix(command, "/owner"):
		Owner(user, currTasks, ret)

	}

	return ret
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
