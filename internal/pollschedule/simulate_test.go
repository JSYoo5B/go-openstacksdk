package pollschedule

// This file measures poll schedules on simulated time; nothing sleeps and no
// HTTP is sent. Each simulated task finishes at an injected completion time
// drawn log-uniformly from its scenario's range, and every schedule is
// measured against the same tasks:
//
//   - detection delay: time from the real completion to the response of the
//     first poll that observes it (always >= the poll latency);
//   - polls: requests sent until completion is observed, initial lookup
//     included;
//   - herd peak: the most polls in any herdWindow when herdClients tasks of
//     the scenario start waiting at the same instant, initial lookups
//     excluded (no schedule can spread those). Without jitter every client
//     polls at the same instants, so the peak equals the clients still
//     waiting;
//   - pareto: no other jitter-free or jittered policy in the scenario has
//     both a lower mean delay and fewer mean polls.
//
// EqualJitter and FullJitter shorten the mean pause (to 3/4 and 1/2), so
// they trade more polls for less delay; CenteredJitter keeps the mean and
// isolates the spreading effect.
//
// Run the matrix with
//
//	go test -run '^$' -bench PollSchedule -benchtime 1x ./internal/pollschedule
//
// or print it as tables with
//
//	POLLSCHEDULE_REPORT=1 go test -run TestPollScheduleReport -v ./internal/pollschedule

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	samples     = 20_000
	herdClients = 1_000
	pollLatency = 150 * time.Millisecond
	herdWindow  = 100 * time.Millisecond
)

type scenario struct {
	name     string
	min, max time.Duration
	seed     uint64
}

var scenarios = []scenario{
	{"fast", time.Second, 15 * time.Second, 11},
	{"moderate", 30 * time.Second, 10 * time.Minute, 22},
	{"very-long", 20 * time.Minute, 3 * time.Hour, 33},
}

type policy struct {
	name string
	sch  Schedule
}

func policies() []policy {
	s := time.Second
	base := []policy{
		{"fixed-1s", Fixed(s)},
		{"fixed-2s", Fixed(2 * s)}, // Python openstacksdk's default wait interval
		{"fixed-5s", Fixed(5 * s)},
		{"linear-1s+1s<=30s", Linear(s, s, 30*s)},
		{"linear-2s+2s<=60s", Linear(2*s, 2*s, 60*s)},
		{"exp-1s*2<=30s", Exponential(s, 2, 30*s)},
		{"exp-1s*1.5<=60s", Exponential(s, 1.5, 60*s)},
		{"exp-2s*2<=120s", Exponential(2*s, 2, 120*s)},
	}
	var out []policy
	for _, p := range base {
		for _, j := range []Jitter{NoJitter, EqualJitter, FullJitter, CenteredJitter} {
			out = append(out, policy{p.name + "/jitter=" + j.String(), p.sch.WithJitter(j)})
		}
	}
	return out
}

// completion draws a log-uniform completion time in [min, max].
func (sc scenario) completion(rng *rand.Rand) time.Duration {
	lo, hi := math.Log(float64(sc.min)), math.Log(float64(sc.max))
	return time.Duration(math.Exp(lo + rng.Float64()*(hi-lo)))
}

// simulate polls a task that completes at done. A poll sent at t observes
// the task as complete when t >= done and its response arrives at
// t+latency; the next poll is sent after the response plus the pause.
// visit, when set, receives each poll's send time.
func simulate(s Schedule, done, latency time.Duration, rng *rand.Rand, visit func(time.Duration)) (delay time.Duration, polls int) {
	var t time.Duration
	for n := 1; ; n++ {
		polls++
		if visit != nil {
			visit(t)
		}
		if t >= done {
			return t + latency - done, polls
		}
		t += latency + s.Delay(n, rng)
	}
}

type stats struct {
	delayMean, delayP50, delayP95, delayMax float64 // seconds
	pollsMean, pollsP95                     float64
	herdPeak                                int
}

func percentile(sorted []float64, p float64) float64 {
	return sorted[int(math.Ceil(p*float64(len(sorted))))-1]
}

func measure(sc scenario, p policy) stats {
	tasks := rand.New(rand.NewPCG(sc.seed, 1))
	jitter := rand.New(rand.NewPCG(sc.seed, 2))
	delays := make([]float64, samples)
	polls := make([]float64, samples)
	var st stats
	for i := range samples {
		d, n := simulate(p.sch, sc.completion(tasks), pollLatency, jitter, nil)
		delays[i], polls[i] = d.Seconds(), float64(n)
		st.delayMean += d.Seconds()
		st.pollsMean += float64(n)
	}
	st.delayMean /= samples
	st.pollsMean /= samples
	slices.Sort(delays)
	slices.Sort(polls)
	st.delayP50, st.delayP95, st.delayMax = percentile(delays, .5), percentile(delays, .95), delays[samples-1]
	st.pollsP95 = percentile(polls, .95)

	herdTasks := rand.New(rand.NewPCG(sc.seed, 3))
	herdJitter := rand.New(rand.NewPCG(sc.seed, 4))
	perWindow := map[int64]int{}
	for range herdClients {
		simulate(p.sch, sc.completion(herdTasks), pollLatency, herdJitter, func(t time.Duration) {
			if t > 0 {
				perWindow[int64(t/herdWindow)]++
			}
		})
	}
	for _, n := range perWindow {
		st.herdPeak = max(st.herdPeak, n)
	}
	return st
}

var (
	measured   sync.Map // scenario/policy -> stats
	measureMux sync.Mutex
)

func cachedMeasure(sc scenario, p policy) stats {
	key := sc.name + "/" + p.name
	if v, ok := measured.Load(key); ok {
		return v.(stats)
	}
	measureMux.Lock()
	defer measureMux.Unlock()
	if v, ok := measured.Load(key); ok {
		return v.(stats)
	}
	st := measure(sc, p)
	measured.Store(key, st)
	return st
}

func BenchmarkPollSchedule(b *testing.B) {
	for _, sc := range scenarios {
		for _, p := range policies() {
			b.Run("scenario="+sc.name+"/policy="+p.name, func(b *testing.B) {
				st := cachedMeasure(sc, p)
				for range b.N {
				}
				b.ReportMetric(0, "ns/op")
				b.ReportMetric(st.delayMean, "delay-mean-s")
				b.ReportMetric(st.delayP50, "delay-p50-s")
				b.ReportMetric(st.delayP95, "delay-p95-s")
				b.ReportMetric(st.delayMax, "delay-max-s")
				b.ReportMetric(st.pollsMean, "polls-mean")
				b.ReportMetric(st.pollsP95, "polls-p95")
				b.ReportMetric(float64(st.herdPeak), "herd-peak/100ms")
			})
		}
	}
}

func TestPollScheduleReport(t *testing.T) {
	if os.Getenv("POLLSCHEDULE_REPORT") == "" {
		t.Skip("set POLLSCHEDULE_REPORT=1 to print the schedule comparison")
	}
	for _, sc := range scenarios {
		var b strings.Builder
		fmt.Fprintf(&b, "\nscenario %s: completion %v..%v (log-uniform), %d tasks, poll latency %v, herd of %d\n",
			sc.name, sc.min, sc.max, samples, pollLatency, herdClients)
		fmt.Fprintf(&b, "%-39s %10s %10s %10s %10s %10s %10s %10s %7s\n", "policy", "delay-mean", "delay-p50", "delay-p95", "delay-max", "polls-mean", "polls-p95", "herd-peak", "pareto")
		all := policies()
		results := make([]stats, len(all))
		for i, p := range all {
			results[i] = cachedMeasure(sc, p)
		}
		for i, p := range all {
			st := results[i]
			mark := ""
			if pareto(results, i) {
				mark = "*"
			}
			fmt.Fprintf(&b, "%-39s %9.2fs %9.2fs %9.2fs %9.2fs %10.1f %10.0f %10d %7s\n", p.name,
				st.delayMean, st.delayP50, st.delayP95, st.delayMax, st.pollsMean, st.pollsP95, st.herdPeak, mark)
		}
		t.Log(b.String())
	}
}

// pareto reports whether no other result has both a lower mean delay and
// fewer mean polls than results[i].
func pareto(results []stats, i int) bool {
	for j, o := range results {
		if j != i && o.delayMean <= results[i].delayMean && o.pollsMean <= results[i].pollsMean &&
			(o.delayMean < results[i].delayMean || o.pollsMean < results[i].pollsMean) {
			return false
		}
	}
	return true
}

func TestSimulateCountsPollsAndDetectionDelay(t *testing.T) {
	// Polls are sent at 0, 2.15, 4.30 and 6.45 seconds; the last one sees the
	// task that finished at 5s and answers 150ms later.
	delay, polls := simulate(Fixed(2*time.Second), 5*time.Second, pollLatency, nil, nil)
	if polls != 4 || delay != 1600*time.Millisecond {
		t.Fatalf("simulate = %v after %d polls, want 1.6s after 4", delay, polls)
	}
	// A task that is already complete is seen by the initial lookup.
	if delay, polls := simulate(Exponential(time.Second, 2, 0), 0, pollLatency, nil, nil); polls != 1 || delay != pollLatency {
		t.Fatalf("immediate completion = %v after %d polls", delay, polls)
	}
}

func TestMeasureIsDeterministic(t *testing.T) {
	sc := scenario{"tiny", time.Second, 20 * time.Second, 5}
	p := policy{"full", Exponential(time.Second, 2, 10*time.Second).WithJitter(FullJitter)}
	if a, b := measure(sc, p), measure(sc, p); a != b {
		t.Fatalf("measure differs between runs: %+v vs %+v", a, b)
	}
}
