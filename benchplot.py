#!/usr/bin/env python3
import argparse
import os
import re
import sys
import pandas as pd
import matplotlib.pyplot as plt

# Regexes for metrics in "go test -bench" output
RE_TIME = re.compile(r'(?P<v>[\d.]+)\s*(?P<u>ns|us|µs|μs|ms|s)/op')
RE_BYTES_OP = re.compile(r'(?P<v>[\d.]+)\s*B/op')
RE_THROUGHPUT = re.compile(r'(?P<v>[\d.]+)\s*(?P<u>[kKmMgG]?B)/s')

UNIT_TIME = {'ns': 1e-9, 'us': 1e-6, 'µs': 1e-6, 'μs': 1e-6, 'ms': 1e-3, 's': 1.0}
UNIT_BYTES_SEC = {'B': 1.0, 'kB': 1e3, 'KB': 1e3, 'MB': 1e6, 'GB': 1e9}  # decimal; constant cancels in ratios

def parse_line(line):
    line = line.strip()
    if not line.startswith('Benchmark'):
        return None

    # Token 0 is like "BenchmarkRPC/sequential/nop/gocapnp-6"
    t0 = line.split()[0]
    name = t0[len('Benchmark'):]
    parts = name.split('/')
    if len(parts) < 4:
        return None

    kind = parts[1]           # sequential | parallel
    workload = parts[2]       # nop | tree | hex | ...
    last = parts[3]           # e.g., tcp-6
    system = last.split('-')[0]

    # Find metrics anywhere in the line
    sec_per_op = None
    m_time = RE_TIME.search(line)
    if m_time:
        sec_per_op = float(m_time.group('v')) * UNIT_TIME[m_time.group('u')]

    bytes_per_op = None
    m_bop = RE_BYTES_OP.search(line)
    if m_bop:
        bytes_per_op = float(m_bop.group('v'))

    bytes_per_sec = None
    m_thr = RE_THROUGHPUT.search(line)
    if m_thr:
        u = m_thr.group('u')
        # normalize unit key to canonical form (kB, MB, GB, B)
        if u.lower() == 'kb':
            ukey = 'kB'
        elif u.lower() == 'mb':
            ukey = 'MB'
        elif u.lower() == 'gb':
            ukey = 'GB'
        else:
            ukey = 'B'
        bytes_per_sec = float(m_thr.group('v')) * UNIT_BYTES_SEC[ukey]

    return {
        'name': name,
        'kind': kind,
        'workload': workload,
        'system': system,
        'sec/op': sec_per_op,
        'bytes/op': bytes_per_op,
        'bytes/sec': bytes_per_sec,
    }

def read_bench(stream):
    rows = []
    for line in stream:
        r = parse_line(line)
        if r:
            rows.append(r)
    if not rows:
        sys.exit("No benchmark lines found (lines must start with 'Benchmark').")
    return pd.DataFrame(rows)

def plot_relative(df, metric, title, ylabel, outfile, kind_filter, lower_better, logy):
    subset = df[df['kind'] == kind_filter].copy()
    if subset.empty:
        print(f"Warning: no rows for kind={kind_filter}")
        return

    # Pivot: workloads × systems, aggregate duplicates by mean
    pivot = subset.pivot_table(index='workload', columns='system', values=metric, aggfunc='mean')

    if 'tcp' not in pivot.columns:
        print(f"Warning: missing tcp baseline for {title}")
        return

    baseline = pivot['tcp']
    valid = baseline > 0
    relative = pivot[valid].divide(baseline[valid], axis=0)

    if relative.empty:
        print(f"Warning: no valid workloads (tcp baseline <= 0) for {title}")
        return

    # tcp first, then sort others by mean ratio (performance)
    cols_other = [c for c in relative.columns if c != 'tcp']
    if cols_other:
        means = relative[cols_other].mean(axis=0, skipna=True)
        order_others = means.sort_values(ascending=lower_better).index.tolist()
        cols = ['tcp'] + order_others
        relative = relative.reindex(columns=cols)
    else:
        relative = relative.reindex(columns=['tcp'])

    os.makedirs(os.path.dirname(outfile), exist_ok=True)
    plt.figure(figsize=(10, 6))
    ax = relative.plot(kind='bar', logy=logy)
    ax.axhline(1, color='gray', linestyle='--', alpha=0.7)
    ax.set_title(title)
    ax.set_ylabel(ylabel)
    ax.set_xlabel('Workload')
    ax.legend(title='System', bbox_to_anchor=(1.02, 1), loc='upper left')
    plt.xticks(rotation=45, ha='right')
    plt.tight_layout()
    plt.savefig(outfile, dpi=150)
    plt.close()
    print(f"Saved: {outfile}")

def main():
    ap = argparse.ArgumentParser(description="Plot Go benchmark results (relative to tcp) from go test -bench output.")
    ap.add_argument('--in', dest='infile', default='-',
                    help="input path (default: '-' for stdin). Example: go test -bench . -benchmem | benchplot.py --in -")
    ap.add_argument('--outdir', default='www', help='output directory (default: plots)')
    args = ap.parse_args()

    if args.infile == '-' or args.infile == '':
        df = read_bench(sys.stdin)
    else:
        with open(args.infile, 'r', encoding='utf-8') as f:
            df = read_bench(f)

    # 1) Sequential sec/op (lower is better)
    plot_relative(
        df, metric='sec/op',
        title='Sequential: sec/op',
        ylabel='ratio vs tcp (log scale)',
        outfile=os.path.join(args.outdir, 'nop-latency.png'),
        kind_filter='sequential',
        lower_better=True,
        logy=True,
    )

    # 2) Sequential bytes/op (lower is better)
    plot_relative(
        df, metric='bytes/op',
        title='Sequential: alloc bytes/op',
        ylabel='ratio vs tcp (log scale)',
        outfile=os.path.join(args.outdir, 'tree-mem.png'),
        kind_filter='sequential',
        lower_better=True,
        logy=True,
    )

    # 3) Parallel bytes/sec (higher is better)
    plot_relative(
        df, metric='bytes/sec',
        title='Parallel: bytes/sec',
        ylabel='ratio vs tcp (linear scale)',
        outfile=os.path.join(args.outdir, 'hex-throughput.png'),
        kind_filter='parallel',
        lower_better=False,
        logy=False,
    )

if __name__ == '__main__':
    main()
