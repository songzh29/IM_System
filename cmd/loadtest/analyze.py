#!/usr/bin/env python3
import json
import sys
import statistics

if len(sys.argv) < 2:
    print("usage: analyze.py <log_file>")
    sys.exit(1)

segments = {
    'search_user_ms': [],
    'search_or_create_conv_ms': [],
    'create_msg_ms': [],
    'mq_publish_ms': [],
    'update_self_read_ms': [],
    'deliver_total_ms': [],
    'total_ms': [],
}

with open(sys.argv[1]) as f:
    for line in f:
        if '消息处理耗时' not in line:
            continue
        try:
            obj = json.loads(line.split('\t', 4)[-1])
            for k in segments:
                if k in obj:
                    segments[k].append(float(obj[k]))
        except:
            continue

print(f"\n样本数: {len(segments['total_ms'])}\n")
print(f"{'段':<25} {'P50':>8} {'P95':>8} {'P99':>8} {'Max':>8}  (单位 ms)")
print('-' * 70)
for name, vals in segments.items():
    if not vals:
        continue
    vals.sort()
    n = len(vals)
    p50 = vals[n*50//100]
    p95 = vals[n*95//100]
    p99 = vals[n*99//100]
    mx = vals[-1]
    print(f"{name:<25} {p50:>8.2f} {p95:>8.2f} {p99:>8.2f} {mx:>8.2f}")