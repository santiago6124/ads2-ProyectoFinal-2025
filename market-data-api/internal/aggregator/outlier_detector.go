package aggregator

import (
	"math"
	"sort"
)

// OutlierDetector handles outlier detection in price data
type OutlierDetector struct {
	method    string  // z_score, iqr, isolation_forest
	threshold float64 // threshold for outlier detection
}

// NewOutlierDetector creates a new outlier detector
func NewOutlierDetector(method string, threshold float64) *OutlierDetector {
	if method == "" {
		method = "z_score"
	}
	if threshold <= 0 {
		threshold = 2.0
	}

	return &OutlierDetector{
		method:    method,
		threshold: threshold,
	}
}

// DetectOutliers detects outliers in a slice of price values
func (od *OutlierDetector) DetectOutliers(values []float64) []int {
	if len(values) < 3 {
		return nil // Not enough data for outlier detection
	}

	switch od.method {
	case "z_score":
		return od.detectOutliersZScore(values)
	case "iqr":
		return od.detectOutliersIQR(values)
	case "isolation_forest":
		return od.detectOutliersIsolationForest(values)
	case "modified_z_score":
		return od.detectOutliersModifiedZScore(values)
	default:
		return od.detectOutliersZScore(values)
	}
}

// detectOutliersZScore detects outliers using Z-score method
func (od *OutlierDetector) detectOutliersZScore(values []float64) []int {
	if len(values) < 3 {
		return nil
	}

	// Calculate mean and standard deviation
	mean := od.calculateMean(values)
	stdDev := od.calculateStdDev(values, mean)

	if stdDev == 0 {
		return nil // No variation in data
	}

	var outliers []int
	for i, value := range values {
		zScore := math.Abs((value - mean) / stdDev)
		if zScore > od.threshold {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

// detectOutliersModifiedZScore detects outliers using Modified Z-score (more robust)
func (od *OutlierDetector) detectOutliersModifiedZScore(values []float64) []int {
	if len(values) < 3 {
		return nil
	}

	// Calculate median
	median := od.calculateMedian(values)

	// Calculate Median Absolute Deviation (MAD)
	deviations := make([]float64, len(values))
	for i, value := range values {
		deviations[i] = math.Abs(value - median)
	}

	mad := od.calculateMedian(deviations)

	if mad == 0 {
		return nil // No variation in data
	}

	var outliers []int
	for i, value := range values {
		modifiedZScore := 0.6745 * (value - median) / mad
		if math.Abs(modifiedZScore) > od.threshold {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

// detectOutliersIQR detects outliers using Interquartile Range method
func (od *OutlierDetector) detectOutliersIQR(values []float64) []int {
	if len(values) < 4 {
		return nil
	}

	// Sort values
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	// Calculate quartiles
	q1 := od.calculatePercentile(sortedValues, 0.25)
	q3 := od.calculatePercentile(sortedValues, 0.75)

	iqr := q3 - q1
	if iqr == 0 {
		return nil // No variation in quartiles
	}

	// Calculate outlier bounds
	lowerBound := q1 - od.threshold*iqr
	upperBound := q3 + od.threshold*iqr

	var outliers []int
	for i, value := range values {
		if value < lowerBound || value > upperBound {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

// detectOutliersIsolationForest simplified isolation forest implementation
func (od *OutlierDetector) detectOutliersIsolationForest(values []float64) []int {
	if len(values) < 3 {
		return nil
	}

	// Simple implementation: use statistical measures as approximation
	// In a full implementation, this would use actual isolation forest algorithm

	mean := od.calculateMean(values)
	stdDev := od.calculateStdDev(values, mean)

	if stdDev == 0 {
		return nil
	}

	// Calculate anomaly scores based on distance from mean
	var scores []float64
	for _, value := range values {
		score := math.Abs(value-mean) / stdDev
		scores = append(scores, score)
	}

	// Use a threshold based on the distribution of scores
	scoreThreshold := od.calculatePercentile(scores, 0.9) // Top 10% as outliers

	var outliers []int
	for i, score := range scores {
		if score > scoreThreshold && score > od.threshold {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

// Advanced outlier detection methods

// DetectOutliersWithContext detects outliers considering additional context
func (od *OutlierDetector) DetectOutliersWithContext(values []float64, volumes []float64, timestamps []int64) []int {
	if len(values) != len(volumes) || len(values) != len(timestamps) {
		return od.DetectOutliers(values) // Fallback to simple detection
	}

	// Weight outlier detection by volume and recency
	weightedValues := make([]float64, len(values))
	totalVolume := 0.0
	for _, volume := range volumes {
		totalVolume += volume
	}

	for i := range values {
		volumeWeight := 1.0
		if totalVolume > 0 {
			volumeWeight = 1.0 + (volumes[i]/totalVolume)*10 // Higher volume = higher weight
		}

		// Recency weight (more recent data has higher weight)
		recencyWeight := 1.0
		if len(timestamps) > 1 {
			maxTime := timestamps[0]
			minTime := timestamps[0]
			for _, ts := range timestamps {
				if ts > maxTime {
					maxTime = ts
				}
				if ts < minTime {
					minTime = ts
				}
			}
			if maxTime > minTime {
				recencyWeight = 1.0 + float64(timestamps[i]-minTime)/float64(maxTime-minTime)
			}
		}

		// Apply weights to values for outlier detection
		weightedValues[i] = values[i] * volumeWeight * recencyWeight
	}

	return od.DetectOutliers(weightedValues)
}

// DetectOutliersMultivariate detects outliers using multiple features
func (od *OutlierDetector) DetectOutliersMultivariate(prices, volumes, spreads []float64) []int {
	if len(prices) != len(volumes) || len(prices) != len(spreads) {
		return od.DetectOutliers(prices) // Fallback
	}

	n := len(prices)
	if n < 3 {
		return nil
	}

	// Normalize each feature
	normalizedPrices := od.normalizeValues(prices)
	normalizedVolumes := od.normalizeValues(volumes)
	normalizedSpreads := od.normalizeValues(spreads)

	// Calculate multivariate distance (simplified Mahalanobis-like distance)
	var outliers []int
	for i := 0; i < n; i++ {
		distance := math.Sqrt(
			normalizedPrices[i]*normalizedPrices[i] +
				normalizedVolumes[i]*normalizedVolumes[i] +
				normalizedSpreads[i]*normalizedSpreads[i],
		)

		if distance > od.threshold {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

// Helper methods

func (od *OutlierDetector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func (od *OutlierDetector) calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func (od *OutlierDetector) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	sumSquaredDiff := 0.0
	for _, value := range values {
		diff := value - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values)-1)
	return math.Sqrt(variance)
}

func (od *OutlierDetector) calculatePercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := percentile * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedValues[lower]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

func (od *OutlierDetector) normalizeValues(values []float64) []float64 {
	if len(values) == 0 {
		return values
	}

	mean := od.calculateMean(values)
	stdDev := od.calculateStdDev(values, mean)

	if stdDev == 0 {
		normalized := make([]float64, len(values))
		return normalized // All zeros
	}

	normalized := make([]float64, len(values))
	for i, value := range values {
		normalized[i] = (value - mean) / stdDev
	}

	return normalized
}

// Configuration methods

// SetThreshold updates the outlier detection threshold
func (od *OutlierDetector) SetThreshold(threshold float64) {
	if threshold > 0 {
		od.threshold = threshold
	}
}

// SetMethod updates the outlier detection method
func (od *OutlierDetector) SetMethod(method string) {
	supportedMethods := []string{"z_score", "iqr", "isolation_forest", "modified_z_score"}
	for _, supported := range supportedMethods {
		if method == supported {
			od.method = method
			return
		}
	}
}

// GetMethod returns the current outlier detection method
func (od *OutlierDetector) GetMethod() string {
	return od.method
}

// GetThreshold returns the current outlier detection threshold
func (od *OutlierDetector) GetThreshold() float64 {
	return od.threshold
}

// GetSupportedMethods returns list of supported outlier detection methods
func GetSupportedMethods() []string {
	return []string{
		"z_score",         // Standard Z-score method
		"modified_z_score", // Modified Z-score using MAD (more robust)
		"iqr",             // Interquartile Range method
		"isolation_forest", // Simplified isolation forest
	}
}

// OutlierDetectionResult contains detailed outlier detection results
type OutlierDetectionResult struct {
	OutlierIndices    []int     `json:"outlier_indices"`
	Scores           []float64 `json:"scores"`
	Threshold        float64   `json:"threshold"`
	Method           string    `json:"method"`
	TotalValues      int       `json:"total_values"`
	OutlierCount     int       `json:"outlier_count"`
	OutlierPercentage float64  `json:"outlier_percentage"`
}

// DetectOutliersDetailed performs outlier detection and returns detailed results
func (od *OutlierDetector) DetectOutliersDetailed(values []float64) *OutlierDetectionResult {
	outlierIndices := od.DetectOutliers(values)

	result := &OutlierDetectionResult{
		OutlierIndices:    outlierIndices,
		Threshold:        od.threshold,
		Method:           od.method,
		TotalValues:      len(values),
		OutlierCount:     len(outlierIndices),
	}

	if len(values) > 0 {
		result.OutlierPercentage = float64(len(outlierIndices)) / float64(len(values)) * 100
	}

	// Calculate scores based on method
	switch od.method {
	case "z_score", "modified_z_score":
		result.Scores = od.calculateZScores(values)
	case "iqr":
		result.Scores = od.calculateIQRScores(values)
	default:
		result.Scores = od.calculateZScores(values)
	}

	return result
}

func (od *OutlierDetector) calculateZScores(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}

	mean := od.calculateMean(values)
	stdDev := od.calculateStdDev(values, mean)

	if stdDev == 0 {
		return make([]float64, len(values))
	}

	scores := make([]float64, len(values))
	for i, value := range values {
		scores[i] = math.Abs((value - mean) / stdDev)
	}

	return scores
}

func (od *OutlierDetector) calculateIQRScores(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}

	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	q1 := od.calculatePercentile(sortedValues, 0.25)
	q3 := od.calculatePercentile(sortedValues, 0.75)
	iqr := q3 - q1

	if iqr == 0 {
		return make([]float64, len(values))
	}

	scores := make([]float64, len(values))
	for i, value := range values {
		if value < q1 {
			scores[i] = (q1 - value) / iqr
		} else if value > q3 {
			scores[i] = (value - q3) / iqr
		} else {
			scores[i] = 0 // Within normal range
		}
	}

	return scores
}