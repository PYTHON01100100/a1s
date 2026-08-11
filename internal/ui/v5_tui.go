package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// autoDetectAI follows the same principle as aliyun-cli profile detection:
// explicit environment configuration wins; otherwise a reachable local Ollama
// is selected automatically. No cloud credentials are copied into a1s.
func (a *App) autoDetectAI(ctx context.Context) {
	if a.ai == nil {
		return
	}
	if a.ai.Enabled() {
		a.aiProvider = "openai-compatible"
		return
	}
	models, err := a.ollamaModels(ctx)
	if err != nil || len(models) == 0 {
		return
	}
	a.aiProvider = "ollama"
	a.ollamaModel = models[0]
	a.ai.BaseURL = strings.TrimRight(a.ollamaURL, "/") + "/v1"
	a.ai.APIKey = "ollama"
	a.ai.Model = models[0]
}

func (a *App) showAIProviders(ctx context.Context) error {
	fmt.Println()
	fmt.Println(orange + bold + " AI PROVIDERS" + reset)

	ollamaState := red + "not found" + reset
	if _, err := exec.LookPath("ollama"); err == nil {
		ollamaState = yellow + "installed / stopped" + reset
		if models, err := a.ollamaModels(ctx); err == nil {
			ollamaState = green + fmt.Sprintf("running • %d model(s)", len(models)) + reset
		}
	}
	envState := yellow + "not configured" + reset
	if strings.TrimSpace(a.cfg.AIBaseURL) != "" && strings.TrimSpace(a.cfg.AIModel) != "" {
		envState = green + "configured" + reset
	}

	active := "none"
	if a.aiProvider != "" {
		active = a.aiProvider
		if a.ai != nil && a.ai.Model != "" {
			active += " / " + a.ai.Model
		}
	}

	fmt.Printf("  %-20s %s\n", "ollama", ollamaState)
	fmt.Printf("  %-20s %s\n", "openai-compatible", envState)
	fmt.Printf("  %s%-20s%s %s%s%s\n", gray, "active", reset, orange, active, reset)
	fmt.Println(dim + "  Commands: ai use ollama • ai models • ai model 1 • chat" + reset)
	return nil
}

// dashboard is a compact, K9s-inspired status surface. The detailed resource
// view still lives in `ecs`; this gives users context before they start typing.
func (a *App) dashboard(ctx context.Context) {
	aiText := "not configured"
	aiColor := yellow
	if a.aiProvider != "" {
		aiText = a.aiProvider
		if a.ai != nil && a.ai.Model != "" {
			aiText += " / " + a.ai.Model
		}
		aiColor = green
	}
	selected := "none"
	if a.selectedID != "" {
		selected = blank(a.selectedName, a.selectedID)
	}

	fmt.Println()
	fmt.Println(dim + " ┌─ CONTEXT ───────────────────────────────┬─ AI ───────────────────────────────────┐" + reset)
	fmt.Printf(" %s│%s profile  %-28s %s│%s provider  %s%-29s%s %s│%s\n", dim, reset, clip(blank(a.cfg.Profile, "default"), 28), dim, reset, aiColor, clip(aiText, 29), reset, dim, reset)
	fmt.Printf(" %s│%s region   %-28s %s│%s selected  %-29s %s│%s\n", dim, reset, clip(blank(a.cfg.Region, "auto"), 28), dim, reset, clip(selected, 29), dim, reset)
	fmt.Println(dim + " └─────────────────────────────────────────┴───────────────────────────────────────┘" + reset)
	fmt.Printf(" %s:%s %secs%s  %s:%s %srun%s  %s:%s %smetrics%s  %s:%s %sbill%s  %s:%s %schat%s  %s:%s %shelp%s\n",
		orange, reset, cyan, reset,
		orange, reset, cyan, reset,
		orange, reset, cyan, reset,
		orange, reset, cyan, reset,
		orange, reset, cyan, reset,
		orange, reset, cyan, reset)
	_ = ctx
}

func (a *App) chat(ctx context.Context, args []string) error {
	a.autoDetectAI(ctx)
	if a.ai == nil || !a.ai.Enabled() {
		fmt.Println()
		fmt.Println(yellow + bold + " CHAT NEEDS AN AI PROVIDER" + reset)
		_ = a.showAIProviders(ctx)
		fmt.Println(dim + "  If Ollama is installed, start it with `ollama serve`; a1s will auto-detect it on the next `chat`." + reset)
		return nil
	}

	fmt.Println()
	fmt.Printf("%s%sA1S CHAT%s  %s%s • %s%s\n", orange, bold, reset, dim, a.aiProvider, a.ai.Model, reset)
	fmt.Println(dim + "  Plain text = ask AI • /run <ecs> <cmd> • /ecs • /use <ecs> • /metrics • /doctor" + reset)
	fmt.Println(dim + "  /providers • /models • /model <n|name> • /clear • /help • /exit" + reset)

	for {
		prompt := fmt.Sprintf("%schat%s %s[%s]%s › ", orange, reset, dim, clip(a.ai.Model, 22), reset)
		line, err := a.editor.ReadLine(prompt, a.chatCompletions)
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			done, err := a.handleChatSlash(ctx, line)
			if err != nil {
				fmt.Println(red + "ERROR " + reset + err.Error())
			}
			if done {
				return nil
			}
			continue
		}
		if err := a.chatAsk(ctx, line); err != nil {
			fmt.Println(red + "ERROR " + reset + err.Error())
		}
	}
}

func (a *App) handleChatSlash(ctx context.Context, line string) (bool, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(line, "/"))
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return false, nil
	}
	cmd := strings.ToLower(parts[0])
	rest := strings.TrimSpace(strings.TrimPrefix(raw, parts[0]))
	switch cmd {
	case "exit", "quit", "q":
		return true, nil
	case "help", "?":
		fmt.Println(dim + "  /run <ecs> <cmd> executes through Cloud Assistant; all cloud actions are explicit slash commands." + reset)
		fmt.Println(dim + "  /ecs /use /metrics /doctor /providers /models /model /clear /exit" + reset)
		return false, nil
	case "clear":
		a.chatHistory = nil
		fmt.Println(green + "✓ chat history cleared" + reset)
		return false, nil
	case "providers":
		return false, a.showAIProviders(ctx)
	case "models":
		return false, a.handleAI(ctx, []string{"models"}, "models")
	case "model":
		if len(parts) < 2 {
			return false, fmt.Errorf("usage: /model <number|name>")
		}
		return false, a.handleAI(ctx, []string{"model", parts[1]}, "model "+parts[1])
	case "ecs", "ls":
		return false, a.ecs(ctx)
	case "use":
		if len(parts) < 2 {
			return false, fmt.Errorf("usage: /use <ecs-name|row|instance-id>")
		}
		return false, a.selectInstance(parts[1])
	case "run":
		id, shell, err := a.resolveRunArgs(rest)
		if err != nil {
			return false, err
		}
		return false, a.runCmd(ctx, id, shell)
	case "metrics":
		id, mins, err := a.resolveMetricsArgs(parts[1:])
		if err != nil {
			return false, err
		}
		return false, a.metrics(ctx, id, mins)
	case "doctor":
		id := a.selectedID
		if len(parts) > 1 {
			if resolved, ok := a.resolveInstanceRef(parts[1]); ok {
				id = resolved
			} else {
				id = parts[1]
			}
		}
		return false, a.doctor(ctx, id)
	default:
		return false, fmt.Errorf("unknown chat command /%s; use /help", cmd)
	}
}

func (a *App) chatAsk(ctx context.Context, question string) error {
	var cloudContext string
	if !(a.cfg.Demo && !a.cfg.SampleData) {
		xs, err := a.cloud.ListInstances(ctx)
		if err == nil {
			cloudContext = fmt.Sprintf("Current Alibaba Cloud context: region=%s profile=%s selected=%s instances=%v", a.cfg.Region, a.cfg.Profile, a.selectedName, xs)
		}
	}

	historyStart := 0
	if len(a.chatHistory) > 8 {
		historyStart = len(a.chatHistory) - 8
	}
	var history strings.Builder
	for _, turn := range a.chatHistory[historyStart:] {
		history.WriteString(turn.Role)
		history.WriteString(": ")
		history.WriteString(turn.Content)
		history.WriteString("\n")
	}

	prompt := strings.TrimSpace(cloudContext + "\n\nConversation:\n" + history.String() + "\nUser: " + question)
	system := "You are the a1s Alibaba Cloud operations copilot. Be concise and practical. You may explain commands and recommend actions. Never claim you executed a cloud action unless its output is present. When an action is needed, tell the user the exact a1s chat slash command, such as /run <ecs> <cmd>, /metrics <ecs>, or /doctor."
	ans, err := a.ai.Ask(ctx, system, prompt)
	if err != nil {
		return err
	}
	a.chatHistory = append(a.chatHistory, chatTurn{Role: "user", Content: question}, chatTurn{Role: "assistant", Content: ans})
	fmt.Printf("\n%s%sAI%s %s%s%s\n%s\n\n", orange, bold, reset, dim, a.ai.Model, reset, ans)
	return nil
}

func (a *App) chatCompletions(line string) []string {
	base := []string{"/help", "/exit", "/providers", "/models", "/model ", "/ecs", "/use ", "/run ", "/metrics ", "/doctor", "/clear"}
	trim := strings.TrimSpace(line)
	if strings.HasPrefix(trim, "/run ") || strings.HasPrefix(trim, "/use ") || strings.HasPrefix(trim, "/metrics ") {
		parts := strings.Fields(line)
		if len(parts) <= 2 {
			prefix := ""
			if len(parts) > 1 {
				prefix = parts[1]
			}
			cmd := parts[0] + " "
			var out []string
			for _, x := range a.instances {
				if strings.HasPrefix(strings.ToLower(x.Name), strings.ToLower(prefix)) {
					suffix := ""
					if parts[0] == "/run" {
						suffix = " "
					}
					out = append(out, cmd+x.Name+suffix)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	var out []string
	for _, c := range base {
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(trim)) {
			out = append(out, c)
		}
	}
	return out
}
