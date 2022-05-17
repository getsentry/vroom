package quantile

import (
	"math"
	"sort"
)

// Quantile is a collection of possibly weighted data points.
type Quantile struct {
	// Xs is the slice of sample values.
	Xs []float64

	// Weights[i] is the weight of sample Xs[i].  If Weights is
	// nil, all Xs have weight 1.  Weights must have the same
	// length of Xs and all values must be non-negative.
	Weights []float64

	// Sorted indicates that Xs is sorted in ascending order.
	Sorted bool
}

// Bounds returns the minimum and maximum values of xs.
func Bounds(xs []float64) (min float64, max float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	min, max = xs[0], xs[0]
	for _, x := range xs {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	return
}

// Bounds returns the minimum and maximum values of the Quantile.
//
// If the Quantile is weighted, this ignores samples with zero weight.
//
// This is constant time if s.Sorted and there are no zero-weighted
// values.
func (q Quantile) Bounds() (min float64, max float64) {
	if len(q.Xs) == 0 || (!q.Sorted && q.Weights == nil) {
		return Bounds(q.Xs)
	}

	if q.Sorted {
		if q.Weights == nil {
			return q.Xs[0], q.Xs[len(q.Xs)-1]
		}
		min, max = 0, 0
		for i, w := range q.Weights {
			if w != 0 {
				min = q.Xs[i]
				break
			}
		}
		if math.IsNaN(min) {
			return
		}
		for i := range q.Weights {
			if q.Weights[len(q.Weights)-i-1] != 0 {
				max = q.Xs[len(q.Weights)-i-1]
				break
			}
		}
	} else {
		min, max = math.Inf(1), math.Inf(-1)
		for i, x := range q.Xs {
			w := q.Weights[i]
			if x < min && w != 0 {
				min = x
			}
			if x > max && w != 0 {
				max = x
			}
		}
		if math.IsInf(min, 0) {
			min, max = 0, 0
		}
	}
	return
}

// vecSum returns the sum of xs.
func vecSum(xs []float64) float64 {
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum
}

// Sum returns the (possibly weighted) sum of the Quantile.
func (q Quantile) Sum() float64 {
	if q.Weights == nil {
		return vecSum(q.Xs)
	}
	sum := 0.0
	for i, x := range q.Xs {
		sum += x * q.Weights[i]
	}
	return sum
}

// Weight returns the total weight of the Sasmple.
func (q Quantile) Weight() float64 {
	if q.Weights == nil {
		return float64(len(q.Xs))
	}
	return vecSum(q.Weights)
}

// Mean returns the arithmetic mean of xs.
func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	m := 0.0
	for i, x := range xs {
		m += (x - m) / float64(i+1)
	}
	return m
}

// Mean returns the arithmetic mean of the Quantile.
func (q Quantile) Mean() float64 {
	if len(q.Xs) == 0 || q.Weights == nil {
		return Mean(q.Xs)
	}

	m, wsum := 0.0, 0.0
	for i, x := range q.Xs {
		// Use weighted incremental mean:
		//   m_i = (1 - w_i/wsum_i) * m_(i-1) + (w_i/wsum_i) * x_i
		//       = m_(i-1) + (x_i - m_(i-1)) * (w_i/wsum_i)
		w := q.Weights[i]
		wsum += w
		m += (x - m) * w / wsum
	}
	return m
}

// GeoMean returns the geometric mean of xs. xs must be positive.
func GeoMean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	m := 0.0
	for i, x := range xs {
		if x <= 0 {
			return math.NaN()
		}
		lx := math.Log(x)
		m += (lx - m) / float64(i+1)
	}
	return math.Exp(m)
}

// GeoMean returns the geometric mean of the Quantile. All samples
// values must be positive.
func (q Quantile) GeoMean() float64 {
	if len(q.Xs) == 0 || q.Weights == nil {
		return GeoMean(q.Xs)
	}

	m, wsum := 0.0, 0.0
	for i, x := range q.Xs {
		w := q.Weights[i]
		wsum += w
		lx := math.Log(x)
		m += (lx - m) * w / wsum
	}
	return math.Exp(m)
}

// Variance returns the sample variance of xs.
func Variance(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	} else if len(xs) <= 1 {
		return 0
	}

	// Based on Wikipedia's presentation of Welford 1962
	// (http://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Online_algorithm).
	// This is more numerically stable than the standard two-pass
	// formula and not prone to massive cancellation.
	mean, M2 := 0.0, 0.0
	for n, x := range xs {
		delta := x - mean
		mean += delta / float64(n+1)
		M2 += delta * (x - mean)
	}
	return M2 / float64(len(xs)-1)
}

func (q Quantile) Variance() float64 {
	if len(q.Xs) == 0 || q.Weights == nil {
		return Variance(q.Xs)
	}
	// TODO(austin)
	panic("Weighted Variance not implemented")
}

// StdDev returns the sample standard deviation of xs.
func StdDev(xs []float64) float64 {
	return math.Sqrt(Variance(xs))
}

// StdDev returns the sample standard deviation of the Quantile.
func (q Quantile) StdDev() float64 {
	if len(q.Xs) == 0 || q.Weights == nil {
		return StdDev(q.Xs)
	}
	// TODO(austin)
	panic("Weighted StdDev not implemented")
}

// Percentile returns the pctileth value from the Quantile. This uses
// interpolation method R8 from Hyndman and Fan (1996).
//
// pctile will be capped to the range [0, 1]. If len(xs) == 0 or all
// weights are 0, returns NaN.
//
// Percentile(0.5) is the median. Percentile(0.25) and
// Percentile(0.75) are the first and third quartiles, respectively.
//
// This is constant time if s.Sorted and s.Weights == nil.
func (q Quantile) Percentile(pctile float64) float64 {
	if len(q.Xs) == 0 {
		return 0
	} else if pctile <= 0 {
		min, _ := q.Bounds()
		return min
	} else if pctile >= 1 {
		_, max := q.Bounds()
		return max
	}

	if !q.Sorted {
		// TODO(austin) Use select algorithm instead
		q = *q.Copy().Sort()
	}

	if q.Weights == nil {
		N := float64(len(q.Xs))
		n := 1/3.0 + pctile*(N+1/3.0) // R8
		kf, frac := math.Modf(n)
		k := int(kf)
		if k <= 0 {
			return q.Xs[0]
		} else if k >= len(q.Xs) {
			return q.Xs[len(q.Xs)-1]
		}
		return q.Xs[k-1] + frac*(q.Xs[k]-q.Xs[k-1])
	} else {
		// TODO(austin): Implement interpolation

		target := q.Weight() * pctile

		// TODO(austin) If we had cumulative weights, we could
		// do this in log time.
		for i, weight := range q.Weights {
			target -= weight
			if target < 0 {
				return q.Xs[i]
			}
		}
		return q.Xs[len(q.Xs)-1]
	}
}

// IQR returns the interquartile range of the Quantile.
//
// This is constant time if s.Sorted and s.Weights == nil.
func (q Quantile) IQR() float64 {
	if !q.Sorted {
		q = *q.Copy().Sort()
	}
	return q.Percentile(0.75) - q.Percentile(0.25)
}

type sampleSorter struct {
	xs      []float64
	weights []float64
}

func (p *sampleSorter) Len() int {
	return len(p.xs)
}

func (p *sampleSorter) Less(i, j int) bool {
	return p.xs[i] < p.xs[j]
}

func (p *sampleSorter) Swap(i, j int) {
	p.xs[i], p.xs[j] = p.xs[j], p.xs[i]
	p.weights[i], p.weights[j] = p.weights[j], p.weights[i]
}

// Sort sorts the samples in place in s and returns s.
//
// A sorted sample improves the performance of some algorithms.
func (q *Quantile) Sort() *Quantile {
	if q.Sorted || sort.Float64sAreSorted(q.Xs) {
		// All set
	} else if q.Weights == nil {
		sort.Float64s(q.Xs)
	} else {
		sort.Sort(&sampleSorter{q.Xs, q.Weights})
	}
	q.Sorted = true
	return q
}

// Copy returns a copy of the Quantile.
//
// The returned Quantile shares no data with the original, so they can
// be modified (for example, sorted) independently.
func (q Quantile) Copy() *Quantile {
	xs := make([]float64, len(q.Xs))
	copy(xs, q.Xs)

	weights := []float64(nil)
	if q.Weights != nil {
		weights = make([]float64, len(q.Weights))
		copy(weights, q.Weights)
	}

	return &Quantile{xs, weights, q.Sorted}
}

func (q *Quantile) Add(v ...float64) {
	q.Xs = append(q.Xs, v...)
}
