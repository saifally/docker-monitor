package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
)

type ExcelDriver struct {
	config     LogConfig
	filePath   string
	file       *excelize.File
	sheetName  string
	currentRow int
}

func NewExcelDriver(config LogConfig) *ExcelDriver {
	filePath := config.OutputPath
	if filePath == "" {
		filePath = "container_metrics.xlsx"
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	sheetName := "Container Metrics"
	if name, exists := config.Options["sheet_name"]; exists {
		sheetName = name
	}

	return &ExcelDriver{
		config:     config,
		filePath:   filePath,
		sheetName:  sheetName,
		currentRow: 1,
	}
}

func (e *ExcelDriver) Initialize() error {
	var err error

	// Try to open existing file
	if _, statErr := os.Stat(e.filePath); statErr == nil {
		e.file, err = excelize.OpenFile(e.filePath)
		if err != nil {
			return fmt.Errorf("failed to open existing Excel file: %w", err)
		}

		// Find the last row with data
		rows, err := e.file.GetRows(e.sheetName)
		if err == nil && len(rows) > 0 {
			e.currentRow = len(rows) + 1
		}
	} else {
		// Create new file
		e.file = excelize.NewFile()

		// Create sheet if it doesn't exist
		if e.sheetName != "Sheet1" {
			index, err := e.file.NewSheet(e.sheetName)
			if err != nil {
				return fmt.Errorf("failed to create sheet: %w", err)
			}
			e.file.SetActiveSheet(index)
			e.file.DeleteSheet("Sheet1")
		} else {
			e.sheetName = "Sheet1"
		}

		// Write headers
		headers := []string{
			"Timestamp", "Name", "Status", "Image", "CPU %",
			"Memory Usage", "Memory Limit", "Project", "Compose Service",
		}

		for i, header := range headers {
			cell := fmt.Sprintf("%c1", 'A'+i)
			e.file.SetCellValue(e.sheetName, cell, header)
		}

		// Style headers
		style, err := e.file.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"E0E0E0"}, Pattern: 1},
		})
		if err == nil {
			e.file.SetCellStyle(e.sheetName, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), style)
		}

		e.currentRow = 2
	}

	return nil
}

func (e *ExcelDriver) LogMetrics(metrics []ContainerMetrics) error {
	for _, metric := range metrics {
		row := e.currentRow

		// Set cell values
		cells := []interface{}{
			metric.Timestamp.Format(time.RFC3339),
			metric.Name,
			metric.Status,
			metric.Image,
			metric.CPUPercent,
			metric.MemoryUsage,
			metric.MemoryLimit,
			metric.Project,
			e.getComposeService(metric.Labels),
		}

		for i, value := range cells {
			cell := fmt.Sprintf("%c%d", 'A'+i, row)
			e.file.SetCellValue(e.sheetName, cell, value)
		}

		// Format CPU percentage as percentage
		cpuCell := fmt.Sprintf("E%d", row)
		e.file.SetCellValue(e.sheetName, cpuCell, metric.CPUPercent/100)
		style, err := e.file.NewStyle(&excelize.Style{
			NumFmt: 10, // Percentage format
		})
		if err == nil {
			e.file.SetCellStyle(e.sheetName, cpuCell, cpuCell, style)
		}

		e.currentRow++
	}

	// Auto-adjust column widths
	e.autoAdjustColumns()

	// Save file
	if err := e.file.SaveAs(e.filePath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

func (e *ExcelDriver) LogInfo(message string, fields map[string]interface{}) error {
	// Create info log sheet if it doesn't exist
	infoSheet := "Info Logs"
	sheets := e.file.GetSheetList()
	sheetExists := false
	for _, sheet := range sheets {
		if sheet == infoSheet {
			sheetExists = true
			break
		}
	}

	if !sheetExists {
		_, err := e.file.NewSheet(infoSheet)
		if err != nil {
			return err
		}

		// Write headers
		e.file.SetCellValue(infoSheet, "A1", "Timestamp")
		e.file.SetCellValue(infoSheet, "B1", "Message")
		e.file.SetCellValue(infoSheet, "C1", "Fields")

		// Style headers
		style, err := e.file.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"E0E0E0"}, Pattern: 1},
		})
		if err == nil {
			e.file.SetCellStyle(infoSheet, "A1", "C1", style)
		}
	}

	// Find next row
	rows, _ := e.file.GetRows(infoSheet)
	nextRow := len(rows) + 1

	// Write log entry
	e.file.SetCellValue(infoSheet, fmt.Sprintf("A%d", nextRow), time.Now().Format(time.RFC3339))
	e.file.SetCellValue(infoSheet, fmt.Sprintf("B%d", nextRow), message)

	// Convert fields to string
	fieldsStr := ""
	for k, v := range fields {
		fieldsStr += fmt.Sprintf("%s=%v; ", k, v)
	}
	e.file.SetCellValue(infoSheet, fmt.Sprintf("C%d", nextRow), fieldsStr)

	return e.file.SaveAs(e.filePath)
}

func (e *ExcelDriver) LogError(message string, err error) error {
	fields := map[string]interface{}{
		"error": err.Error(),
	}
	return e.LogInfo(fmt.Sprintf("ERROR: %s", message), fields)
}

func (e *ExcelDriver) Close() error {
	if e.file != nil {
		return e.file.SaveAs(e.filePath)
	}
	return nil
}

func (e *ExcelDriver) getComposeService(labels map[string]string) string {
	if service, exists := labels["com.docker.compose.service"]; exists {
		return service
	}
	return ""
}

func (e *ExcelDriver) autoAdjustColumns() {
	columns := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	widths := []float64{20, 25, 15, 30, 10, 15, 15, 15, 20}

	for i, col := range columns {
		if i < len(widths) {
			e.file.SetColWidth(e.sheetName, col, col, widths[i])
		}
	}
}

// CreateChart creates a chart in the Excel file for CPU usage trends
func (e *ExcelDriver) CreateChart() error {
	// Get data range
	rows, err := e.file.GetRows(e.sheetName)
	if err != nil || len(rows) < 3 {
		return fmt.Errorf("insufficient data for chart creation")
	}

	// Create chart - simplified version without Title field that's not available in this excelize version
	chart := excelize.Chart{
		Type: excelize.Line,
		Series: []excelize.ChartSeries{
			{
				Name:       "CPU Usage %",
				Categories: fmt.Sprintf("%s!$A$2:$A$%d", e.sheetName, len(rows)),
				Values:     fmt.Sprintf("%s!$E$2:$E$%d", e.sheetName, len(rows)),
			},
		},
		PlotArea: excelize.ChartPlotArea{
			ShowCatName:     false,
			ShowLeaderLines: false,
			ShowPercent:     true,
			ShowSerName:     true,
			ShowVal:         false,
		},
	}

	// Insert chart
	return e.file.AddChart(e.sheetName, "K2", &chart)
}