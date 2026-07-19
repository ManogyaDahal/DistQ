package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/client"
)

func main() {
	base := os.Getenv("DISTQ_API_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	c := client.New(base)
	reader := bufio.NewReader(os.Stdin)

	for {
		showMenu()
		i, _ := reader.ReadString('\n')
		i = strings.TrimSpace(i)
		switch i {
		case "1":
			connectionTest(c)
		case "2":
			submitImmediate(c, reader)
		case "3":
			submitPriorityTasks(c)
		case "4":
			submitScheduled(c)
		case "5":
			submitFailure(c)
		case "6":
			monitorTask(c, reader)
		case "7":
			showMetrics(c)
		case "8":
			showWorkers(c)
		case "9":
			stressTest(c)
		case "10":
			runCompleteDemo(c)
		case "0":
			fmt.Println("Goodbye")
			return
		default:
			fmt.Println("Unknown option")
		}
		fmt.Println()
	}
}

func showMenu() {
	fmt.Println("=====================================")
	fmt.Println("DistQ Client SDK Demonstration")
	fmt.Println("=====================================")
	fmt.Println("1. Connection Test")
	fmt.Println("2. Submit Immediate Task")
	fmt.Println("3. Submit Priority Tasks")
	fmt.Println("4. Submit Scheduled Task")
	fmt.Println("5. Submit Failure Task")
	fmt.Println("6. Monitor Task Status")
	fmt.Println("7. Show Metrics")
	fmt.Println("8. Show Workers")
	fmt.Println("9. Stress Test")
	fmt.Println("10. Run Complete Demo")
	fmt.Println("0. Exit")
	fmt.Print("> ")
}

func connectionTest(c *client.Client) {
	ctx := context.Background()
	m, err := c.GetMetrics(ctx)
	if err != nil {
		fmt.Println("\u001b[31mConnection test failed:\u001b[0m", err)
		return
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	fmt.Println("\u001b[32mConnection OK\u001b[0m")
	fmt.Println(string(b))
}

func submitImmediate(c *client.Client, r *bufio.Reader) {
	fmt.Print("Task type (demo.sleep): ")
	typeStr, _ := r.ReadString('\n')
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		typeStr = "demo.sleep"
	}

	fmt.Print("Priority (5): ")
	pstr, _ := r.ReadString('\n')
	pstr = strings.TrimSpace(pstr)
	p := 5
	if pstr != "" {
		if v, err := strconv.Atoi(pstr); err == nil {
			p = v
		}
	}

	payload := map[string]any{"ts": time.Now().Unix()}
	data, _ := json.Marshal(payload)
	ctx := context.Background()
	res, err := c.SubmitTask(ctx, client.SubmitTaskRequest{Type: typeStr, Payload: data, Priority: p, Source: "Go SDK"})
	if err != nil {
		fmt.Println("\u001b[31mSubmit failed:\u001b[0m", err)
		return
	}
	fmt.Println("\u001b[32mSubmitted\u001b[0m ID:", res.ID)
}

func submitPriorityTasks(c *client.Client) {
	ctx := context.Background()
	fmt.Println("Submitting priority groups: 10 high, 20 normal, 10 low...")
	for i := 0; i < 10; i++ {
		c.SubmitTask(ctx, client.SubmitTaskRequest{Type: "demo.sleep", Payload: json.RawMessage(`{"i":` + strconv.Itoa(i) + `}`), Priority: 10, Source: "Go SDK"})
	}
	for i := 0; i < 20; i++ {
		c.SubmitTask(ctx, client.SubmitTaskRequest{Type: "demo.sleep", Payload: json.RawMessage(`{"i":` + strconv.Itoa(i) + `}`), Priority: 5, Source: "Go SDK"})
	}
	for i := 0; i < 10; i++ {
		c.SubmitTask(ctx, client.SubmitTaskRequest{Type: "demo.sleep", Payload: json.RawMessage(`{"i":` + strconv.Itoa(i) + `}`), Priority: 1, Source: "Go SDK"})
	}
	fmt.Println("\u001b[32mPriority tasks submitted\u001b[0m")
}

func submitScheduled(c *client.Client) {
	ctx := context.Background()
	now := time.Now()
	eta := now.Add(30 * time.Second)
	payload, _ := json.Marshal(map[string]any{"note": "scheduled-30"})
	_, err := c.SubmitTask(ctx, client.SubmitTaskRequest{Type: "demo.sleep", Payload: payload, Priority: 5, ETA: &eta, Source: "Go SDK"})
	if err != nil {
		fmt.Println("\u001b[31mScheduled submit failed:\u001b[0m", err)
		return
	}
	eta2 := now.Add(60 * time.Second)
	payload2, _ := json.Marshal(map[string]any{"note": "scheduled-60"})
	_, _ = c.SubmitTask(ctx, client.SubmitTaskRequest{Type: "demo.sleep", Payload: payload2, Priority: 5, ETA: &eta2, Source: "Go SDK"})
	fmt.Println("\u001b[32mScheduled tasks submitted\u001b[0m")
}

func submitFailure(c *client.Client) {
	ctx := context.Background()
	_, err := c.SubmitTask(ctx, client.SubmitTaskRequest{Type: "demo.fail", Payload: json.RawMessage(`{"fail":true}`), Priority: 5, Source: "Go SDK"})
	if err != nil {
		fmt.Println("\u001b[31mSubmit failed:\u001b[0m", err)
		return
	}
	fmt.Println("\u001b[32mFailure task submitted\u001b[0m")
}

func monitorTask(c *client.Client, r *bufio.Reader) {
	fmt.Print("Task ID to monitor: ")
	id, _ := r.ReadString('\n')
	id = strings.TrimSpace(id)
	if id == "" {
		fmt.Println("No ID provided")
		return
	}
	ctx := context.Background()
	for {
		st, err := c.GetTask(ctx, id)
		if err != nil {
			fmt.Println("\u001b[31mGetTask failed:\u001b[0m", err)
			return
		}
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Println(string(b))
		if st.Status == "done" || st.Status == "failed" || st.Status == "dead" {
			break
		}
		time.Sleep(2 * time.Second)
	}
}

func showMetrics(c *client.Client) {
	ctx := context.Background()
	m, err := c.GetMetrics(ctx)
	if err != nil {
		fmt.Println("\u001b[31mMetrics failed:\u001b[0m", err)
		return
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	fmt.Println(string(b))
}

func showWorkers(c *client.Client) {
	ctx := context.Background()
	w, err := c.GetWorkers(ctx)
	if err != nil {
		fmt.Println("\u001b[31mWorkers failed:\u001b[0m", err)
		return
	}
	b, _ := json.MarshalIndent(w, "", "  ")
	fmt.Println(string(b))
}

func stressTest(c *client.Client) {
	ctx := context.Background()
	fmt.Println("Submitting 100 mixed tasks...")
	for i := 0; i < 100; i++ {
		p := 5
		if i%3 == 0 {
			p = 10
		} else if i%3 == 2 {
			p = 1
		}
		t := "demo.sleep"
		if i%15 == 0 {
			t = "demo.fail"
		}
		_, _ = c.SubmitTask(ctx, client.SubmitTaskRequest{Type: t, Payload: json.RawMessage(`{"i":` + strconv.Itoa(i) + `}`), Priority: p, Source: "Go SDK"})
	}
	fmt.Println("\u001b[32mStress test submitted\u001b[0m")
}

func runCompleteDemo(c *client.Client) {
	fmt.Println("Running full demo: submit variety, monitor one, show metrics")
	ctx := context.Background()
	// submit a few tasks
	res, _ := c.SubmitTask(ctx, client.SubmitTaskRequest{Type: "demo.sleep", Payload: json.RawMessage(`{"demo":true}`), Priority: 5, Source: "Go SDK"})
	if res != nil {
		id := res.ID
		fmt.Println("Monitoring task", id)
		for {
			st, err := c.GetTask(ctx, id)
			if err != nil {
				fmt.Println("GetTask error:", err)
				break
			}
			fmt.Println("status:", st.Status)
			if st.Status == "done" || st.Status == "failed" {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	showMetrics(c)
	showWorkers(c)
}
