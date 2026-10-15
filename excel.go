package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"
)

// GenerateExcelReport creates a professional styled Excel spreadsheet with color-coded results
func GenerateExcelReport(results []TestResult, exportPath string) (string, error) {
	if len(results) == 0 {
		return "", fmt.Errorf("no test results available to export")
	}

	// Sort results: SUCCESS first, then highest download, then lowest ping, then lowest connect time
	sortedResults := make([]TestResult, len(results))
	copy(sortedResults, results)
	sort.Slice(sortedResults, func(i, j int) bool {
		r1 := sortedResults[i]
		r2 := sortedResults[j]

		if r1.Status == "SUCCESS" && r2.Status != "SUCCESS" {
			return true
		}
		if r1.Status != "SUCCESS" && r2.Status == "SUCCESS" {
			return false
		}
		if r1.Download != r2.Download {
			return r1.Download > r2.Download
		}
		if r1.Ping > 0 && r2.Ping > 0 && r1.Ping != r2.Ping {
			return r1.Ping < r2.Ping
		}
		return r1.Duration < r2.Duration
	})

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Windscribe Test Results"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", err
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	// Define styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#1A2332"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "top", Color: "#3E4C5E", Style: 1},
			{Type: "bottom", Color: "#3E4C5E", Style: 2},
			{Type: "left", Color: "#3E4C5E", Style: 1},
			{Type: "right", Color: "#3E4C5E", Style: 1},
		},
	})

	greenRowStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100", Bold: true, Size: 10},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#C6EFCE", Style: 1},
		},
	})

	yellowRowStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#7F6000", Size: 10},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFF2CC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#FFE29C", Style: 1},
		},
	})

	redRowStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C0006", Size: 10},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FCE4D6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#FFC7CE", Style: 1},
		},
	})

	headers := []string{
		"#", "Timestamp", "ISP / Operator", "Server Nickname", "City", "Country", "Continent",
		"Protocol", "Port", "Connect Time (s)", "Status", "Ping (ms)", "Jitter (ms)",
		"Download (Mbps)", "Upload (Mbps)", "Protocol Tweaks", "Amnezia Config",
		"Server Routing", "Custom SNI", "Large TLS", "Internal DNS", "Quality Grade",
	}

	// Write Headers
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(sheetName, 1, 28)

	// Write Data Rows
	for rowIdx, r := range sortedResults {
		rowNum := rowIdx + 2

		// Determine Quality Grade & Style
		var grade string
		var rowStyle int

		if r.Status == "SUCCESS" {
			if r.Download >= 10.0 && r.Ping <= 200 {
				grade = "🟢 Excellent"
				rowStyle = greenRowStyle
			} else if r.Download >= 3.0 || r.Ping <= 300 {
				grade = "🟡 Good / Fair"
				rowStyle = yellowRowStyle
			} else {
				grade = "🟠 Poor"
				rowStyle = yellowRowStyle
			}
		} else {
			grade = "🔴 Failed / Timeout"
			rowStyle = redRowStyle
		}

		ispStr := r.ISP
		if ispStr == "" {
			ispStr = "Default / Direct"
		}
		tweaksStr := "Disabled"
		if r.ProtocolTweaks {
			tweaksStr = "Enabled"
		}
		tlsStr := "No"
		if r.LargeTLS {
			tlsStr = "Yes"
		}
		sniStr := r.CustomSNI
		if r.Protocol != "stealth" && r.Protocol != "wstunnel" {
			sniStr = "-"
		} else if sniStr == "" {
			sniStr = "-"
		}
		amneziaStr := r.AntiCensorshipMode
		if !r.ProtocolTweaks {
			amneziaStr = "Disabled"
		} else if amneziaStr == "" || amneziaStr == "true" {
			amneziaStr = "Default"
		} else if amneziaStr == "false" {
			amneziaStr = "None"
		}

		vals := []interface{}{
			rowIdx + 1,
			r.Timestamp,
			ispStr,
			r.Location,
			r.City,
			r.Country,
			r.Continent,
			r.Protocol,
			r.Port,
			fmt.Sprintf("%.2f", r.Duration),
			r.Status,
			fmt.Sprintf("%.0f", r.Ping),
			fmt.Sprintf("%.0f", r.Jitter),
			fmt.Sprintf("%.2f", r.Download),
			fmt.Sprintf("%.2f", r.Upload),
			tweaksStr,
			amneziaStr,
			r.ServerRouting,
			sniStr,
			tlsStr,
			r.InternalDNS,
			grade,
		}

		for colIdx, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
			_ = f.SetCellValue(sheetName, cell, v)
			_ = f.SetCellStyle(sheetName, cell, cell, rowStyle)
		}
		_ = f.SetRowHeight(sheetName, rowNum, 20)
	}

	// Auto-fit column widths
	colWidths := map[string]float64{
		"A": 6,  // #
		"B": 20, // Timestamp
		"C": 22, // ISP / Operator
		"D": 18, // Server Nickname
		"E": 16, // City
		"F": 18, // Country
		"G": 14, // Continent
		"H": 12, // Protocol
		"I": 10, // Port
		"J": 16, // Connect Time
		"K": 12, // Status
		"L": 12, // Ping
		"M": 12, // Jitter
		"N": 16, // Download
		"O": 16, // Upload
		"P": 16, // Protocol Tweaks
		"Q": 25, // Amnezia Config
		"R": 16, // Server Routing
		"S": 18, // Custom SNI
		"T": 12, // Large TLS
		"U": 14, // Internal DNS
		"V": 18, // Quality Grade
	}
	for col, width := range colWidths {
		_ = f.SetColWidth(sheetName, col, col, width)
	}

	// Determine output path
	if exportPath == "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		exportPath = filepath.Join(".", fmt.Sprintf("Windscribe_Results_%s.xlsx", timestamp))
	}

	// Ensure directory exists
	dir := filepath.Dir(exportPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := f.SaveAs(exportPath); err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(exportPath)
	if err == nil {
		return absPath, nil
	}
	return exportPath, nil
}
