package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/models"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
)

func Banner() {
	fmt.Println()
	fmt.Println(Bold + Cyan)
	fmt.Println("DistQ SDK Test Suite")
	fmt.Println(Reset)
}

func Header(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println(Bold + Cyan + title + Reset)
	fmt.Println(strings.Repeat("=", 70))
}

func Divider() {
	fmt.Println(strings.Repeat("-", 70))
}

func Success(msg string) {
	fmt.Println(Green + "✔ " + msg + Reset)
}

func Failure(msg string) {
	fmt.Println(Red + "✘ " + msg + Reset)
}

func Info(msg string) {
	fmt.Println(Blue + msg + Reset)
}

func Warn(msg string) {
	fmt.Println(Yellow + msg + Reset)
}

func PrintTask(task *models.Task) {
	fmt.Println()
	fmt.Println(Bold + "Task Information" + Reset)
	fmt.Printf("ID        : %s\n", task.ID)
	fmt.Printf("Type      : %s\n", task.Type)
	fmt.Printf("Status    : %s\n", task.Status)
	fmt.Printf("Priority  : %d\n", task.Priority)
	fmt.Printf("Retries   : %d\n", task.RetryCount)

	if task.ETA != nil {
		fmt.Printf("ETA       : %s\n", task.ETA.Format(time.RFC3339))
	}

	fmt.Printf("Created   : %s\n", task.CreatedAt.Format(time.RFC3339))
}

func WaitForCompletion(ctx context.Context, sdk *redisclient.SDKClient, id string) {
	fmt.Println()
	fmt.Println("Watching task...")

	for {
		task, err := sdk.Status(ctx, id)
		if err != nil {
			Failure(err.Error())
			return
		}

		fmt.Printf("\rCurrent Status : %-12s", task.Status)

		switch task.Status {
		case models.StatusSuccess,
			models.StatusFailed,
			models.StatusDead:
			fmt.Println()
			PrintTask(task)
			return
		}

		time.Sleep(time.Second)
	}
}

func PollStatus(ctx context.Context, sdk *redisclient.SDKClient, id string, count int) {
	for i := 0; i < count; i++ {
		task, err := sdk.Status(ctx, id)
		if err != nil {
			Failure(err.Error())
			return
		}

		fmt.Printf("[%d/%d] Status : %s\n", i+1, count, task.Status)
		time.Sleep(time.Second)
	}
}

func Pause(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
}
