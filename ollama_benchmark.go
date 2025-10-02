package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Ollama API request/response structures
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

// Benchmark test case
type TestCase struct {
	Name        string
	Prompt      string
	Category    string
	ExpectedLen int
}

// Benchmark result
type BenchmarkResult struct {
	ModelName           string
	TestName            string
	Category            string
	TokensPerSecond     float64
	TimeToFirstToken    float64 // milliseconds
	TotalTokens         int
	PromptTokens        int
	TotalTimeMs         float64
	Response            string
	Success             bool
	Error               string
}

// Model comparison summary
type ModelComparison struct {
	ModelName       string
	AvgTokensPerSec float64
	AvgTotalTimeMs  float64
	TestResults     []BenchmarkResult
}

func main() {
	fmt.Println("=== Ollama LLM Benchmark Tool ===\n")

	// Check if Ollama is running
	if !checkOllamaRunning() {
		fmt.Println("Error: Ollama is not running. Please start Ollama first.")
		fmt.Println("Run: ollama serve")
		return
	}

	// Define models to test
	models := []string{
		"llama3.2:1b",
		"llama3.2:3b",
		"gemma2:2b",
		"qwen2.5:0.5b",
	}

	// Define test cases
	testCases := []TestCase{
		{
			Name:     "Simple Reasoning",
			Category: "reasoning",
			Prompt:   "Explain the concept of recursion in programming in one paragraph.",
		},
		{
			Name:     "Code Generation",
			Category: "coding",
			Prompt:   "Write a Python function to calculate the factorial of a number using recursion.",
		},
		{
			Name:     "Mathematical Problem",
			Category: "math",
			Prompt:   "If a train travels at 60 mph for 2.5 hours, how far does it travel? Show your work.",
		},
		{
			Name:     "Creative Writing",
			Category: "creative",
			Prompt:   "Write a short haiku about artificial intelligence.",
		},
		{
			Name:     "Question Answering",
			Category: "qa",
			Prompt:   "What is the capital of France and what is it famous for?",
		},
	}

	// Run benchmarks
	var comparisons []ModelComparison

	for _, model := range models {
		fmt.Printf("\n=== Testing Model: %s ===\n", model)

		// Check if model is available
		if !checkModelAvailable(model) {
			fmt.Printf("Model %s not found. Pulling model...\n", model)
			if !pullModel(model) {
				fmt.Printf("Failed to pull model %s. Skipping...\n\n", model)
				continue
			}
		}

		var results []BenchmarkResult
		var totalTPS float64
		var totalTime float64
		successCount := 0

		for _, test := range testCases {
			fmt.Printf("\n  Running test: %s (%s)\n", test.Name, test.Category)
			result := runBenchmark(model, test)
			results = append(results, result)

			if result.Success {
				totalTPS += result.TokensPerSecond
				totalTime += result.TotalTimeMs
				successCount++
				fmt.Printf("    ✓ Tokens/sec: %.2f | Total time: %.2fms | Tokens: %d\n",
					result.TokensPerSecond, result.TotalTimeMs, result.TotalTokens)
			} else {
				fmt.Printf("    ✗ Error: %s\n", result.Error)
			}
		}

		avgTPS := 0.0
		avgTime := 0.0
		if successCount > 0 {
			avgTPS = totalTPS / float64(successCount)
			avgTime = totalTime / float64(successCount)
		}

		comparisons = append(comparisons, ModelComparison{
			ModelName:       model,
			AvgTokensPerSec: avgTPS,
			AvgTotalTimeMs:  avgTime,
			TestResults:     results,
		})
	}

	// Display comparison
	fmt.Println("\n\n=== Model Comparison Summary ===\n")
	displayComparison(comparisons)
}

func checkOllamaRunning() bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func checkModelAvailable(model string) bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	for _, m := range result.Models {
		if m.Name == model {
			return true
		}
	}
	return false
}

func pullModel(model string) bool {
	reqBody := map[string]string{"name": model}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://localhost:11434/api/pull",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Wait for pull to complete
	decoder := json.NewDecoder(resp.Body)
	for {
		var status map[string]interface{}
		if err := decoder.Decode(&status); err != nil {
			break
		}
		if status["status"] == "success" {
			return true
		}
	}
	return false
}

func runBenchmark(model string, test TestCase) BenchmarkResult {
	result := BenchmarkResult{
		ModelName: model,
		TestName:  test.Name,
		Category:  test.Category,
	}

	reqData := GenerateRequest{
		Model:  model,
		Prompt: test.Prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to marshal request: %v", err)
		return result
	}

	startTime := time.Now()
	resp, err := http.Post("http://localhost:11434/api/generate",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		result.Error = fmt.Sprintf("Failed to send request: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		result.Error = fmt.Sprintf("Failed to parse response: %v", err)
		return result
	}

	totalTime := time.Since(startTime)

	// Calculate metrics
	result.Success = true
	result.Response = genResp.Response
	result.TotalTokens = genResp.EvalCount
	result.PromptTokens = genResp.PromptEvalCount
	result.TotalTimeMs = float64(totalTime.Milliseconds())

	// Tokens per second = eval_count / (eval_duration in nanoseconds) * 10^9
	if genResp.EvalDuration > 0 {
		result.TokensPerSecond = float64(genResp.EvalCount) / float64(genResp.EvalDuration) * 1e9
	}

	// Time to first token (approximate using load + prompt eval time)
	if genResp.LoadDuration > 0 && genResp.PromptEvalDuration > 0 {
		result.TimeToFirstToken = float64(genResp.LoadDuration+genResp.PromptEvalDuration) / 1e6
	}

	return result
}

func displayComparison(comparisons []ModelComparison) {
	if len(comparisons) == 0 {
		fmt.Println("No results to display.")
		return
	}

	// Overall performance ranking
	fmt.Println("Overall Performance Ranking (by avg tokens/sec):")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for i, comp := range comparisons {
		fmt.Printf("%d. %-20s | Avg Speed: %6.2f t/s | Avg Time: %7.2f ms\n",
			i+1, comp.ModelName, comp.AvgTokensPerSec, comp.AvgTotalTimeMs)
	}

	// Category breakdown
	categories := map[string]bool{}
	for _, comp := range comparisons {
		for _, result := range comp.TestResults {
			categories[result.Category] = true
		}
	}

	for category := range categories {
		fmt.Printf("\n\nCategory: %s\n", category)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		for _, comp := range comparisons {
			for _, result := range comp.TestResults {
				if result.Category == category && result.Success {
					fmt.Printf("%-20s | %6.2f t/s | %7.2f ms | %d tokens\n",
						comp.ModelName, result.TokensPerSecond, result.TotalTimeMs, result.TotalTokens)
				}
			}
		}
	}

	// Best model for each category
	fmt.Println("\n\nBest Model for Each Category:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for category := range categories {
		bestModel := ""
		bestSpeed := 0.0

		for _, comp := range comparisons {
			for _, result := range comp.TestResults {
				if result.Category == category && result.Success && result.TokensPerSecond > bestSpeed {
					bestSpeed = result.TokensPerSecond
					bestModel = comp.ModelName
				}
			}
		}

		if bestModel != "" {
			fmt.Printf("%-15s: %s (%.2f t/s)\n", category, bestModel, bestSpeed)
		}
	}
}
