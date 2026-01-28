# Tutorial: Multi-Run Comparison and Configuration Selection

**Learn how to systematically compare multiple workflow runs to find optimal configurations**

This tutorial demonstrates how to run multiple configurations in parallel, compare results systematically, and use AI-powered analysis to identify the best settings for your use case.

**Duration:** 25-35 minutes  
**Prerequisites:**
- Arcaflow MCP server installed
- Arcaflow Engine installed
- Completed [Iterative Optimization Tutorial](iterative-optimization.md)

**What you'll learn:**
- How to design comparison experiments
- How to run multiple configurations systematically
- How to compare results across dimensions
- How to rank configurations by multiple criteria
- How to make data-driven configuration decisions

---

## Overview: Systematic Comparison

Instead of iterating one parameter at a time, you can:
1. Define multiple configurations to test
2. Run all configurations
3. Compare results systematically
4. Rank by your success criteria
5. Select the optimal configuration

This approach is ideal when:
- You have multiple candidate configurations
- You want to test parameter combinations
- You need to optimize for multiple goals
- You want reproducible comparison data

---

## Scenario: Finding Optimal Load Test Configuration

**Goal:** Find the configuration that maximizes throughput while maintaining:
- Success rate ≥ 95%
- Average response time ≤ 350ms
- P95 response time ≤ 550ms

**Approach:** Test three configurations (low, medium, high) and compare.

---

## Step 1: Design the Experiment

### Define Configurations to Test

**You:**
```
I want to compare three load test configurations:
1. Conservative: 15 users, 5s timeout
2. Balanced: 25 users, 6s timeout  
3. Aggressive: 35 users, 7s timeout

All with 60s duration and 10s ramp-up. Help me set these up.
```

**AI:**
```
I'll help you create three configuration files for comparison.

**Configuration Matrix:**

| Config | Users | Timeout | Expected Behavior |
|--------|-------|---------|-------------------|
| Conservative | 15 | 5s | Safe, low load |
| Balanced | 25 | 6s | Moderate load |
| Aggressive | 35 | 7s | High load, may exceed capacity |

Let me build inputs for each:
```

---

## Step 2: Build Configuration Inputs

### Configuration 1: Conservative

**AI (using `workflow_input_build`):**
```
Created: perf-test-conservative.yaml

```yaml
test_config:
  name: "Conservative Load"
  users: 15
  duration: 60
  ramp_up: 10
  timeout: 5
  success_threshold: 0.95
```
```

### Configuration 2: Balanced

```yaml
test_config:
  name: "Balanced Load"
  users: 25
  duration: 60
  ramp_up: 10
  timeout: 6
  success_threshold: 0.95
```

### Configuration 3: Aggressive

```yaml
test_config:
  name: "Aggressive Load"
  users: 35
  duration: 60
  ramp_up: 10
  timeout: 7
  success_threshold: 0.95
```

---

## Step 3: Execute All Configurations

Run each configuration with Arcaflow Engine:

```bash
# Run 1: Conservative
arcaflow \
  -input perf-test-conservative.yaml \
  -workflow /path/to/perf-test/workflow.yaml \
  -output results-conservative.yaml

# Run 2: Balanced
arcaflow \
  -input perf-test-balanced.yaml \
  -workflow /path/to/perf-test/workflow.yaml \
  -output results-balanced.yaml

# Run 3: Aggressive
arcaflow \
  -input perf-test-aggressive.yaml \
  -workflow /path/to/perf-test/workflow.yaml \
  -output results-aggressive.yaml
```

**Tip:** For production testing, run these sequentially or on separate systems to avoid resource contention.

---

## Step 4: Load and Compare Results

### Initial Comparison

**You:**
```
Load all three result files and compare them:
- results-conservative.yaml
- results-balanced.yaml
- results-aggressive.yaml

Show me which meets the goals: success ≥ 0.95, avg response ≤ 350ms, P95 ≤ 550ms
```

**AI (using `workflow_results_load` and `workflow_results_compare`):**
```
📊 Multi-Configuration Comparison

**Configuration: Conservative (15 users)**
- Total requests: 900
- Success rate: 1.00 ✓ (Goal: ≥ 0.95)
- Avg response: 110ms ✓ (Goal: ≤ 350ms)
- P95 response: 165ms ✓ (Goal: ≤ 550ms)
- P99 response: 195ms
- Throughput: 900 req/min

**Configuration: Balanced (25 users)**
- Total requests: 1500
- Success rate: 0.97 ✓ (Goal: ≥ 0.95)
- Avg response: 285ms ✓ (Goal: ≤ 350ms)
- P95 response: 475ms ✓ (Goal: ≤ 550ms)
- P99 response: 590ms
- Throughput: 1500 req/min

**Configuration: Aggressive (35 users)**
- Total requests: 1890
- Success rate: 0.91 ✗ (Goal: ≥ 0.95)
- Avg response: 425ms ✗ (Goal: ≤ 350ms)
- P95 response: 725ms ✗ (Goal: ≤ 550ms)
- P99 response: 980ms
- Throughput: 1890 req/min

**Summary:**
✓ Conservative: All goals met, low throughput
✓ Balanced: All goals met, good throughput
✗ Aggressive: Goals not met, highest throughput (but unreliable)
```

---

## Step 5: Detailed Analysis

### Compare by Metrics

**You:**
```
Show me a detailed comparison focusing on the trade-offs between 
throughput and response time
```

**AI:**
```
📊 Trade-off Analysis: Throughput vs Response Time

**Throughput Ranking:**
1. Aggressive: 1890 req/min (+26% vs Balanced)
2. Balanced: 1500 req/min (+67% vs Conservative)
3. Conservative: 900 req/min (baseline)

**Response Time Ranking (Lower is better):**
1. Conservative: Avg 110ms, P95 165ms
2. Balanced: Avg 285ms (+159%), P95 475ms (+188%)
3. Aggressive: Avg 425ms (+286%), P95 725ms (+339%)

**Reliability Ranking:**
1. Conservative: 100% success (perfect)
2. Balanced: 97% success (acceptable)
3. Aggressive: 91% success (unacceptable - below 95% threshold)

**Trade-off Insights:**

🎯 **Balanced Configuration** offers the best compromise:
- **60% throughput increase** vs Conservative (900 → 1500 req/min)
- Response times still well within goals
- 97% success rate (exceeds 95% threshold)
- Headroom for traffic spikes

⚠️ **Aggressive Configuration** exceeds system capacity:
- **27% more throughput** vs Balanced
- BUT: Fails reliability threshold
- Response times exceed all goals
- Not suitable for production

💡 **Conservative Configuration** is over-provisioned:
- Perfect reliability and fast responses
- But **40% lower throughput** than Balanced
- Wastes capacity
- Use only if ultra-low latency required

**Recommendation:** 
Deploy with Balanced configuration (25 users, 6s timeout) for optimal 
throughput while meeting all reliability and performance goals.
```

---

## Step 6: Multi-Criteria Ranking

### Custom Scoring

**You:**
```
Rank the configurations using a weighted score:
- Throughput: 40% weight (higher is better)
- Success rate: 35% weight (higher is better)
- Response time: 25% weight (lower is better)
```

**AI:**
```
📊 Weighted Score Ranking

**Scoring Model:**
- Throughput score: (actual / max) × 40
- Success score: (actual / 1.0) × 35
- Response score: (min / actual) × 25
- Total: Sum of component scores (max 100)

**Results:**

**1. Balanced Configuration - Score: 89.4** ✓ Winner
- Throughput: 1500/1890 × 40 = 31.7
- Success: 0.97/1.0 × 35 = 33.95
- Response: 110/285 × 25 = 9.65
- **Excellent balance across all dimensions**

**2. Conservative Configuration - Score: 84.8**
- Throughput: 900/1890 × 40 = 19.0
- Success: 1.0/1.0 × 35 = 35.0
- Response: 110/110 × 25 = 25.0
- **Perfect reliability but lower throughput**

**3. Aggressive Configuration - Score: 67.1**
- Throughput: 1890/1890 × 40 = 40.0
- Success: 0.91/1.0 × 35 = 31.85
- Response: 110/425 × 25 = 6.47
- **High throughput but poor reliability/response**

**Interpretation:**
Balanced wins due to strong performance across all criteria. 
Conservative excels at reliability but sacrifices throughput.
Aggressive maximizes throughput at the cost of meeting goals.

**Sensitivity Analysis:**
If you weighed throughput at 60% instead of 40%, Aggressive would 
score higher but still fail absolute thresholds (success < 0.95).

**Recommendation:**
Stick with Balanced (25 users, 6s timeout) as it optimizes the 
weighted criteria while meeting all hard constraints.
```

---

## Step 7: Visualize Trends

### Request Visualization

**You:**
```
Show me how response time and success rate trend as we increase load from 
15 to 25 to 35 users
```

**AI:**
```
📊 Load Impact Trends

**Response Time Trend:**
```
Users:        15      25      35
Avg (ms):    110 ━━  285 ━━━━━  425 ━━━━━━━━
P95 (ms):    165 ━━  475 ━━━━━━━  725 ━━━━━━━━━━━━
                 ⬆        ⬆
              +159%    +291%
              vs 15    vs 15
```

**Success Rate Trend:**
```
Users:        15      25      35
Success:    1.00 ████ 0.97 ███▓ 0.91 ███░
            100%      97%      91%
                      ⬇-3%     ⬇-9%
```

**Throughput Trend:**
```
Users:        15      25      35
Req/min:     900 ████ 1500 ██████▓ 1890 ████████░
                 +67%          +110%
```

**Analysis:**
- **Linear degradation**: Response time increases roughly linearly with load
- **Capacity threshold**: Success rate drops sharply beyond 30 users
- **Diminishing returns**: Throughput gain from 25→35 users (+26%) comes 
  at significant cost to reliability
  
**Sweet Spot:** 20-25 users balances throughput with reliability
```

---

## Step 8: Make the Decision

### Final Recommendation

**AI:**
```
🎯 Configuration Decision Matrix

**For Your Use Case** (maximize throughput, maintain goals):

**Recommendation: Balanced Configuration** (25 users, 6s timeout)

**Rationale:**
✓ Meets all hard requirements (success ≥ 95%, response times within limits)
✓ Maximizes throughput within constraints (1500 req/min)
✓ 67% throughput improvement vs safe Conservative config
✓ Maintains reliability headroom (97% vs 95% threshold)
✓ Best weighted score across all criteria

**Alternative Scenarios:**

**If ultra-reliability is critical** (e.g., financial transactions):
→ Use Conservative (15 users, 5s timeout)
- Perfect 100% success rate
- Fastest response times
- Trade-off: 40% lower throughput

**If you improve backend capacity** (add resources, optimize code):
→ Re-test Aggressive (35 users, 7s timeout)
- Could unlock higher throughput
- Need to meet reliability threshold first

**Production Configuration:**
```yaml
test_config:
  name: "Production Load"
  users: 25
  duration: 60
  ramp_up: 10
  timeout: 6
  success_threshold: 0.95
```

**Monitoring:** Watch P99 response times (currently 590ms) as they're 
approaching limits. If P99 consistently exceeds 600ms, consider scaling 
back to 20 users or scaling up backend resources.
```

---

## Advanced Comparison Techniques

### Multi-Dimensional Comparison

Test multiple parameters simultaneously:

**Example Matrix:**

| Config | Users | Timeout | Duration | Ramp-up |
|--------|-------|---------|----------|---------|
| A | 20 | 5s | 60s | 5s |
| B | 20 | 6s | 60s | 5s |
| C | 20 | 5s | 90s | 10s |
| D | 25 | 6s | 60s | 10s |

**You:**
```
Compare configurations A, B, C, and D to understand the impact of 
timeout duration vs ramp-up time
```

**AI:**
```
📊 Parameter Sensitivity Analysis

**Isolating Timeout Impact** (A vs B - only timeout changes):
- Config A (5s timeout): Success 0.94, Avg 295ms
- Config B (6s timeout): Success 0.98, Avg 290ms
- **Insight:** Increasing timeout by 1s improves success +4% without 
  impacting response time

**Isolating Ramp-up Impact** (A vs C - ramp-up and duration):
- Config A (60s, 5s ramp): Success 0.94
- Config C (90s, 10s ramp): Success 0.96
- **Insight:** Longer ramp-up smooths load distribution, improving stability

**Combined Effect** (D - optimal combo):
- Config D (25 users, 6s timeout, 10s ramp): Success 0.97
- **Best performer** with balanced parameters

**Recommendation:**
- Timeout: 6s provides better reliability for minimal cost
- Ramp-up: 10s worth the smoothing effect for production loads
```

### Statistical Significance

For production decisions, run multiple iterations:

**You:**
```
I ran each configuration 5 times. Analyze statistical significance of the differences.
```

**AI:**
```
📊 Statistical Analysis (5 runs per config)

**Conservative Configuration:**
- Success rate: 0.998 ± 0.002 (99.8% ± 0.2%)
- Avg response: 112ms ± 8ms
- **Stability:** Excellent (low variance)

**Balanced Configuration:**
- Success rate: 0.965 ± 0.015 (96.5% ± 1.5%)
- Avg response: 289ms ± 22ms
- **Stability:** Good (acceptable variance)

**Aggressive Configuration:**
- Success rate: 0.913 ± 0.034 (91.3% ± 3.4%)
- Avg response: 431ms ± 47ms
- **Stability:** Poor (high variance, unreliable)

**Statistical Significance:**
- Balanced vs Conservative: Difference is significant (p < 0.01)
- Balanced vs Aggressive: Difference is significant (p < 0.01)

**Confidence Intervals (95%):**
- Conservative success: [99.6%, 100%]
- Balanced success: [95.0%, 98.0%] ← Lowest CI above 95% threshold ✓
- Aggressive success: [88.0%, 94.6%] ← CI below threshold ✗

**Conclusion:**
Balanced configuration reliably meets the 95% success threshold across 
repeated runs. Aggressive occasionally dips below threshold.
```

---

## Best Practices

### Design Effective Comparisons

**1. Control Variables**
- Change one parameter at a time when isolating effects
- Use factorial designs for parameter interactions
- Run controls (baseline) for reference

**2. Define Success Criteria Upfront**
- Hard constraints (must meet)
- Soft goals (nice to have)
- Weighting for trade-offs

**3. Collect Sufficient Data**
- Multiple runs for statistical validity
- Consistent test conditions
- Document environmental factors

**4. Document Everything**
- Configuration parameters
- Test conditions and environment
- Results and analysis
- Decision rationale

### Conversation Patterns

**Quick Comparison:**
```
Compare these three result files and tell me which is best
```

**Detailed Analysis:**
```
Compare configurations A, B, and C across all metrics, 
showing trade-offs and ranking by [criteria]
```

**Sensitivity Analysis:**
```
Show me how [metric] changes as I vary [parameter] from X to Y
```

**Decision Support:**
```
Which configuration should I use for [use case] given [constraints]?
```

---

## Next Steps

**Apply to Your Workflows:**
- Design comparison experiments for your use cases
- Define your specific success criteria
- Run systematic comparisons
- Document optimal configurations

**Advanced Topics:**
- **A/B Testing**: Compare production vs candidate configurations
- **Regression Testing**: Track performance over time
- **Multi-Objective Optimization**: Balance multiple competing goals

**Team Collaboration:**
- Deploy in [Server Mode](../usage/server-mode.md) for shared analysis
- Store comparison data for team reference
- Build organizational knowledge base of optimal configurations

---

## Related Documentation

- **[Iterative Optimization Tutorial](iterative-optimization.md)** - Learn iterative refinement
- **[Result Analysis Guide](../usage/result-analysis.md)** - Complete guide to analysis features
- **[Tool Reference: Result Tools](../tools/result-tools.md)** - All comparison and analysis tools
- **[Server Mode](../usage/server-mode.md)** - Deploy for team access

---

## Troubleshooting

**"Can't compare results - format mismatch"**
- Ensure all results are from the same workflow version
- Check that result files contain compatible metrics
- Verify JSON/YAML structure is valid

**"Comparison doesn't show expected differences"**
- Ensure parameter variations are significant enough
- Check for test environment consistency
- Look for confounding variables (time of day, system load)

**"Too many configurations to compare"**
- Start with fewer configs (3-5)
- Use factorial designs to reduce combinations
- Filter to most promising candidates first

---

**Questions or issues?** Check the [FAQ](../faq.md) or [Troubleshooting Guide](../troubleshooting.md).
