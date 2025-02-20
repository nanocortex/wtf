package uptimekuma

import (
	"strconv"
	"strings"
)

// PrometheusMetricItem represents a single metric measurement with labels
type PrometheusMetricItem struct {
	Labels map[string]string
	Value  float64
}

// PrometheusMetric represents a collection of related metric items
type PrometheusMetric struct {
	Name  string
	Type  string
	Items []PrometheusMetricItem
}

// Parse converts raw Prometheus metrics data into structured format
func Parse(metricsData string) ([]*PrometheusMetric, error) {
	var result []*PrometheusMetric
	var currentMetricInfo *PrometheusMetric

	lines := strings.Split(metricsData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle type header
		if strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "# TYPE") {
				pieces := strings.SplitN(line, " ", 4)
				if len(pieces) >= 4 {
					currentMetricInfo = &PrometheusMetric{
						Name:  pieces[2],
						Type:  pieces[3],
						Items: make([]PrometheusMetricItem, 0),
					}
					result = append(result, currentMetricInfo)
				}
			}
			continue
		}

		// Parse metric line
		labelIndex := strings.Index(line, "{")
		closingLabelIndex := strings.Index(line, "}")

		var metricName string
		var metricValueString string
		item := PrometheusMetricItem{
			Labels: make(map[string]string),
		}

		if labelIndex != -1 { // Metric has labels
			metricName = strings.TrimSpace(line[:labelIndex])
			labelsString := line[labelIndex+1 : closingLabelIndex]
			labels := strings.Split(labelsString, ",")

			for _, label := range labels {
				keyValue := strings.Split(label, "=")
				if len(keyValue) == 2 {
					key := strings.TrimSpace(keyValue[0])
					// Remove quotes around the label value if present
					value := strings.Trim(strings.TrimSpace(keyValue[1]), "\"")
					item.Labels[key] = value
				}
			}

			// Split remaining part and get the second element (value)
			parts := strings.Fields(line[closingLabelIndex+1:])
			if len(parts) >= 2 {
				metricValueString = parts[1] // Assumes no timestamp
			} else {
				metricValueString = parts[0]
			}
		} else { // No labels
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				metricName = parts[0]
				metricValueString = parts[1]
			} else {
				metricName = line
				metricValueString = "0"
			}
		}

		// Find or create metric info
		if currentMetricInfo == nil || currentMetricInfo.Name != metricName {
			found := false
			for _, m := range result {
				if m.Name == metricName {
					currentMetricInfo = m
					currentMetricInfo.Items = currentMetricInfo.Items[:0] // Clear items
					found = true
					break
				}
			}
			if !found {
				currentMetricInfo = &PrometheusMetric{
					Name:  metricName,
					Type:  "unknown",
					Items: make([]PrometheusMetricItem, 0),
				}
				result = append(result, currentMetricInfo)
			}
		}

		// Parse value
		value, err := strconv.ParseFloat(metricValueString, 64)
		if err == nil {
			item.Value = value
			currentMetricInfo.Items = append(currentMetricInfo.Items, item)
		} else {
			item.Value = 0
			currentMetricInfo.Items = append(currentMetricInfo.Items, item)
		}
	}

	return result, nil
}
