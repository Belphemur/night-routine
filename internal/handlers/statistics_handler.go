package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
)

// ParentStatsForTemplate holds processed monthly statistics for a single parent,
// structured for easy use in the template.
type ParentStatsForTemplate struct {
	ParentName    string
	MonthlyCounts map[string]int // Key: "YYYY-MM", Value: Count
}

// StatisticsPageData contains data for the statistics page template.
type StatisticsPageData struct {
	BasePageData
	ErrorMessage string
	ParentsStats []ParentStatsForTemplate
	MonthHeaders []string // Sorted list of "YYYY-MM" for table columns, e.g., ["2023-06", "2023-07"]
}

// StatisticsHandler manages statistics page functionality.
type StatisticsHandler struct {
	*BaseHandler
	configStore *database.ConfigStore
}

// NewStatisticsHandler creates a new statistics page handler.
func NewStatisticsHandler(baseHandler *BaseHandler, configStore *database.ConfigStore) *StatisticsHandler {
	return &StatisticsHandler{
		BaseHandler: baseHandler,
		configStore: configStore,
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

	data := StatisticsPageData{
		BasePageData: h.NewBasePageData(r, true), // Assuming authenticated
	}
	nowForStats := time.Now() // Use a consistent "now" for this request processing

	// Get the stats order from configuration (we only need statsOrder, ignore other schedule values)
	_, _, _, statsOrder, err := h.configStore.GetSchedule()
	if err != nil {
		handlerLogger.Warn().Err(err).Msg("Failed to get schedule configuration, defaulting to descending order")
		statsOrder = constants.StatsOrderDesc
	}

	rawStats, err := h.Tracker.GetParentMonthlyStatsForLastNMonths(nowForStats, 12)
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get parent monthly stats from tracker")
		data.ErrorMessage = "Could not retrieve statistics data. Please try again later."
		h.RenderTemplate(w, "statistics.html", data)
		return
	}

	if len(rawStats) == 0 {
		// No data from the database, so show "No statistics data available"
		// data.ParentsStats is already nil, data.MonthHeaders is empty.
		handlerLogger.Info().Msg("No raw statistics data found. Rendering page with 'No data available'.")
		h.RenderTemplate(w, "statistics.html", data)
		return
	}

	// 1. Create a lookup map from rawStats: ParentName -> MonthYear -> Count
	statsLookupMap := make(map[string]map[string]int)
	parentNamesSet := make(map[string]struct{})
	for _, stat := range rawStats {
		parentNamesSet[stat.ParentName] = struct{}{}
		if _, ok := statsLookupMap[stat.ParentName]; !ok {
			statsLookupMap[stat.ParentName] = make(map[string]int)
		}
		statsLookupMap[stat.ParentName][stat.MonthYear] = stat.Count
	}

	var sortedParentNames []string
	for name := range parentNamesSet {
		sortedParentNames = append(sortedParentNames, name)
	}
	sort.Strings(sortedParentNames)

	// 2. Generate all 12 potential month headers for the last 12 months
	allPossibleMonthHeaders := []string{}
	// Use the same nowForStats as used for fetching data, for consistency in month generation
	for i := 0; i < 12; i++ {
		// This loop generates months in chronological order (ascending):
		// i=0: -(11-0) = -11 (oldest month in range)
		// ...
		// i=11: -(11-11) = 0 (current month)
		month := nowForStats.AddDate(0, -(11 - i), 0)
		allPossibleMonthHeaders = append(allPossibleMonthHeaders, month.Format("2006-01"))
	}

	// 3. Filter month headers: only keep months where at least one parent has a non-zero count.
	finalMonthHeaders := []string{}
	for _, monthStr := range allPossibleMonthHeaders {
		hasDataForThisMonth := false
		for _, parentName := range sortedParentNames {
			if parentData, ok := statsLookupMap[parentName]; ok {
				if count, ok2 := parentData[monthStr]; ok2 && count > 0 {
					hasDataForThisMonth = true
					break
				}
			}
		}
		if hasDataForThisMonth {
			finalMonthHeaders = append(finalMonthHeaders, monthStr)
		}
	}

	// 4. If no month headers remain after filtering (all months had zero counts for all parents),
	//    then treat as "No data available".
	if len(finalMonthHeaders) == 0 {
		handlerLogger.Info().Msg("All months have zero counts for all parents. Rendering page with 'No data available'.")
		// data.ParentsStats is still nil. data.MonthHeaders should be empty for the template's "No data" block.
		data.MonthHeaders = nil // Explicitly set to nil for clarity, though empty slice works too.
		h.RenderTemplate(w, "statistics.html", data)
		return
	}

	// 5. Apply the configured sort order
	if statsOrder == constants.StatsOrderDesc {
		// Reverse the slice for descending order (newest month first)
		for i, j := 0, len(finalMonthHeaders)-1; i < j; i, j = i+1, j-1 {
			finalMonthHeaders[i], finalMonthHeaders[j] = finalMonthHeaders[j], finalMonthHeaders[i]
		}
	}
	// For ascending order (StatsOrderAsc), keep the order as-is (oldest first)

	data.MonthHeaders = finalMonthHeaders

	// 6. Build data.ParentsStats using the filtered finalMonthHeaders.
	for _, parentName := range sortedParentNames {
		parentStat := ParentStatsForTemplate{
			ParentName:    parentName,
			MonthlyCounts: make(map[string]int),
		}
		// For each of the *filtered* display month headers, fill in the count for the current parent
		for _, monthHeader := range data.MonthHeaders {
			count := 0 // Default to 0
			if parentMonthlyData, parentExists := statsLookupMap[parentName]; parentExists {
				if monthCount, monthExists := parentMonthlyData[monthHeader]; monthExists {
					count = monthCount
				}
			}
			parentStat.MonthlyCounts[monthHeader] = count
		}
		data.ParentsStats = append(data.ParentsStats, parentStat)
	}

	handlerLogger.Debug().
		Int("parent_count", len(data.ParentsStats)).
		Int("month_header_count", len(data.MonthHeaders)).
		Str("stats_order", statsOrder.String()).
		Msg("Processed statistics data for template")
	h.RenderTemplate(w, "statistics.html", data)
}
