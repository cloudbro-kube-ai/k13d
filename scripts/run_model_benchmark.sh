#!/bin/bash
# k13d AI Model Benchmark Script
# Tests multiple models against k8s-ai-bench tasks

set -e

# Configuration
API_ENDPOINT="https://youngjudell.hopto.org/api/v1"
API_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImMzY2UwNzE4LTNlOWItNGFhMy05MGVmLTAyYTBiZWE1MDUzNCIsImV4cCI6MTc4Njg4NjE4NSwianRpIjoiNzRmYmRlNTctZGVmZC00OTNlLWE1OTUtYWM0NWUzN2ZiM2I0In0.vRWcXbBOUXojLcuLNYSyY88s_6b-U7AcCARxJd52e0o"

SOLAR_ENDPOINT="https://api.upstage.ai/v1"
SOLAR_API_KEY="up_z13Pj76IBqhcMRIM2FAbdqYTzzGLi"

OUTPUT_DIR=".build/bench-results"
TASK_DIR="benchmarks/tasks"

# Models to test (selecting appropriate sizes)
MODELS=(
    "qwen3:8b"
    "gemma3:4b"
    "gemma3:27b"
    "gpt-oss:latest"
    "deepseek-r1:32b"
)

echo "============================================================"
echo "  k13d AI Model Benchmark"
echo "  Based on k8s-ai-bench methodology"
echo "============================================================"
echo ""
echo "API Endpoint: $API_ENDPOINT"
echo "Output Dir: $OUTPUT_DIR"
echo "Task Dir: $TASK_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build benchmark tool
echo "Building benchmark tool..."
go build -o k13d-bench ./cmd/bench/

# Test connection
echo "Testing API connection..."
curl -s "$API_ENDPOINT/models" -H "Authorization: Bearer $API_KEY" > /dev/null || {
    echo "ERROR: Cannot connect to API"
    exit 1
}
echo "API connection OK"
echo ""

# Run benchmarks for each model
for MODEL in "${MODELS[@]}"; do
    echo "============================================================"
    echo "Testing: $MODEL"
    echo "============================================================"

    ./k13d-bench run \
        --task-dir "$TASK_DIR" \
        --llm-provider openai \
        --llm-model "$MODEL" \
        --llm-endpoint "$API_ENDPOINT" \
        --llm-api-key "$API_KEY" \
        --output-dir "$OUTPUT_DIR/$MODEL" \
        --auto-approve \
        --timeout 5m 2>&1 | tee "$OUTPUT_DIR/${MODEL//[:\/]/_}.log"

    echo ""
done

# Test Solar Pro2
echo "============================================================"
echo "Testing: solar-pro2"
echo "============================================================"

./k13d-bench run \
    --task-dir "$TASK_DIR" \
    --llm-provider solar \
    --llm-model solar-pro2 \
    --llm-endpoint "$SOLAR_ENDPOINT" \
    --llm-api-key "$SOLAR_API_KEY" \
    --output-dir "$OUTPUT_DIR/solar-pro2" \
    --auto-approve \
    --timeout 5m 2>&1 | tee "$OUTPUT_DIR/solar-pro2.log"

echo ""
echo "============================================================"
echo "Analyzing results..."
echo "============================================================"

./k13d-bench analyze \
    --input-dir "$OUTPUT_DIR" \
    --output-format markdown \
    --output "BENCHMARK_RESULTS.md"

echo ""
echo "Results saved to: BENCHMARK_RESULTS.md"
echo "Done!"
