// Package ui renders the demo table with plain ANSI codes.
package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	reset = "\033[0m"
	dim   = "\033[2m"
	bold  = "\033[1m"
	red   = "\033[31m"
	green = "\033[32m"
	cyan  = "\033[36m"
)

// Row is one rendered headline.
type Row struct {
	Headline  string
	Expected  string // bullish | bearish | none
	Regex     string
	Clef      string
	PClef     float64
	Confirmed float64
	Latency   time.Duration
	Order     string // executor result or reason for no action
}

// Table prints rows as they arrive and a scoreboard at the end.
type Table struct {
	w     io.Writer
	rows  []Row
	color bool
}

// NewTable renders to w; color enables ANSI codes.
func NewTable(w io.Writer, color bool) *Table { return &Table{w: w, color: color} }

func (t *Table) c(code, s string) string {
	if !t.color {
		return s
	}
	return code + s + reset
}

// Header prints the column titles.
func (t *Table) Header() {
	fmt.Fprintln(t.w, t.c(bold, fmt.Sprintf("%-52s %-11s %-16s %7s  %s", "HEADLINE", "REGEX", "CLEF-FLASH", "LATENCY", "DECISION")))
	fmt.Fprintln(t.w, t.c(dim, strings.Repeat("─", 100)))
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// Add records and prints one row.
func (t *Table) Add(r Row) {
	t.rows = append(t.rows, r)
	fmt.Fprintf(t.w, "%s %s %s %7s  %s\n",
		pad(trunc(r.Headline, 52), 52),
		t.cell(r.Regex, 8, r.Regex == r.Expected),
		t.cell(fmt.Sprintf("%s %.2f", r.Clef, r.PClef), 13, r.Clef == r.Expected),
		fmt.Sprintf("%dms", r.Latency.Milliseconds()),
		t.c(dim, r.Order))
}

// cell pads plain text first, then colours it, so ANSI codes don't skew alignment.
func (t *Table) cell(text string, width int, ok bool) string {
	code, m := red, "❌"
	if ok {
		code, m = green, "✅"
	}
	return t.c(code, pad(text, width)) + " " + m
}

func pad(s string, width int) string {
	if n := width - len([]rune(s)); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// Summary prints accuracy and latency percentiles.
func (t *Table) Summary() {
	n := len(t.rows)
	if n == 0 {
		return
	}
	var rx, cl int
	lat := make([]time.Duration, 0, n)
	for _, r := range t.rows {
		if r.Regex == r.Expected {
			rx++
		}
		if r.Clef == r.Expected {
			cl++
		}
		lat = append(lat, r.Latency)
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	pct := func(p float64) time.Duration { return lat[int(float64(len(lat)-1)*p)] }
	fmt.Fprintln(t.w, t.c(dim, strings.Repeat("─", 100)))
	fmt.Fprintf(t.w, "%s  regex %d/%d   %s  clef-flash %d/%d   latency p50 %dms  p95 %dms\n",
		t.c(bold, "Accuracy"), rx, n, t.c(cyan, "│"), cl, n, pct(0.5).Milliseconds(), pct(0.95).Milliseconds())
}
