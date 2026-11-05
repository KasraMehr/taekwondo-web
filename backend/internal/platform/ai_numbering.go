package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"backend/internal/modules/tournaments"
)

type aiNumberingPlan struct {
	Priority []string `json:"priority"`
	Reason   string   `json:"reason"`
}

func suggestNumbering(ctx context.Context, t *tournaments.Tournament, selectedCourts map[string][]int, hoguPairs map[string]int) (aiNumberingPlan, error) {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if key == "" {
		return aiNumberingPlan{}, httpError{503, "OPENAI_API_KEY is not configured on the backend"}
	}
	counts := map[string]int{}
	for _, m := range t.Matches {
		if m.IsBye == nil || !*m.IsBye {
			counts[m.WeightCategory]++
		}
	}
	categories := make([]string, 0, len(counts))
	for category := range counts {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	type categoryInput struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		MatchCount     int    `json:"matchCount"`
		SelectedCourts []int  `json:"selectedCourts"`
	}
	categoryInputs := make([]categoryInput, 0, len(categories))
	for index, category := range categories {
		categoryInputs = append(categoryInputs, categoryInput{index + 1, category, counts[category], selectedCourts[category]})
	}
	data, _ := json.Marshal(struct {
		Categories []categoryInput `json:"categories"`
		Courts     int             `json:"courts"`
		HoguPairs  map[string]int  `json:"hoguPairs,omitempty"`
	}{categoryInputs, t.Courts, hoguPairs})
	requestBody, _ := json.Marshal(map[string]any{
		"model":           modelName(),
		"temperature":     0,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": "You propose the processing order of taekwondo weight categories. Each input category has a unique integer id. Return one JSON object only, in this exact shape: {\"priority\":[2,1,3],\"reason\":\"یک توضیح کوتاه فارسی\"}. priority must contain every input id exactly once, as JSON integers (not category names or strings). Use only ids present in the input; never invent an id. Keep selectedCourts unchanged. Prefer an order that can keep the selected courts busy, considering matchCount. The server, not you, enforces bracket dependencies, rest, hogu capacity and final match numbers. If uncertain, return all ids in the order received. No markdown, code fences, additional fields or prose outside JSON."},
			{"role": "user", "content": "Arrange these category ids. Copy each id exactly once into priority. Input: " + string(data)},
		},
	})
	callCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, numberingBaseURL()+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return aiNumberingPlan{}, httpError{502, "آدرس سرویس هوش مصنوعی معتبر نیست؛ AI_NUMBERING_BASE_URL را بررسی کنید."}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 36 * time.Second}).Do(req)
	if err != nil {
		return aiNumberingPlan{}, httpError{502, "اتصال بک‌اند به سرویس هوش مصنوعی برقرار نشد؛ اتصال اینترنت سرور را بررسی کنید."}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return aiNumberingPlan{}, httpError{502, "خواندن پاسخ سرویس هوش مصنوعی ناموفق بود."}
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return aiNumberingPlan{}, httpError{502, "کلید API توسط سرویس هوش مصنوعی پذیرفته نشد. مقدار کلید و آدرس API را در backend/.env بررسی کنید."}
		}
		return aiNumberingPlan{}, httpError{502, fmt.Sprintf("AI service returned HTTP %d", resp.StatusCode)}
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Choices) == 0 {
		return aiNumberingPlan{}, httpError{502, "AI service returned an invalid response"}
	}
	var modelPlan struct {
		Priority json.RawMessage `json:"priority"`
		Reason   string          `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &modelPlan); err != nil {
		return aiNumberingPlan{}, httpError{502, "AI service returned invalid JSON"}
	}
	plan := normalizeAIIDPlan(modelPlan.Priority, modelPlan.Reason, categories)
	if reasonRunes := []rune(plan.Reason); len(reasonRunes) > 300 {
		plan.Reason = string(reasonRunes[:300])
	}
	return plan, nil
}

// IDs avoid asking the model to copy category names with exact Unicode spelling.
func normalizeAIIDPlan(raw json.RawMessage, reason string, categories []string) aiNumberingPlan {
	var proposed []json.RawMessage
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return normalizeAIPlan(aiNumberingPlan{Reason: reason}, categories)
	}
	priority := make([]string, 0, len(proposed))
	for _, item := range proposed {
		var id int
		if json.Unmarshal(item, &id) == nil && id >= 1 && id <= len(categories) {
			priority = append(priority, categories[id-1])
			continue
		}
		var value string
		if json.Unmarshal(item, &value) == nil {
			if parsed, err := strconv.Atoi(value); err == nil && parsed >= 1 && parsed <= len(categories) {
				priority = append(priority, categories[parsed-1])
			} else {
				priority = append(priority, value)
			}
		}
	}
	return normalizeAIPlan(aiNumberingPlan{Priority: priority, Reason: reason}, categories)
}

// The model only proposes an order. The tournament's category list remains authoritative.
func normalizeAIPlan(plan aiNumberingPlan, categories []string) aiNumberingPlan {
	allowed := make(map[string]string, len(categories))
	for _, category := range categories {
		allowed[normalizeCategoryName(category)] = category
	}
	seen := make(map[string]bool, len(categories))
	priority := make([]string, 0, len(categories))
	for _, proposed := range plan.Priority {
		category, ok := allowed[normalizeCategoryName(proposed)]
		if ok && !seen[category] {
			priority = append(priority, category)
			seen[category] = true
		}
	}
	if len(priority) != len(categories) {
		for _, category := range categories {
			if !seen[category] {
				priority = append(priority, category)
			}
		}
		plan.Reason = "اولویت وزن‌ها با اطلاعات واقعی مسابقه تکمیل شد و برنامه پیش از ثبت قابل بازبینی است."
	}
	plan.Priority = priority
	return plan
}

func normalizeCategoryName(category string) string {
	category = strings.TrimSpace(category)
	category = strings.TrimSpace(strings.TrimPrefix(category, "وزن"))
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '۰' && r <= '۹':
			return '0' + r - '۰'
		case r >= '٠' && r <= '٩':
			return '0' + r - '٠'
		case r == '−' || r == '–' || r == '—' || r == '‑':
			return '-'
		case r == '＋':
			return '+'
		}
		return r
	}, category)
}

func numberingBaseURL() string {
	if base := strings.TrimSpace(os.Getenv("AI_NUMBERING_BASE_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}
	return "https://api.gapgpt.app/v1"
}

func modelName() string {
	if model := strings.TrimSpace(os.Getenv("OPENAI_NUMBERING_MODEL")); model != "" {
		return model
	}
	return "gpt-4o-mini"
}
