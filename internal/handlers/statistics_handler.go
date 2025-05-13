package handlers

import (
	"net/http"
	"sort"
	"time"
)

// ParentStatsForTemplate holds processed monthly statistics for a single parent,
// structured for easy use in the template.
type ParentStatsForTemplate struct {
	ParentName    string
	MonthlyCounts map[string]int // Key: "YYYY-MM", Value: Count
}

// StatisticsPageData contains data for the statistics page template.
type StatisticsPageData struct {
	ErrorMessage string
	ParentsStats []ParentStatsForTemplate
	MonthHeaders []string // Sorted list of "YYYY-MM" for table columns, e.g., ["2023-06", "2023-07"]
}

// StatisticsHandler manages statistics page functionality.
type StatisticsHandler struct {
	*BaseHandler
	// Tracker is accessed via BaseHandler: h.Tracker
}

// NewStatisticsHandler creates a new statistics page handler.
func NewStatisticsHandler(baseHandler *BaseHandler) *StatisticsHandler {
	return &StatisticsHandler{
		BaseHandler: baseHandler,
	}
}

// RegisterRoutes registers statistics page related routes.
func (h *StatisticsHandler) RegisterRoutes() {
	http.HandleFunc("/statistics", h.handleStatisticsPage)
}

// handleStatisticsPage shows the statistics page.
func (h *StatisticsHandler) handleStatisticsPage(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleStatisticsPage").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling statistics page request")

	data := StatisticsPageData{}

	rawStats, err := h.Tracker.GetParentMonthlyStatsForLastNMonths(12)
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get parent monthly stats from tracker")
		data.ErrorMessage = "Could not retrieve statistics data. Please try again later."
		h.RenderTemplate(w, "statistics.html", data)
		return
	}

	parentNamesSet := make(map[string]struct{})
	monthHeadersSet := make(map[string]struct{})

	now := time.Now()
	for i := 0; i < 12; i++ {
		// Iterate from 11 months ago to current month to get the last 12 distinct months
		month := now.AddDate(0, -(11 - i), 0)
		monthHeadersSet[month.Format("2006-01")] = struct{}{}
	}

	for _, stat := range rawStats {
		parentNamesSet[stat.ParentName] = struct{}{}
	}

	var sortedParentNames []string
	for name := range parentNamesSet {
		sortedParentNames = append(sortedParentNames, name)
	}
	sort.Strings(sortedParentNames)

	for month := range monthHeadersSet {
		data.MonthHeaders = append(data.MonthHeaders, month)
	}
	sort.Strings(data.MonthHeaders) // Sorts "YYYY-MM" chronologically

	statsMap := make(map[string]map[string]int) // ParentName -> MonthYear -> Count
	for _, stat := range rawStats {
		if _, ok := statsMap[stat.ParentName]; !ok {
			statsMap[stat.ParentName] = make(map[string]int)
		}
		statsMap[stat.ParentName][stat.MonthYear] = stat.Count
	}

	for _, parentName := range sortedParentNames {
		parentStat := ParentStatsForTemplate{
			ParentName:    parentName,
			MonthlyCounts: make(map[string]int),
		}
		for _, monthHeader := range data.MonthHeaders {
			if count, ok := statsMap[parentName][monthHeader]; ok {
				parentStat.MonthlyCounts[monthHeader] = count
			} else {
				// Ensure every parent has an entry for every month in the header, defaulting to 0
				parentStat.MonthlyCounts[monthHeader] = 0
			}
		}
		data.ParentsStats = append(data.ParentsStats, parentStat)
	}

	handlerLogger.Debug().Int("parent_count", len(data.ParentsStats)).Int("month_header_count", len(data.MonthHeaders)).Msg("Processed statistics data for template")
	h.RenderTemplate(w, "statistics.html", data)
}
