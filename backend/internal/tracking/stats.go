package tracking

// allStatuses is every Status value the funnel/stats view reports a count
// for (story 1), in pipeline order.
var allStatuses = []Status{
	StatusSaved,
	StatusTailoring,
	StatusSent,
	StatusInterviewing,
	StatusRejected,
	StatusOffer,
}

// stageRank places each Status on the fixed pipeline order the PRD names —
// Saved -> Tailoring -> Sent -> Interviewing -> Rejected/Offer — for
// "reached-or-passed" conversion math (story 2): an Application's current
// Status alone, with no history, can't say whether a Rejected outcome came
// via Interviewing or straight from Sent, so both terminal outcomes are
// placed at the same final rank. This is a pragmatic simplification (noted
// in the PRD's Further Notes) to revisit once StatusHistory has
// accumulated enough real data to compute conversion from observed
// transitions instead.
var stageRank = map[Status]int{
	StatusSaved:        0,
	StatusTailoring:    1,
	StatusSent:         2,
	StatusInterviewing: 3,
	StatusRejected:     4,
	StatusOffer:        4,
}

// conversionPairs are the adjacent-stage conversions the stats view
// reports (story 2), matching the PRD's own examples plus the win-rate
// story 4 explicitly asks for (Interviewing -> Offer specifically, not a
// merged terminal bucket).
var conversionPairs = [][2]Status{
	{StatusSaved, StatusTailoring},
	{StatusTailoring, StatusSent},
	{StatusSent, StatusInterviewing},
	{StatusInterviewing, StatusOffer},
}

// StatusCount is how many Applications currently sit in Status.
type StatusCount struct {
	Status Status `json:"status"`
	Count  int    `json:"count"`
}

// ConversionRate is the fraction of Applications that reached From which
// went on to reach To, using "reached-or-passed" semantics (an
// Application's current Status implies every earlier pipeline stage was
// passed through). Conversions with a zero denominator (story 11: no
// Applications have reached From yet) are omitted from Stats.Conversions
// entirely rather than reported as 0 or NaN.
type ConversionRate struct {
	From Status  `json:"from"`
	To   Status  `json:"to"`
	Rate float64 `json:"rate"`
}

// StageTime is the average time Applications spent in Status before their
// next recorded Status change, computed only from StatusHistory entries
// that actually exist (story 8) — a Status with no such sample is omitted
// from Stats.TimeInStage, never reported as 0 or "N/A".
type StageTime struct {
	Status      Status  `json:"status"`
	AverageDays float64 `json:"averageDays"`
	SampleSize  int     `json:"sampleSize"`
}

// Stats is the Application funnel/stats view's data (issue #36): a count
// per Status, stage-to-stage conversion rates, and average time-in-stage
// where history supports it.
type Stats struct {
	Total       int              `json:"total"`
	Counts      []StatusCount    `json:"counts"`
	Conversions []ConversionRate `json:"conversions"`
	TimeInStage []StageTime      `json:"timeInStage"`
}

// ComputeStats aggregates funnel/stats data from Applications alone — no
// storage/HTTP dependency (Testing Decisions), and no new persistence:
// everything here is derived from data List already reads (story 6, story
// 10).
func ComputeStats(applications []Application) Stats {
	counts := make(map[Status]int, len(allStatuses))
	for _, s := range allStatuses {
		counts[s] = 0
	}
	reached := make(map[Status]int, len(allStatuses))
	for _, s := range allStatuses {
		reached[s] = 0
	}

	stageDurationDaysSum := map[Status]float64{}
	stageSampleSize := map[Status]int{}

	for _, app := range applications {
		counts[app.Status]++

		currentRank, known := stageRank[app.Status]
		if known {
			for _, s := range allStatuses {
				if stageRank[s] <= currentRank {
					reached[s]++
				}
			}
		}

		for i := 1; i < len(app.StatusHistory); i++ {
			prev := app.StatusHistory[i-1]
			next := app.StatusHistory[i]
			days := next.ChangedAt.Sub(prev.ChangedAt).Hours() / 24
			stageDurationDaysSum[prev.Status] += days
			stageSampleSize[prev.Status]++
		}
	}

	statusCounts := make([]StatusCount, 0, len(allStatuses))
	for _, s := range allStatuses {
		statusCounts = append(statusCounts, StatusCount{Status: s, Count: counts[s]})
	}

	var conversions []ConversionRate
	for _, pair := range conversionPairs {
		from, to := pair[0], pair[1]
		if reached[from] == 0 {
			continue
		}
		conversions = append(conversions, ConversionRate{
			From: from,
			To:   to,
			Rate: float64(reached[to]) / float64(reached[from]),
		})
	}

	var timeInStage []StageTime
	for _, s := range allStatuses {
		if stageSampleSize[s] == 0 {
			continue
		}
		timeInStage = append(timeInStage, StageTime{
			Status:      s,
			AverageDays: stageDurationDaysSum[s] / float64(stageSampleSize[s]),
			SampleSize:  stageSampleSize[s],
		})
	}
	return Stats{
		Total:       len(applications),
		Counts:      statusCounts,
		Conversions: conversions,
		TimeInStage: timeInStage,
	}
}
