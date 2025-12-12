package dataset

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"voice-insights-go/internal/types"
)

// Load attempts to auto-detect audio URL column by header heuristics
func Load(path string) ([]types.CallRecord, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) <= 1 {
		return nil, fmt.Errorf("no data rows")
	}
	header := rows[0]
	// find audio column
	audioIdx := -1
	callIDIdx := -1
	callTypeIdx := -1
	cityIdx := -1
	vintageIdx := -1
	repeatIdx := -1
	for i, h := range header {
		l := strings.ToLower(strings.TrimSpace(h))
		switch {
		case strings.Contains(l, "audio") || strings.Contains(l, "record") || strings.Contains(l, "call") && strings.Contains(l, "link") || strings.Contains(l, "url"):
			if audioIdx == -1 {
				audioIdx = i
			}
		case strings.Contains(l, "call id") || strings.Contains(l, "callid") || strings.Contains(l, "id"):
			if callIDIdx == -1 {
				callIDIdx = i
			}
		case strings.Contains(l, "type"):
			if callTypeIdx == -1 {
				callTypeIdx = i
			}
		case strings.Contains(l, "city"):
			cityIdx = i
		case strings.Contains(l, "vintage") || strings.Contains(l, "month"):
			vintageIdx = i
		case strings.Contains(l, "repeat") || strings.Contains(l, "escalation"):
			repeatIdx = i
		}
	}
	// fallback heuristics
	if audioIdx == -1 {
		// try common positions: 4 or 5
		if len(header) > 4 {
			audioIdx = 4
		} else {
			audioIdx = -1
		}
	}
	var out []types.CallRecord
	for i, r := range rows {
		if i == 0 {
			continue
		}
		record := types.CallRecord{}
		if callIDIdx >= 0 && callIDIdx < len(r) {
			record.CallID = r[callIDIdx]
		}
		if callTypeIdx >= 0 && callTypeIdx < len(r) {
			record.CallType = r[callTypeIdx]
		}
		if audioIdx >= 0 && audioIdx < len(r) {
			record.AudioURL = r[audioIdx]
		}
		if cityIdx >= 0 && cityIdx < len(r) {
			record.City = r[cityIdx]
		}
		if vintageIdx >= 0 && vintageIdx < len(r) {
			record.VintageMonth, _ = strconv.Atoi(strings.TrimSpace(r[vintageIdx]))
		}
		if repeatIdx >= 0 && repeatIdx < len(r) {
			record.RepeatEsc, _ = strconv.Atoi(strings.TrimSpace(r[repeatIdx]))
		}
		// if audio URL doesn't look like URL, skip
		if !(strings.HasPrefix(strings.ToLower(record.AudioURL), "http://") || strings.HasPrefix(strings.ToLower(record.AudioURL), "https://")) {
			// skip invalid audio rows quietly
			continue
		}
		out = append(out, record)
	}
	return out, nil
}
