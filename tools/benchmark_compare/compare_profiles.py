#!/usr/bin/env python3
"""
Compare two Go pprof CPU profiles and identify performance regressions.

Usage:
    python compare_profiles.py <old_profile.pb.gz> <new_profile.pb.gz> [options]

Options:
    --threshold PERCENT   Regression threshold percentage (default: 30)
    --min-delta MS        Minimum delta in ms to report (default: 50)
    --exclude PATTERN     Exclude functions matching pattern (repeatable)
    --top N               Number of top entries per category (default: 20)
    --nodecount N         Number of nodes to extract from pprof (default: 300)
    --output FILE         Write results to file instead of stdout

Examples:
    python compare_profiles.py old.pb.gz new.pb.gz
    python compare_profiles.py old.pb.gz new.pb.gz --threshold 20 --min-delta 100
    python compare_profiles.py old.pb.gz new.pb.gz --exclude "vector.*Path" --exclude "guioverworld"
    python compare_profiles.py old.pb.gz new.pb.gz --output results.txt
"""

import subprocess
import re
import sys
import argparse
from collections import defaultdict


def parse_args():
    parser = argparse.ArgumentParser(
        description="Compare two Go pprof CPU profiles for regressions"
    )
    parser.add_argument("old_profile", help="Path to the old/baseline .pb.gz profile")
    parser.add_argument("new_profile", help="Path to the new .pb.gz profile")
    parser.add_argument(
        "--threshold",
        type=float,
        default=30.0,
        help="Regression threshold percentage (default: 30)",
    )
    parser.add_argument(
        "--min-delta",
        type=float,
        default=50.0,
        help="Minimum delta in ms to report (default: 50)",
    )
    parser.add_argument(
        "--exclude",
        action="append",
        default=[],
        help="Exclude functions matching pattern (repeatable)",
    )
    parser.add_argument(
        "--top",
        type=int,
        default=20,
        help="Number of top entries per category (default: 20)",
    )
    parser.add_argument(
        "--nodecount",
        type=int,
        default=300,
        help="Number of nodes to extract from pprof (default: 300)",
    )
    parser.add_argument(
        "--output", type=str, default=None, help="Write results to file"
    )
    return parser.parse_args()


def run_pprof(profile_path, nodecount=300):
    """Run go tool pprof and return raw output."""
    cmd = [
        "go",
        "tool",
        "pprof",
        "-top",
        f"-nodecount={nodecount}",
        profile_path,
    ]
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
    if result.returncode != 0:
        print(f"Error running pprof on {profile_path}:", file=sys.stderr)
        print(result.stderr, file=sys.stderr)
        sys.exit(1)
    return result.stdout


def parse_pprof_output(output):
    """Parse pprof -top output into a dict of {function_name: (flat_ms, cum_ms)}."""
    functions = {}
    total_samples = None

    # Extract total samples from header
    total_match = re.search(r"Total samples\s*=\s*([\d.]+)(ms|s)", output)
    if total_match:
        val = float(total_match.group(1))
        unit = total_match.group(2)
        total_samples = val if unit == "ms" else val * 1000

    # Extract duration
    duration_match = re.search(r"Duration:\s*([\d.]+)s", output)
    duration = float(duration_match.group(1)) if duration_match else None

    # Parse each function line
    # Format: flat flat% sum% cum cum% function_name
    # Example: 16141ms 37.92% 37.92% 16515ms 38.80%  runtime.cgocall
    pattern = re.compile(
        r"^\s*([\d.]+)(ms|s)\s+[\d.]+%\s+[\d.]+%\s+([\d.]+)(ms|s)\s+[\d.]+%\s+(.+)$"
    )

    for line in output.split("\n"):
        m = pattern.match(line)
        if m:
            flat_val = float(m.group(1))
            flat_unit = m.group(2)
            cum_val = float(m.group(3))
            cum_unit = m.group(4)
            func_name = m.group(5).strip()

            flat_ms = flat_val if flat_unit == "ms" else flat_val * 1000
            cum_ms = cum_val if cum_unit == "ms" else cum_val * 1000

            functions[func_name] = (flat_ms, cum_ms)

    return functions, total_samples, duration


def should_exclude(func_name, exclude_patterns):
    """Check if function matches any exclusion pattern."""
    for pattern in exclude_patterns:
        if re.search(pattern, func_name):
            return True
    return False


def categorize_function(func_name):
    """Categorize a function into a group."""
    if func_name.startswith("game_main/"):
        return "YOUR_CODE"
    elif func_name.startswith("runtime."):
        return "RUNTIME"
    elif "ebiten" in func_name or "ebitenui" in func_name:
        return "EBITEN"
    elif "ecs" in func_name:
        return "ECS"
    else:
        return "OTHER"


def format_ms(ms):
    """Format milliseconds for display."""
    if ms >= 1000:
        return f"{ms:,.0f}ms"
    return f"{ms:.0f}ms"


def format_change(old_val, new_val):
    """Format percentage change."""
    if old_val == 0:
        if new_val == 0:
            return "0%"
        return "+NEW"
    pct = ((new_val - old_val) / old_val) * 100
    sign = "+" if pct > 0 else ""
    return f"{sign}{pct:.0f}%"


def shorten_func(name, max_len=55):
    """Shorten function name for display."""
    # Remove common prefixes
    name = name.replace("game_main/", "")
    name = name.replace("github.com/hajimehoshi/ebiten/v2/", "ebiten/")
    name = name.replace("github.com/ebitenui/ebitenui/", "ebitenui/")
    name = name.replace("github.com/bytearena/ecs.", "ecs.")
    name = name.replace("github.com/golang/freetype/", "freetype/")
    name = name.replace("golang.org/x/sys/windows.", "windows.")

    if len(name) > max_len:
        name = name[: max_len - 3] + "..."
    return name


def print_table(headers, rows, out):
    """Print a formatted table."""
    if not rows:
        out.write("  (none)\n")
        return

    col_widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            col_widths[i] = max(col_widths[i], len(str(cell)))

    # Header
    header_line = " | ".join(h.ljust(col_widths[i]) for i, h in enumerate(headers))
    out.write(f"  {header_line}\n")
    sep_line = "-+-".join("-" * col_widths[i] for i in range(len(headers)))
    out.write(f"  {sep_line}\n")

    # Rows
    for row in rows:
        row_line = " | ".join(
            str(cell).ljust(col_widths[i]) for i, cell in enumerate(row)
        )
        out.write(f"  {row_line}\n")


def main():
    args = parse_args()

    # Extract profiles
    print(f"Extracting profile: {args.old_profile} ...", file=sys.stderr)
    old_output = run_pprof(args.old_profile, args.nodecount)
    old_funcs, old_total, old_duration = parse_pprof_output(old_output)

    print(f"Extracting profile: {args.new_profile} ...", file=sys.stderr)
    new_output = run_pprof(args.new_profile, args.nodecount)
    new_funcs, new_total, new_duration = parse_pprof_output(new_output)

    # Find common functions (excluding patterns)
    common_names = set(old_funcs.keys()) & set(new_funcs.keys())
    excluded = set()
    for name in list(common_names):
        if should_exclude(name, args.exclude):
            excluded.add(name)
            common_names.discard(name)

    # Build comparison data
    regressions = []  # (func, category, old_flat, new_flat, flat_delta, flat_pct, old_cum, new_cum, cum_delta, cum_pct)
    improvements = []
    stable = []

    for name in common_names:
        old_flat, old_cum = old_funcs[name]
        new_flat, new_cum = new_funcs[name]

        flat_delta = new_flat - old_flat
        cum_delta = new_cum - old_cum

        flat_pct = ((flat_delta / old_flat) * 100) if old_flat > 0 else (100 if new_flat > 0 else 0)
        cum_pct = ((cum_delta / old_cum) * 100) if old_cum > 0 else (100 if new_cum > 0 else 0)

        category = categorize_function(name)
        entry = (name, category, old_flat, new_flat, flat_delta, flat_pct, old_cum, new_cum, cum_delta, cum_pct)

        # Classify based on cumulative time change (primary) or flat time change
        is_regression = (
            (cum_pct > args.threshold and abs(cum_delta) > args.min_delta)
            or (flat_pct > args.threshold and abs(flat_delta) > args.min_delta)
        )
        is_improvement = (
            (cum_pct < -args.threshold and abs(cum_delta) > args.min_delta)
            or (flat_pct < -args.threshold and abs(flat_delta) > args.min_delta)
        )

        if is_regression:
            regressions.append(entry)
        elif is_improvement:
            improvements.append(entry)
        else:
            stable.append(entry)

    # Sort by cumulative delta descending
    regressions.sort(key=lambda x: x[8], reverse=True)
    improvements.sort(key=lambda x: x[8])

    # Output
    if args.output:
        out = open(args.output, "w", encoding="utf-8")
    else:
        # Force UTF-8 on Windows to support box-drawing characters
        out = open(sys.stdout.fileno(), "w", encoding="utf-8", closefd=False)

    out.write("=" * 80 + "\n")
    out.write("PPROF BENCHMARK COMPARISON\n")
    out.write("=" * 80 + "\n\n")

    out.write(f"Old profile: {args.old_profile}\n")
    out.write(f"New profile: {args.new_profile}\n")
    if old_duration:
        out.write(f"Old duration: {old_duration:.1f}s\n")
    if new_duration:
        out.write(f"New duration: {new_duration:.1f}s\n")
    if old_total and new_total:
        total_delta = new_total - old_total
        total_pct = (total_delta / old_total) * 100 if old_total else 0
        out.write(f"Old total samples: {format_ms(old_total)}\n")
        out.write(f"New total samples: {format_ms(new_total)}\n")
        out.write(f"Total sample delta: {format_ms(total_delta)} ({format_change(old_total, new_total)})\n")

    out.write(f"\nCommon functions compared: {len(common_names)}\n")
    out.write(f"Excluded by patterns: {len(excluded)}\n")
    out.write(f"Regression threshold: >{args.threshold}% and >{args.min_delta}ms delta\n")
    out.write(f"Regressions found: {len(regressions)}\n")
    out.write(f"Improvements found: {len(improvements)}\n")
    out.write(f"Stable: {len(stable)}\n")

    # --- Regressions by category ---
    categories = ["YOUR_CODE", "ECS", "RUNTIME", "EBITEN", "OTHER"]
    for cat in categories:
        cat_regressions = [r for r in regressions if r[1] == cat]
        if not cat_regressions:
            continue

        out.write(f"\n{'─' * 80}\n")
        out.write(f"REGRESSIONS: {cat} (sorted by cumulative delta)\n")
        out.write(f"{'─' * 80}\n")

        headers = ["Function", "Old Flat", "New Flat", "Flat Chg", "Old Cum", "New Cum", "Cum Chg"]
        rows = []
        for entry in cat_regressions[: args.top]:
            name, _, old_flat, new_flat, flat_delta, flat_pct, old_cum, new_cum, cum_delta, cum_pct = entry
            rows.append([
                shorten_func(name),
                format_ms(old_flat),
                format_ms(new_flat),
                format_change(old_flat, new_flat),
                format_ms(old_cum),
                format_ms(new_cum),
                format_change(old_cum, new_cum),
            ])
        print_table(headers, rows, out)

    # --- Improvements by category ---
    has_improvements = False
    for cat in categories:
        cat_improvements = [r for r in improvements if r[1] == cat]
        if not cat_improvements:
            continue
        if not has_improvements:
            out.write(f"\n{'─' * 80}\n")
            out.write(f"IMPROVEMENTS (sorted by cumulative delta)\n")
            out.write(f"{'─' * 80}\n")
            has_improvements = True

        out.write(f"\n  [{cat}]\n")
        headers = ["Function", "Old Flat", "New Flat", "Flat Chg", "Old Cum", "New Cum", "Cum Chg"]
        rows = []
        for entry in cat_improvements[: args.top]:
            name, _, old_flat, new_flat, flat_delta, flat_pct, old_cum, new_cum, cum_delta, cum_pct = entry
            rows.append([
                shorten_func(name),
                format_ms(old_flat),
                format_ms(new_flat),
                format_change(old_flat, new_flat),
                format_ms(old_cum),
                format_ms(new_cum),
                format_change(old_cum, new_cum),
            ])
        print_table(headers, rows, out)

    # --- Summary of new-only functions (in new but not old) ---
    new_only = set(new_funcs.keys()) - set(old_funcs.keys()) - excluded
    new_only_filtered = []
    for name in new_only:
        if should_exclude(name, args.exclude):
            continue
        _, cum = new_funcs[name]
        if cum > 200:  # Only show significant new functions
            new_only_filtered.append((name, new_funcs[name][0], cum))
    new_only_filtered.sort(key=lambda x: x[2], reverse=True)

    if new_only_filtered:
        out.write(f"\n{'─' * 80}\n")
        out.write(f"NEW FUNCTIONS (not in old profile, cum > 200ms)\n")
        out.write(f"{'─' * 80}\n")
        headers = ["Function", "Flat", "Cum"]
        rows = [(shorten_func(n), format_ms(f), format_ms(c)) for n, f, c in new_only_filtered[:args.top]]
        print_table(headers, rows, out)

    # --- Functions removed (in old but not new) ---
    old_only = set(old_funcs.keys()) - set(new_funcs.keys()) - excluded
    old_only_filtered = []
    for name in old_only:
        if should_exclude(name, args.exclude):
            continue
        _, cum = old_funcs[name]
        if cum > 200:
            old_only_filtered.append((name, old_funcs[name][0], cum))
    old_only_filtered.sort(key=lambda x: x[2], reverse=True)

    if old_only_filtered:
        out.write(f"\n{'─' * 80}\n")
        out.write(f"REMOVED FUNCTIONS (in old profile only, cum > 200ms)\n")
        out.write(f"{'─' * 80}\n")
        headers = ["Function", "Flat", "Cum"]
        rows = [(shorten_func(n), format_ms(f), format_ms(c)) for n, f, c in old_only_filtered[:args.top]]
        print_table(headers, rows, out)

    out.write(f"\n{'=' * 80}\n")

    if args.output:
        out.close()
        print(f"Results written to {args.output}", file=sys.stderr)


if __name__ == "__main__":
    main()
