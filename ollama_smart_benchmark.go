package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config structures
type Config struct {
	LLMFamilies    []LLMFamily    `json:"llm_families"`
	ResourceLimits ResourceLimits `json:"resource_limits"`
	TestSettings   TestSettings   `json:"test_settings"`
}

type LLMFamily struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	TestAllVariants bool   `json:"test_all_variants"`
}

type ResourceLimits struct {
	MaxRAMUsagePercent int `json:"max_ram_usage_percent"`
	MinFreeRAMGB       int `json:"min_free_ram_gb"`
}

type TestSettings struct {
	AutoPullModels             bool `json:"auto_pull_models"`
	SkipIfInsufficientResources bool `json:"skip_if_insufficient_resources"`
	ParallelTesting            bool `json:"parallel_testing"`
}

// System resources
type SystemInfo struct {
	TotalRAMGB    int64
	AvailableRAMGB int64
	Arch          string
}

// Ollama API structures
type OllamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

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

// Test structures
type TestCase struct {
	Name     string
	Prompt   string
	Category string
}

type BenchmarkResult struct {
	ModelName        string
	ModelSize        string
	TestName         string
	Category         string
	TokensPerSecond  float64
	TimeToFirstToken float64
	TotalTokens      int
	PromptTokens     int
	TotalTimeMs      float64
	Response         string
	Success          bool
	Error            string
	RAMUsedGB        float64
}

type ModelSummary struct {
	ModelName       string
	ModelSize       string
	AvgTokensPerSec float64
	AvgTotalTimeMs  float64
	TestResults     []BenchmarkResult
	CanRun          bool
	SkipReason      string
}

func main() {
	fmt.Println("=== Smart Ollama LLM Benchmark ===\n")

	// Load config
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Get system info
	sysInfo, err := getSystemInfo()
	if err != nil {
		fmt.Printf("Error getting system info: %v\n", err)
		return
	}

	fmt.Printf("System Info:\n")
	fmt.Printf("  Total RAM: %d GB\n", sysInfo.TotalRAMGB)
	fmt.Printf("  Available RAM: %d GB\n", sysInfo.AvailableRAMGB)
	fmt.Printf("  Architecture: %s\n\n", sysInfo.Arch)

	// Check if Ollama is running
	if !checkOllamaRunning() {
		fmt.Println("Error: Ollama is not running. Please start Ollama first.")
		fmt.Println("Run: ollama serve")
		return
	}

	// Get all available models from Ollama library
	fmt.Println("Fetching available models from Ollama library...")
	availableModels := getOllamaLibraryModels(config)

	if len(availableModels) == 0 {
		fmt.Println("No models found to test. Please check your config.json")
		return
	}

	fmt.Printf("\nFound %d model variants to test:\n", len(availableModels))
	for _, model := range availableModels {
		fmt.Printf("  - %s\n", model)
	}

	// Filter models based on system resources
	testableModels := filterModelsByResources(availableModels, sysInfo, config)

	fmt.Printf("\n%d models are testable on your system:\n", len(testableModels))
	for _, model := range testableModels {
		fmt.Printf("  ✓ %s\n", model)
	}

	if len(availableModels) > len(testableModels) {
		fmt.Printf("\n%d models skipped due to insufficient resources:\n", len(availableModels)-len(testableModels))
		for _, model := range availableModels {
			found := false
			for _, tm := range testableModels {
				if tm == model {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("  ✗ %s\n", model)
			}
		}
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
	var summaries []ModelSummary

	for _, model := range testableModels {
		fmt.Printf("\n=== Testing Model: %s ===\n", model)

		// Check if model is installed locally
		if !checkModelInstalled(model) {
			if config.TestSettings.AutoPullModels {
				fmt.Printf("Model %s not installed. Pulling model...\n", model)
				if !pullModel(model) {
					fmt.Printf("Failed to pull model %s. Skipping...\n", model)
					summaries = append(summaries, ModelSummary{
						ModelName:  model,
						CanRun:     false,
						SkipReason: "Failed to pull model",
					})
					continue
				}
			} else {
				fmt.Printf("Model %s not installed. Skipping (auto_pull disabled)...\n", model)
				summaries = append(summaries, ModelSummary{
					ModelName:  model,
					CanRun:     false,
					SkipReason: "Model not installed",
				})
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
				fmt.Printf("    ✓ Tokens/sec: %.2f | Total time: %.2fms | Tokens: %d | RAM: %.1f GB\n",
					result.TokensPerSecond, result.TotalTimeMs, result.TotalTokens, result.RAMUsedGB)
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

		summaries = append(summaries, ModelSummary{
			ModelName:       model,
			ModelSize:       extractModelSize(model),
			AvgTokensPerSec: avgTPS,
			AvgTotalTimeMs:  avgTime,
			TestResults:     results,
			CanRun:          successCount > 0,
		})
	}

	// Display results
	fmt.Println("\n\n=== Benchmark Results ===\n")
	displayResults(summaries, sysInfo)
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func getSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		Arch: runtime.GOARCH,
	}

	// Get total RAM (macOS specific)
	ramCmd := exec.Command("sysctl", "-n", "hw.memsize")
	ramOutput, err := ramCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get RAM: %v", err)
	}
	ramBytes, err := strconv.ParseInt(strings.TrimSpace(string(ramOutput)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RAM: %v", err)
	}
	info.TotalRAMGB = ramBytes / (1024 * 1024 * 1024)

	// For Apple Silicon, use 70% of total RAM as available for LLMs
	if info.Arch == "arm64" {
		info.AvailableRAMGB = int64(float64(info.TotalRAMGB) * 0.7)
	} else {
		info.AvailableRAMGB = info.TotalRAMGB - 8 // Reserve 8GB for system
	}

	return info, nil
}

func checkOllamaRunning() bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func getOllamaLibraryModels(config *Config) []string {
	var models []string

	// Get installed models
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return models
	}
	defer resp.Body.Close()

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return models
	}

	installedModels := make(map[string]bool)
	for _, m := range tagsResp.Models {
		installedModels[m.Name] = true
	}

	// For each enabled family, find all variants
	for _, family := range config.LLMFamilies {
		if !family.Enabled {
			continue
		}

		// Check installed models for this family
		for modelName := range installedModels {
			if strings.HasPrefix(modelName, family.Name) {
				models = append(models, modelName)
			}
		}

		// If test_all_variants is true, we'd need to query Ollama library
		// For now, we'll add common variants
		if family.TestAllVariants {
			commonVariants := getCommonVariants(family.Name)
			for _, variant := range commonVariants {
				// Only add if not already in list
				found := false
				for _, m := range models {
					if m == variant {
						found = true
						break
					}
				}
				if !found {
					models = append(models, variant)
				}
			}
		}
	}

	// Sort models
	sort.Strings(models)

	return models
}

func getCommonVariants(family string) []string {
	variants := map[string][]string{
		"qwen2.5": {"qwen2.5:0.5b", "qwen2.5:1.5b", "qwen2.5:3b", "qwen2.5:7b", "qwen2.5:14b", "qwen2.5:32b"},
		"gemma2":  {"gemma2:2b", "gemma2:9b", "gemma2:27b"},
		"llama3.2": {"llama3.2:1b", "llama3.2:3b"},
		"llama3.1": {"llama3.1:8b", "llama3.1:70b", "llama3.1:405b"},
		"mistral":  {"mistral:7b", "mistral:latest"},
		"codellama": {"codellama:7b", "codellama:13b", "codellama:34b", "codellama:70b"},
		"phi3":      {"phi3:mini", "phi3:medium"},
		"deepseek-coder": {"deepseek-coder:1.3b", "deepseek-coder:6.7b", "deepseek-coder:33b"},
	}

	if v, ok := variants[family]; ok {
		return v
	}
	return []string{family + ":latest"}
}

func extractModelSize(modelName string) string {
	parts := strings.Split(modelName, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "unknown"
}

func estimateModelRAM(modelName string) int64 {
	size := extractModelSize(modelName)

	// RAM estimates based on Q4 quantization (typical for Ollama)
	// Formula: ~0.5-0.6 GB per billion parameters for Q4
	// For Q5: add ~20%, for Q8: add ~50%, for F16: multiply by ~2
	var sizeNum int64 = 5 // default

	if strings.Contains(size, "0.5b") || strings.Contains(size, "0.6b") {
		sizeNum = 1
	} else if strings.Contains(size, "1b") || strings.Contains(size, "1.3b") || strings.Contains(size, "1.5b") || strings.Contains(size, "1.7b") {
		sizeNum = 2
	} else if strings.Contains(size, "2b") {
		sizeNum = 2
	} else if strings.Contains(size, "3b") {
		sizeNum = 3
	} else if strings.Contains(size, "6.7b") || strings.Contains(size, "7b") {
		sizeNum = 5
	} else if strings.Contains(size, "8b") {
		sizeNum = 6
	} else if strings.Contains(size, "9b") {
		sizeNum = 6
	} else if strings.Contains(size, "13b") || strings.Contains(size, "14b") {
		sizeNum = 9
	} else if strings.Contains(size, "27b") {
		sizeNum = 16
	} else if strings.Contains(size, "32b") || strings.Contains(size, "33b") || strings.Contains(size, "34b") {
		sizeNum = 20
	} else if strings.Contains(size, "70b") {
		sizeNum = 40
	} else if strings.Contains(size, "235b") {
		sizeNum = 130
	} else if strings.Contains(size, "405b") {
		sizeNum = 220
	} else if strings.Contains(size, "671b") {
		sizeNum = 370
	} else if strings.Contains(size, "mini") {
		sizeNum = 3
	} else if strings.Contains(size, "medium") {
		sizeNum = 9
	}

	return sizeNum
}

func filterModelsByResources(models []string, sysInfo *SystemInfo, config *Config) []string {
	if !config.TestSettings.SkipIfInsufficientResources {
		return models
	}

	var testable []string

	for _, model := range models {
		estimatedRAM := estimateModelRAM(model)
		minFree := int64(config.ResourceLimits.MinFreeRAMGB)

		if estimatedRAM+minFree <= sysInfo.AvailableRAMGB {
			testable = append(testable, model)
		}
	}

	return testable
}

func checkModelInstalled(model string) bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return false
	}

	for _, m := range tagsResp.Models {
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

	// Read pull progress
	decoder := json.NewDecoder(resp.Body)
	for {
		var status map[string]interface{}
		if err := decoder.Decode(&status); err == io.EOF {
			break
		} else if err != nil {
			return false
		}

		if statusStr, ok := status["status"].(string); ok {
			if strings.Contains(statusStr, "success") {
				return true
			}
		}
	}
	return true
}

func runBenchmark(model string, test TestCase) BenchmarkResult {
	result := BenchmarkResult{
		ModelName: model,
		ModelSize: extractModelSize(model),
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
	result.RAMUsedGB = float64(estimateModelRAM(model))

	if genResp.EvalDuration > 0 {
		result.TokensPerSecond = float64(genResp.EvalCount) / float64(genResp.EvalDuration) * 1e9
	}

	if genResp.LoadDuration > 0 && genResp.PromptEvalDuration > 0 {
		result.TimeToFirstToken = float64(genResp.LoadDuration+genResp.PromptEvalDuration) / 1e6
	}

	return result
}

func displayResults(summaries []ModelSummary, sysInfo *SystemInfo) {
	if len(summaries) == 0 {
		fmt.Println("No results to display.")
		return
	}

	// Filter only successful models
	var successful []ModelSummary
	for _, s := range summaries {
		if s.CanRun {
			successful = append(successful, s)
		}
	}

	// Sort by average tokens per second (descending)
	sort.Slice(successful, func(i, j int) bool {
		return successful[i].AvgTokensPerSec > successful[j].AvgTokensPerSec
	})

	// Overall ranking
	fmt.Println("Overall Performance Ranking (by avg tokens/sec):")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for i, s := range successful {
		fmt.Printf("%d. %-25s | Size: %-8s | Avg Speed: %6.2f t/s | Avg Time: %7.2f ms\n",
			i+1, s.ModelName, s.ModelSize, s.AvgTokensPerSec, s.AvgTotalTimeMs)
	}

	// Category breakdown
	categories := map[string]bool{}
	for _, s := range successful {
		for _, r := range s.TestResults {
			if r.Success {
				categories[r.Category] = true
			}
		}
	}

	for category := range categories {
		fmt.Printf("\n\nCategory: %s\n", category)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		for _, s := range successful {
			for _, r := range s.TestResults {
				if r.Category == category && r.Success {
					fmt.Printf("%-25s | %6.2f t/s | %7.2f ms | %d tokens\n",
						s.ModelName, r.TokensPerSecond, r.TotalTimeMs, r.TotalTokens)
				}
			}
		}
	}

	// Best model for each category
	fmt.Println("\n\nBest Model for Each Category:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for category := range categories {
		bestModel := ""
		bestSpeed := 0.0

		for _, s := range successful {
			for _, r := range s.TestResults {
				if r.Category == category && r.Success && r.TokensPerSecond > bestSpeed {
					bestSpeed = r.TokensPerSecond
					bestModel = s.ModelName
				}
			}
		}

		if bestModel != "" {
			fmt.Printf("%-15s: %s (%.2f t/s)\n", category, bestModel, bestSpeed)
		}
	}

	// Recommendations
	fmt.Println("\n\n=== Recommendations for Your System ===")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if len(successful) > 0 {
		fmt.Printf("✓ Best overall performer: %s (%.2f t/s)\n",
			successful[0].ModelName, successful[0].AvgTokensPerSec)

		// Find smallest working model
		var smallest *ModelSummary
		for i := range successful {
			if smallest == nil || estimateModelRAM(successful[i].ModelName) < estimateModelRAM(smallest.ModelName) {
				smallest = &successful[i]
			}
		}
		if smallest != nil {
			fmt.Printf("✓ Most efficient (smallest): %s (~%.0f GB RAM)\n",
				smallest.ModelName, float64(estimateModelRAM(smallest.ModelName)))
		}
	}

	fmt.Printf("\nSystem capacity: %d GB RAM available for LLMs\n", sysInfo.AvailableRAMGB)
	fmt.Printf("Architecture: %s\n", sysInfo.Arch)

	if sysInfo.Arch == "arm64" {
		fmt.Println("✓ Apple Silicon detected - excellent performance with Metal API")
	}
}
