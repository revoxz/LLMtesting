package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type SystemResources struct {
	OS          string
	Arch        string
	CPUCores    int
	TotalRAM    int64 // in GB
	GPU         string
	GPUMemory   int64 // in GB
	HasMetalAPI bool
}

type ColimaInfo struct {
	Installed     bool
	Running       bool
	CPUs          int
	Memory        int64 // in GB
	Disk          int64 // in GB
	Runtime       string
	Arch          string
}

type LLMModel struct {
	Name         string
	MinRAM       int64 // in GB
	MinGPUMemory int64 // in GB (0 if CPU only)
	RequiresGPU  bool
}

func main() {
	fmt.Println("=== LLM Compatibility Checker for Mac ===\n")

	// Get system resources
	resources, err := getSystemResources()
	if err != nil {
		fmt.Printf("Error getting system resources: %v\n", err)
		return
	}

	// Display system information
	displaySystemInfo(resources)

	// Check Colima
	colima := checkColima()
	displayColimaInfo(colima, resources)

	// Define popular LLM models with their requirements
	// RAM estimates are based on Q4/Q5 quantization (typical for Ollama)
	// Formula: ~1.5-2GB per billion parameters for Q4, ~2-2.5GB for Q5
	models := []LLMModel{
		{"Llama 3.2 1B (Q4)", 2, 0, false},
		{"Llama 3.2 3B (Q4)", 4, 0, false},
		{"Llama 3.1 8B (Q4)", 6, 0, false},
		{"Llama 3.1 70B (Q4)", 40, 0, false},
		{"Llama 3.1 405B (Q4)", 220, 0, false},
		{"GPT-2 Small 124M (Q4)", 1, 0, false},
		{"GPT-2 Medium 355M (Q4)", 1, 0, false},
		{"GPT-2 Large 774M (Q4)", 2, 0, false},
		{"Mistral 7B (Q4)", 5, 0, false},
		{"Mixtral 8x7B (Q4)", 30, 0, false},
		{"Phi-3 Mini 3.8B (Q4)", 3, 0, false},
		{"Phi-3 Medium 14B (Q4)", 9, 0, false},
		{"Gemma 2B (Q4)", 2, 0, false},
		{"Gemma 7B (Q4)", 5, 0, false},
		{"CodeLlama 7B (Q4)", 5, 0, false},
		{"CodeLlama 13B (Q4)", 8, 0, false},
		{"CodeLlama 34B (Q4)", 20, 0, false},
		{"Qwen 2.5 0.5B (Q4)", 1, 0, false},
		{"Qwen 2.5 1.5B (Q4)", 2, 0, false},
		{"Qwen 2.5 7B (Q4)", 5, 0, false},
		{"Qwen 2.5 14B (Q4)", 9, 0, false},
		{"Qwen 3 0.6B (Q4)", 1, 0, false},
		{"Qwen 3 1.7B (Q4)", 2, 0, false},
		{"Qwen 3 3B (Q4)", 3, 0, false},
		{"Qwen 3 8B (Q4)", 6, 0, false},
		{"Qwen 3 14B (Q4)", 9, 0, false},
		{"Qwen 3 32B (Q4)", 20, 0, false},
		{"Qwen 3 70B (Q4)", 40, 0, false},
		{"Qwen 3 235B (Q4)", 130, 0, false},
		{"DeepSeek R1 1.5B (Q4)", 2, 0, false},
		{"DeepSeek R1 7B (Q4)", 5, 0, false},
		{"DeepSeek R1 8B (Q4)", 6, 0, false},
		{"DeepSeek R1 14B (Q4)", 9, 0, false},
		{"DeepSeek R1 32B (Q4)", 20, 0, false},
		{"DeepSeek R1 70B (Q4)", 40, 0, false},
		{"DeepSeek R1 671B (Q4)", 370, 0, false},
		{"DeepSeek Coder 1.3B (Q4)", 2, 0, false},
		{"DeepSeek Coder 6.7B (Q4)", 5, 0, false},
		{"DeepSeek Coder 33B (Q4)", 20, 0, false},
		{"Nomic Embed Text v1.5", 1, 0, false},
		{"Nomic Embed Text v1", 1, 0, false},
		{"Stable Diffusion XL", 10, 6, true},
		{"Stable Diffusion 1.5", 6, 4, true},
	}

	// Check compatibility
	fmt.Println("\n=== Model Compatibility Check ===\n")
	checkModelCompatibility(resources, models)
}

func getSystemResources() (*SystemResources, error) {
	resources := &SystemResources{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
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
	resources.TotalRAM = ramBytes / (1024 * 1024 * 1024) // Convert to GB

	// Get GPU information (macOS specific)
	gpuCmd := exec.Command("system_profiler", "SPDisplaysDataType")
	gpuOutput, err := gpuCmd.Output()
	if err == nil {
		gpuInfo := string(gpuOutput)
		resources.GPU = extractGPUName(gpuInfo)
		resources.GPUMemory = extractGPUMemory(gpuInfo)
	}

	// Check for Metal API support (all modern Macs have it)
	if resources.GPU != "" && runtime.GOOS == "darwin" {
		resources.HasMetalAPI = true
	}

	return resources, nil
}

func extractGPUName(gpuInfo string) string {
	lines := strings.Split(gpuInfo, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Chipset Model:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

func extractGPUMemory(gpuInfo string) int64 {
	lines := strings.Split(gpuInfo, "\n")
	for _, line := range lines {
		if strings.Contains(line, "VRAM") || strings.Contains(line, "Metal Support") {
			// Try to extract memory size
			if strings.Contains(line, "GB") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if strings.Contains(part, "GB") && i > 0 {
						memStr := strings.TrimSpace(parts[i-1])
						if mem, err := strconv.ParseFloat(memStr, 64); err == nil {
							return int64(mem)
						}
					}
				}
			}
		}
	}
	// For Apple Silicon, unified memory is shared
	return 0 // Will use shared memory estimate
}

func displaySystemInfo(resources *SystemResources) {
	fmt.Println("System Information:")
	fmt.Printf("  OS: %s\n", resources.OS)
	fmt.Printf("  Architecture: %s\n", resources.Arch)
	fmt.Printf("  CPU Cores: %d\n", resources.CPUCores)
	fmt.Printf("  Total RAM: %d GB\n", resources.TotalRAM)
	fmt.Printf("  GPU: %s\n", resources.GPU)
	if resources.GPUMemory > 0 {
		fmt.Printf("  GPU Memory: %d GB\n", resources.GPUMemory)
	} else {
		fmt.Printf("  GPU Memory: Unified memory (shared with RAM)\n")
	}
	fmt.Printf("  Metal API Support: %v\n", resources.HasMetalAPI)
}

func checkColima() *ColimaInfo {
	info := &ColimaInfo{}

	// Check if colima is installed
	_, err := exec.LookPath("colima")
	if err != nil {
		info.Installed = false
		return info
	}
	info.Installed = true

	// Check if colima is running
	statusCmd := exec.Command("colima", "status")
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(statusOutput), "running") {
		info.Running = false
		return info
	}
	info.Running = true

	// Get colima configuration
	listCmd := exec.Command("colima", "list", "--json")
	listOutput, err := listCmd.Output()
	if err == nil {
		output := string(listOutput)

		// Parse CPU
		if strings.Contains(output, "\"cpu\":") {
			parts := strings.Split(output, "\"cpu\":")
			if len(parts) > 1 {
				cpuPart := strings.Split(parts[1], ",")[0]
				cpuPart = strings.TrimSpace(cpuPart)
				if cpu, err := strconv.Atoi(cpuPart); err == nil {
					info.CPUs = cpu
				}
			}
		}

		// Parse Memory
		if strings.Contains(output, "\"memory\":") {
			parts := strings.Split(output, "\"memory\":")
			if len(parts) > 1 {
				memPart := strings.Split(parts[1], ",")[0]
				memPart = strings.TrimSpace(memPart)
				if mem, err := strconv.ParseInt(memPart, 10, 64); err == nil {
					info.Memory = mem
				}
			}
		}

		// Parse Disk
		if strings.Contains(output, "\"disk\":") {
			parts := strings.Split(output, "\"disk\":")
			if len(parts) > 1 {
				diskPart := strings.Split(parts[1], ",")[0]
				diskPart = strings.TrimSpace(diskPart)
				if disk, err := strconv.ParseInt(diskPart, 10, 64); err == nil {
					info.Disk = disk
				}
			}
		}

		// Parse Runtime
		if strings.Contains(output, "\"runtime\":") {
			parts := strings.Split(output, "\"runtime\":")
			if len(parts) > 1 {
				runtimePart := strings.Split(parts[1], "\"")[1]
				info.Runtime = runtimePart
			}
		}

		// Parse Arch
		if strings.Contains(output, "\"arch\":") {
			parts := strings.Split(output, "\"arch\":")
			if len(parts) > 1 {
				archPart := strings.Split(parts[1], "\"")[1]
				info.Arch = archPart
			}
		}
	}

	return info
}

func displayColimaInfo(colima *ColimaInfo, resources *SystemResources) {
	fmt.Println("\n=== Colima (Container Runtime) Check ===")

	if !colima.Installed {
		fmt.Println("Status: Not installed")
		fmt.Println("ℹ️  Colima is a container runtime for macOS (alternative to Docker Desktop)")
		fmt.Println("   Install: brew install colima")
		return
	}

	fmt.Println("Status: Installed ✓")

	if !colima.Running {
		fmt.Println("Running: No")
		fmt.Println("ℹ️  Start Colima: colima start")
		return
	}

	fmt.Println("Running: Yes ✓")
	fmt.Printf("\nColima Configuration:\n")
	fmt.Printf("  CPUs: %d (of %d system cores)\n", colima.CPUs, resources.CPUCores)
	fmt.Printf("  Memory: %d GB (of %d GB system RAM)\n", colima.Memory, resources.TotalRAM)
	fmt.Printf("  Disk: %d GB\n", colima.Disk)
	fmt.Printf("  Runtime: %s\n", colima.Runtime)
	fmt.Printf("  Architecture: %s\n", colima.Arch)

	// Recommendations
	fmt.Println("\n=== Colima Recommendations ===")

	recommendedCPU := resources.CPUCores / 2
	if recommendedCPU < 2 {
		recommendedCPU = 2
	}
	if recommendedCPU > 8 {
		recommendedCPU = 8
	}

	recommendedRAM := resources.TotalRAM / 2
	if recommendedRAM < 4 {
		recommendedRAM = 4
	}
	if recommendedRAM > 16 {
		recommendedRAM = 16
	}

	needsReconfiguration := false

	if colima.CPUs < recommendedCPU {
		fmt.Printf("⚠️  CPU: Consider increasing to %d cores for better performance\n", recommendedCPU)
		needsReconfiguration = true
	} else {
		fmt.Printf("✓ CPU: %d cores is good\n", colima.CPUs)
	}

	if colima.Memory < recommendedRAM {
		fmt.Printf("⚠️  RAM: Consider increasing to %d GB for better performance\n", recommendedRAM)
		needsReconfiguration = true
	} else {
		fmt.Printf("✓ RAM: %d GB is good\n", colima.Memory)
	}

	// LLM-specific recommendations
	fmt.Println("\n=== Running LLMs in Containers (Ollama in Colima) ===")

	maxLLMRAM := colima.Memory - 2 // Reserve 2GB for system
	if maxLLMRAM < 0 {
		maxLLMRAM = 0
	}

	fmt.Printf("Available RAM for LLMs in containers: ~%d GB\n", maxLLMRAM)

	if maxLLMRAM < 4 {
		fmt.Println("⚠️  WARNING: Not enough RAM for most LLMs in containers")
		fmt.Println("   Recommendation: Increase Colima RAM to at least 8 GB")
		fmt.Println("   Or run Ollama directly on your Mac (not in container)")
	} else if maxLLMRAM < 8 {
		fmt.Println("✓ You can run small models (1B-3B) in containers")
		fmt.Println("  Recommended: Llama 3.2 3B, Qwen 3 3B")
	} else if maxLLMRAM < 16 {
		fmt.Println("✓ You can run medium models (3B-8B) in containers")
		fmt.Println("  Recommended: Llama 3.1 8B, Qwen 2.5 7B")
	} else {
		fmt.Println("✓ You can run large models (8B-14B+) in containers")
		fmt.Println("  Recommended: Llama 3.1 8B, Phi-3 Medium 14B, Mixtral 8x7B")
	}

	if needsReconfiguration {
		fmt.Println("\n💡 To reconfigure Colima:")
		fmt.Printf("   colima stop\n")
		fmt.Printf("   colima start --cpu %d --memory %d\n", recommendedCPU, recommendedRAM)
	}

	// Best practice recommendation
	fmt.Println("\n💡 Best Practice:")
	fmt.Println("   For best LLM performance, consider running Ollama directly on macOS")
	fmt.Println("   instead of in a container (Colima/Docker). Native execution with")
	fmt.Println("   Metal API provides better performance and uses less overhead.")
}

func checkModelCompatibility(resources *SystemResources, models []LLMModel) {
	compatible := []string{}
	incompatible := []string{}

	for _, model := range models {
		canRun := true
		reason := ""

		// For Apple Silicon, GPU memory is unified with system RAM
		availableMemory := resources.TotalRAM
		if resources.Arch == "arm64" && resources.HasMetalAPI {
			// On Apple Silicon, we can use ~70% of RAM for models safely
			availableMemory = int64(float64(resources.TotalRAM) * 0.7)
		}

		// Check RAM requirement
		if model.MinRAM > availableMemory {
			canRun = false
			reason = fmt.Sprintf("Insufficient RAM (need %d GB, have %d GB available)", model.MinRAM, availableMemory)
		}

		// Check GPU requirement for models that need dedicated GPU
		if model.RequiresGPU && !resources.HasMetalAPI {
			canRun = false
			reason = "Requires GPU acceleration (Metal API not available)"
		}

		// Check dedicated GPU memory (mainly for image generation models on Intel Macs)
		if model.MinGPUMemory > 0 && resources.Arch != "arm64" {
			if resources.GPUMemory < model.MinGPUMemory {
				canRun = false
				reason = fmt.Sprintf("Insufficient GPU memory (need %d GB, have %d GB)", model.MinGPUMemory, resources.GPUMemory)
			}
		}

		if canRun {
			status := "✓"
			requirements := ""

			// Build requirements string
			if model.MinRAM > 0 {
				requirements = fmt.Sprintf("RAM: %d GB", model.MinRAM)
			}
			if model.MinGPUMemory > 0 {
				if requirements != "" {
					requirements += ", "
				}
				requirements += fmt.Sprintf("GPU Memory: %d GB", model.MinGPUMemory)
			}
			if model.RequiresGPU {
				if requirements != "" {
					requirements += ", "
				}
				requirements += "GPU required"
			}

			if resources.Arch == "arm64" && resources.HasMetalAPI {
				status += " (Metal optimized)"
			}

			compatible = append(compatible, fmt.Sprintf("  %s %-30s [%s]", status, model.Name, requirements))
		} else {
			incompatible = append(incompatible, fmt.Sprintf("  ✗ %s - %s", model.Name, reason))
		}
	}

	// Display results
	fmt.Println("Compatible Models:")
	if len(compatible) > 0 {
		for _, model := range compatible {
			fmt.Println(model)
		}
	} else {
		fmt.Println("  None")
	}

	fmt.Println("\nIncompatible Models:")
	if len(incompatible) > 0 {
		for _, model := range incompatible {
			fmt.Println(model)
		}
	} else {
		fmt.Println("  None")
	}

	// Recommendations
	fmt.Println("\n=== Recommendations ===")

	// Architecture-specific recommendations
	if resources.Arch == "arm64" && resources.HasMetalAPI {
		fmt.Println("✓ Your Mac has Apple Silicon with Metal support - excellent for running LLMs!")
		fmt.Println("✓ Consider using llama.cpp, Ollama, or MLX for optimized performance")
	} else {
		fmt.Println("• Your Mac has Intel architecture - LLMs will run slower than on Apple Silicon")
		fmt.Println("• Consider using llama.cpp or Ollama for CPU inference")
	}

	// RAM-specific recommendations
	fmt.Println()
	if resources.TotalRAM >= 64 {
		fmt.Println("✓ You have plenty of RAM for large models (32B-70B with Q4)")
		fmt.Println("  Suggested: Llama 3.1 70B, Qwen 3 70B, Mixtral 8x7B")
	} else if resources.TotalRAM >= 32 {
		fmt.Println("✓ You have good RAM for medium-sized models (7B-32B with Q4)")
		fmt.Println("  Suggested: Llama 3.1 8B, DeepSeek R1 32B, CodeLlama 34B")
	} else if resources.TotalRAM >= 16 {
		fmt.Println("✓ You have sufficient RAM for small-medium models (1B-14B with Q4)")
		fmt.Println("  Suggested: Qwen 2.5 7B, Phi-3 Medium, DeepSeek Coder 6.7B")
	} else {
		fmt.Println("• Your RAM is limited - stick to smaller models (0.5B-3B)")
		fmt.Println("  Suggested: Llama 3.2 3B, Qwen 3 3B, Phi-3 Mini")
	}

	// Quantization recommendations
	fmt.Println("\n=== About Quantization ===")
	fmt.Println("All RAM estimates above assume Q4 quantization (most common).")
	fmt.Println("• Q4: Best balance of quality and size (~0.5-0.6GB per billion params)")
	fmt.Println("• Q5: Better quality, 20% more RAM")
	fmt.Println("• Q8: Near-perfect quality, 50% more RAM")
	fmt.Println("• Q2: Very small, noticeable quality loss")
	fmt.Println("\nTo use specific quantization in Ollama:")
	fmt.Println("  ollama pull llama3.1:8b-q4_0   # Q4 (recommended)")
	fmt.Println("  ollama pull llama3.1:8b-q5_K_M # Q5 (better quality)")
}
