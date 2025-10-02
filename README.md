# LLM Testing

A comprehensive toolkit for Mac systems to check LLM compatibility and benchmark Ollama model performance.

## Overview

This project contains three Go-based utilities:
1. **System Compatibility Checker** - Analyzes your Mac's hardware and determines which LLMs can run
2. **Ollama Benchmark Tool** - Tests and compares multiple Ollama models for speed and capability
3. **Smart Ollama Benchmark** - Config-based automatic testing of LLM families with resource-aware filtering

## Project Structure

```
LLMtesting/
├── llm_checker.go            # System resource and LLM compatibility checker
├── ollama_benchmark.go       # Basic Ollama model benchmarking tool
├── ollama_smart_benchmark.go # Smart config-based benchmarking with auto-detection
├── config.json               # Configuration file for LLM families to test
├── pom.xml                   # Maven project configuration
├── .gitignore                # Git ignore rules
└── LLMtesting.iml            # IntelliJ IDEA module file
```

## Features

### System Compatibility Checker (llm_checker.go)

- **System Resource Detection**
  - Detects OS and architecture (Intel vs Apple Silicon)
  - Reports CPU cores and total RAM
  - Identifies GPU model and memory
  - Checks for Metal API support

- **LLM Compatibility Check**
  - Evaluates compatibility with popular LLM models including:
    - Llama 3.2 (1B, 3B)
    - Llama 3.1 (8B, 70B, 405B)
    - GPT-2 (Small, Medium, Large)
    - Mistral 7B
    - Mixtral 8x7B
    - Phi-3 (Mini, Medium)
    - Gemma (2B, 7B)
    - CodeLlama (7B, 13B, 34B)
    - Qwen 2.5 (0.5B, 1.5B, 7B, 14B)
    - Stable Diffusion (1.5, XL)

- **Smart Recommendations**
  - Apple Silicon optimization detection
  - RAM-based model suggestions
  - Recommended tools (llama.cpp, Ollama, MLX)

### Ollama Benchmark Tool (ollama_benchmark.go)

- **Performance Benchmarking**
  - Measures tokens per second (t/s) for each model
  - Calculates time to first token (TTFT)
  - Tracks total response time and token counts
  - Tests multiple models in parallel

- **Capability Testing**
  - Simple reasoning tasks
  - Code generation tests
  - Mathematical problem solving
  - Creative writing evaluation
  - Question answering

- **Model Comparison**
  - Side-by-side performance metrics
  - Category-specific rankings (coding, reasoning, math, creative, Q&A)
  - Best model identification for each category
  - Automatic model pulling if not installed

### Smart Ollama Benchmark (ollama_smart_benchmark.go) ⭐ NEW

- **Config-Based Testing**
  - Define LLM families in `config.json` (e.g., qwen2.5, gemma2, llama3.2)
  - Automatically discovers all available variants (0.5b, 1b, 3b, 7b, etc.)
  - Enable/disable model families without code changes

- **Resource-Aware Filtering**
  - Automatically detects system RAM
  - Estimates RAM requirements for each model variant
  - Skips models that won't fit in available memory
  - Configurable RAM safety margins

- **Intelligent Model Detection**
  - Checks locally installed models
  - Discovers common variants for each LLM family
  - Auto-pulls missing models (configurable)
  - Tests only compatible models

- **Enhanced Reporting**
  - Shows which models can/cannot run on your system
  - Identifies most efficient model (smallest with good performance)
  - Best overall performer ranking
  - Category-specific best model recommendations

## Requirements

- macOS
- Go 1.16 or higher
- Ollama (for benchmark tool) - Install from [ollama.com](https://ollama.com)
- Java 24 (if using Maven components)

## Usage

### System Compatibility Checker

```bash
go run llm_checker.go
```

The tool will:
1. Detect your system resources
2. Display your hardware specifications
3. Show which models are compatible with your system
4. Provide recommendations based on your hardware

**Output includes:**
- System information summary
- List of compatible models (with Metal optimization status for Apple Silicon)
- List of incompatible models with reasons
- Personalized recommendations based on your hardware

### Ollama Benchmark Tool (Basic)

First, ensure Ollama is running:
```bash
ollama serve
```

Then run the benchmark:
```bash
go run ollama_benchmark.go
```

The tool will:
1. Check if Ollama is running
2. Pull any missing models automatically
3. Run 5 different test categories for each model
4. Measure performance metrics (tokens/sec, total time, etc.)
5. Display comprehensive comparison results

**Tested Models** (default configuration):
- llama3.2:1b
- llama3.2:3b
- gemma2:2b
- qwen2.5:0.5b

### Smart Ollama Benchmark (Recommended) ⭐

First, configure which LLM families to test in `config.json`:

```json
{
  "llm_families": [
    {
      "name": "qwen2.5",
      "enabled": true,
      "test_all_variants": true
    },
    {
      "name": "gemma2",
      "enabled": true,
      "test_all_variants": true
    },
    {
      "name": "llama3.2",
      "enabled": true,
      "test_all_variants": true
    }
  ],
  "resource_limits": {
    "max_ram_usage_percent": 70,
    "min_free_ram_gb": 4
  },
  "test_settings": {
    "auto_pull_models": true,
    "skip_if_insufficient_resources": true,
    "parallel_testing": false
  }
}
```

Then run:
```bash
ollama serve  # In one terminal
go run ollama_smart_benchmark.go  # In another terminal
```

The tool will:
1. Load your configuration from `config.json`
2. Detect your system's available RAM
3. Discover all variants for enabled LLM families (e.g., qwen2.5:0.5b, qwen2.5:1.5b, qwen2.5:3b, etc.)
4. Filter models based on available resources
5. Automatically pull missing models (if enabled)
6. Run comprehensive benchmarks on compatible models
7. Show which models can/cannot run on your system

**Test Categories:**
- Simple Reasoning
- Code Generation
- Mathematical Problems
- Creative Writing
- Question Answering

**Output includes:**
- System resource summary
- List of discovered model variants
- Which models are testable vs. skipped (due to RAM)
- Overall performance ranking by average tokens/second
- Category-specific performance breakdown
- Best model identification for each task type
- Recommendations (best overall, most efficient)
- Detailed metrics: tokens/sec, total time, token counts, RAM usage

## Configuration Guide

### config.json Structure

**LLM Families:**
- `name`: The base model family name (e.g., "qwen2.5", "gemma2", "llama3.2")
- `enabled`: Set to `true` to test this family, `false` to skip
- `test_all_variants`: When `true`, tests all size variants (0.5b, 1b, 3b, 7b, etc.)

**Supported LLM Families:**
- `qwen2.5` - Variants: 0.5b, 1.5b, 3b, 7b, 14b, 32b
- `gemma2` - Variants: 2b, 9b, 27b
- `llama3.2` - Variants: 1b, 3b
- `llama3.1` - Variants: 8b, 70b, 405b
- `mistral` - Variants: 7b, latest
- `codellama` - Variants: 7b, 13b, 34b, 70b
- `phi3` - Variants: mini, medium
- `deepseek-coder` - Variants: 1.3b, 6.7b, 33b

**Resource Limits:**
- `max_ram_usage_percent`: Maximum % of total RAM to use (default: 70 for Apple Silicon)
- `min_free_ram_gb`: Minimum GB to keep free (default: 4)

**Test Settings:**
- `auto_pull_models`: Automatically download missing models (default: true)
- `skip_if_insufficient_resources`: Skip models that won't fit in RAM (default: true)
- `parallel_testing`: Run tests in parallel (default: false, not yet implemented)

### Adding New LLM Families

Simply add to `config.json`:
```json
{
  "name": "mistral",
  "enabled": true,
  "test_all_variants": true
}
```

### Customizing Test Cases

Edit the test cases in the Go files to add your own prompts and categories.

## Understanding the Metrics

- **Tokens/sec (t/s)**: Generation speed - higher is better
- **Total Time (ms)**: Complete response time including model loading
- **Time to First Token (TTFT)**: Latency before first token appears
- **Token Count**: Number of tokens generated in response
- **Prompt Tokens**: Number of tokens in the input prompt

## Understanding Quantization

**Quantization** compresses LLM models to use less memory and run faster, with some quality tradeoff.

### Quantization Levels

| Level | Bits | Size Reduction | Quality | Use Case |
|-------|------|----------------|---------|----------|
| **F16** | 16-bit | 0% (original) | 100% | Maximum quality, large RAM |
| **Q8** | 8-bit | ~50% | ~99% | Near-perfect quality |
| **Q5** | 5-bit | ~65% | ~95% | Excellent balance |
| **Q4** | 4-bit | ~75% | ~90% | **Recommended** - best balance |
| **Q2** | 2-bit | ~87% | ~75% | Very small, noticeable quality loss |

### RAM Requirements Formula

**Q4 (Recommended):** ~0.5-0.6 GB per billion parameters
- 7B model: ~5 GB
- 8B model: ~6 GB
- 70B model: ~40 GB

**Q5:** Add ~20% to Q4 estimates
**Q8:** Add ~50% to Q4 estimates
**F16:** Double Q4 estimates

### Using Quantized Models in Ollama

```bash
# Default (usually Q4 or Q5)
ollama pull llama3.1:8b

# Explicitly specify quantization
ollama pull llama3.1:8b-q4_0      # Q4 quantization
ollama pull llama3.1:8b-q5_K_M    # Q5 quantization
ollama pull llama3.1:8b-q8_0      # Q8 quantization

# List available variants
ollama list
```

### Important Notes

- **All RAM estimates in this toolkit assume Q4 quantization** (most common for Ollama)
- Q4/Q5 is the sweet spot for most users - good quality, manageable size
- Apple Silicon Macs handle quantized models very efficiently with Metal API
- Smaller quantization = faster inference but lower quality

## Model Size Recommendations

Choosing the right model size depends on your use case, available RAM, and performance needs.

### Model Size Categories

#### 🔹 Tiny Models (0.5B - 2B)
**RAM Required (Q4):** 1-2 GB
**Speed:** Very Fast
**Quality:** Basic

**Best for:**
- Extremely resource-constrained systems
- Simple text completion
- Basic chatbots
- Embedded systems

**Recommended Models:**
- Qwen 2.5 0.5B - Fastest, minimal RAM
- Qwen 3 1.7B - Better quality, still tiny
- Gemma 2B - Good balance for basic tasks

---

#### 🔸 Small Models (3B - 4B)
**RAM Required (Q4):** 3-4 GB
**Speed:** Fast
**Quality:** Good

**Best for:**
- Daily coding assistance
- General Q&A
- Content summarization
- Educational purposes

**Recommended Models:**
- Llama 3.2 3B - Excellent for general use
- Qwen 3 3B - Strong reasoning abilities
- Phi-3 Mini 3.8B - Great for coding

---

#### 🔶 Medium Models (7B - 9B)
**RAM Required (Q4):** 5-6 GB
**Speed:** Moderate
**Quality:** Very Good

**Best for:**
- Professional coding assistance
- Complex reasoning tasks
- Content creation
- Technical documentation
- Research assistance

**Recommended Models:**
- Llama 3.1 8B - Industry standard, well-rounded
- Qwen 2.5 7B - Excellent for multilingual tasks
- Mistral 7B - Strong instruction following
- DeepSeek Coder 6.7B - Specialized for coding
- Gemma 7B - Good for creative writing

**💡 Sweet Spot for Most Users**

---

#### 🔷 Large Models (13B - 34B)
**RAM Required (Q4):** 8-20 GB
**Speed:** Slow
**Quality:** Excellent

**Best for:**
- Advanced reasoning and problem-solving
- Professional code generation
- Complex analysis tasks
- High-quality content creation
- Research and technical writing

**Recommended Models:**
- Phi-3 Medium 14B - Excellent reasoning
- Qwen 3 14B - Strong multilingual performance
- CodeLlama 13B - Professional coding
- DeepSeek R1 14B - Deep reasoning tasks
- Mixtral 8x7B (30B effective) - Best-in-class for size
- CodeLlama 34B - Advanced code generation

---

#### 🔴 Extra Large Models (70B+)
**RAM Required (Q4):** 40+ GB
**Speed:** Very Slow
**Quality:** Outstanding

**Best for:**
- Research and academic work
- Production-level applications
- Maximum accuracy requirements
- Complex multi-step reasoning
- Professional use cases

**Recommended Models:**
- Llama 3.1 70B - Top-tier general purpose
- Qwen 3 70B - Best multilingual large model
- DeepSeek R1 70B - Advanced reasoning capabilities
- Llama 3.1 405B - State-of-the-art (220GB Q4)

---

### Quick Decision Guide

| Your Situation | Recommended Size | Example Models |
|----------------|------------------|----------------|
| Learning/Experimenting | 3B-7B | Llama 3.2 3B, Qwen 2.5 7B |
| Daily coding helper | 7B-8B | Llama 3.1 8B, DeepSeek Coder 6.7B |
| Professional work | 13B-34B | Mixtral 8x7B, CodeLlama 34B |
| Research/Production | 70B+ | Llama 3.1 70B, Qwen 3 70B |
| Limited RAM (<8GB) | 1B-3B | Qwen 3 1.7B, Llama 3.2 3B |
| Moderate RAM (8-16GB) | 3B-8B | Llama 3.1 8B, Phi-3 Mini |
| Good RAM (16-32GB) | 7B-14B | Mixtral 8x7B, Phi-3 Medium |
| Lots of RAM (32-64GB) | 14B-34B | CodeLlama 34B, DeepSeek R1 32B |
| Massive RAM (64GB+) | 70B+ | Llama 3.1 70B, Qwen 3 70B |

### Performance vs Quality Tradeoff

```
Quality ↑
    │
    │                                    ● 70B+
    │                           ● 34B
    │                    ● 14B
    │              ● 8B (Sweet Spot)
    │         ● 3B
    │    ● 1B
    │
    └──────────────────────────────────────→ Speed
```

**Pro Tip:** Start with 7B-8B models (like Llama 3.1 8B) - they offer the best balance of quality, speed, and resource usage for most tasks!

## Notes

- Apple Silicon Macs benefit from unified memory architecture and Metal API acceleration
- The llm_checker tool assumes ~70% of total RAM is safely available for LLMs on Apple Silicon
- Intel Macs may have slower inference speeds but can still run many models
- Ollama benchmark requires Ollama to be installed and running (default port: 11434)
- First run may take longer as models need to be downloaded
- Performance varies based on hardware, model quantization, and system load
