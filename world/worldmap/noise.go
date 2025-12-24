package worldmap

import (
	"math"
	"math/rand"
)

// NoiseGenerator provides reusable noise generation utilities
type NoiseGenerator struct {
	seed int64
	rng  *rand.Rand
}

// NewNoiseGenerator creates a noise generator with the given seed
func NewNoiseGenerator(seed int64) *NoiseGenerator {
	return &NoiseGenerator{
		seed: seed,
		rng:  rand.New(rand.NewSource(seed)),
	}
}

// GeneratePerlinNoise creates a 2D Perlin noise map with multiple octaves (fBm)
// Parameters:
//   - width, height: dimensions of the noise map
//   - scale: base frequency (smaller = larger features, typically 0.05-0.2)
//   - octaves: number of noise layers to combine (3-6 typical)
//   - persistence: amplitude falloff per octave (0.5 typical)
func (ng *NoiseGenerator) GeneratePerlinNoise(width, height int, scale float64, octaves int, persistence float64) [][]float64 {
	noiseMap := make([][]float64, height)
	for i := range noiseMap {
		noiseMap[i] = make([]float64, width)
	}

	// Generate gradients for each octave
	gradientLayers := make([][][]float64, octaves)
	currentScale := scale
	for o := 0; o < octaves; o++ {
		gridWidth := int(math.Ceil(float64(width)*currentScale)) + 2
		gridHeight := int(math.Ceil(float64(height)*currentScale)) + 2
		gradientLayers[o] = ng.generateGradients(gridWidth, gridHeight)
		currentScale *= 2.0 // Each octave doubles the frequency
	}

	// Combine octaves
	maxAmplitude := 0.0
	amplitude := 1.0
	for o := 0; o < octaves; o++ {
		maxAmplitude += amplitude
		amplitude *= persistence
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := 0.0
			amplitude = 1.0
			currentScale = scale

			for o := 0; o < octaves; o++ {
				gridWidth := int(math.Ceil(float64(width)*currentScale)) + 2
				gridHeight := int(math.Ceil(float64(height)*currentScale)) + 2
				octaveValue := ng.perlinInterpolate(float64(x), float64(y), currentScale, gradientLayers[o], gridWidth, gridHeight)
				value += octaveValue * amplitude

				amplitude *= persistence
				currentScale *= 2.0
			}

			// Normalize to 0-1 range
			noiseMap[y][x] = (value/maxAmplitude + 1.0) / 2.0
			noiseMap[y][x] = clamp(noiseMap[y][x], 0.0, 1.0)
		}
	}

	return noiseMap
}

// GenerateSimpleNoise creates a single-octave Perlin noise map
func (ng *NoiseGenerator) GenerateSimpleNoise(width, height int, scale float64) [][]float64 {
	return ng.GeneratePerlinNoise(width, height, scale, 1, 1.0)
}

// ApplyDomainWarping distorts a noise map using another noise field
// This creates organic, non-uniform shapes
// Parameters:
//   - noiseMap: the map to warp
//   - warpAmount: how much to distort (2.0-5.0 typical)
//   - warpScale: frequency of the warp noise (0.05-0.15 typical)
func (ng *NoiseGenerator) ApplyDomainWarping(noiseMap [][]float64, warpAmount, warpScale float64) [][]float64 {
	height := len(noiseMap)
	if height == 0 {
		return noiseMap
	}
	width := len(noiseMap[0])

	// Generate warp noise fields for X and Y displacement
	warpX := ng.GenerateSimpleNoise(width, height, warpScale)
	warpY := ng.GenerateSimpleNoise(width, height, warpScale*1.3) // Slightly different scale for variety

	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Calculate warped coordinates
			dx := (warpX[y][x] - 0.5) * 2.0 * warpAmount
			dy := (warpY[y][x] - 0.5) * 2.0 * warpAmount

			srcX := float64(x) + dx
			srcY := float64(y) + dy

			// Sample from warped position with bilinear interpolation
			result[y][x] = ng.sampleBilinear(noiseMap, srcX, srcY, width, height)
		}
	}

	return result
}

// GenerateVoronoi creates a Voronoi diagram
// Returns:
//   - distanceMap: normalized distance to nearest point (0-1)
//   - regionMap: ID of the nearest point for each cell
func (ng *NoiseGenerator) GenerateVoronoi(width, height, numPoints int) ([][]float64, [][]int) {
	distanceMap := make([][]float64, height)
	regionMap := make([][]int, height)
	for i := range distanceMap {
		distanceMap[i] = make([]float64, width)
		regionMap[i] = make([]int, width)
	}

	// Generate random seed points
	points := make([][2]int, numPoints)
	for i := 0; i < numPoints; i++ {
		points[i] = [2]int{
			ng.rng.Intn(width),
			ng.rng.Intn(height),
		}
	}

	// Calculate distance and region for each cell
	maxDist := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			minDist := math.MaxFloat64
			nearestPoint := 0

			for i, point := range points {
				dx := float64(x - point[0])
				dy := float64(y - point[1])
				dist := math.Sqrt(dx*dx + dy*dy)

				if dist < minDist {
					minDist = dist
					nearestPoint = i
				}
			}

			distanceMap[y][x] = minDist
			regionMap[y][x] = nearestPoint

			if minDist > maxDist {
				maxDist = minDist
			}
		}
	}

	// Normalize distances to 0-1
	if maxDist > 0 {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				distanceMap[y][x] /= maxDist
			}
		}
	}

	return distanceMap, regionMap
}

// GenerateVoronoiWithJitter creates Voronoi with Lloyd relaxation for more uniform distribution
func (ng *NoiseGenerator) GenerateVoronoiWithJitter(width, height, numPoints, relaxIterations int) ([][]float64, [][]int) {
	// Generate initial points
	points := make([][2]float64, numPoints)
	for i := 0; i < numPoints; i++ {
		points[i] = [2]float64{
			float64(ng.rng.Intn(width)),
			float64(ng.rng.Intn(height)),
		}
	}

	// Lloyd relaxation - move points toward centroid of their region
	for iter := 0; iter < relaxIterations; iter++ {
		// Count cells and sum positions per region
		counts := make([]int, numPoints)
		sumX := make([]float64, numPoints)
		sumY := make([]float64, numPoints)

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				minDist := math.MaxFloat64
				nearestPoint := 0

				for i, point := range points {
					dx := float64(x) - point[0]
					dy := float64(y) - point[1]
					dist := dx*dx + dy*dy // Squared distance is fine for comparison

					if dist < minDist {
						minDist = dist
						nearestPoint = i
					}
				}

				counts[nearestPoint]++
				sumX[nearestPoint] += float64(x)
				sumY[nearestPoint] += float64(y)
			}
		}

		// Move points toward centroids
		for i := range points {
			if counts[i] > 0 {
				points[i][0] = sumX[i] / float64(counts[i])
				points[i][1] = sumY[i] / float64(counts[i])
			}
		}
	}

	// Final Voronoi calculation with relaxed points
	distanceMap := make([][]float64, height)
	regionMap := make([][]int, height)
	for i := range distanceMap {
		distanceMap[i] = make([]float64, width)
		regionMap[i] = make([]int, width)
	}

	maxDist := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			minDist := math.MaxFloat64
			nearestPoint := 0

			for i, point := range points {
				dx := float64(x) - point[0]
				dy := float64(y) - point[1]
				dist := math.Sqrt(dx*dx + dy*dy)

				if dist < minDist {
					minDist = dist
					nearestPoint = i
				}
			}

			distanceMap[y][x] = minDist
			regionMap[y][x] = nearestPoint

			if minDist > maxDist {
				maxDist = minDist
			}
		}
	}

	// Normalize
	if maxDist > 0 {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				distanceMap[y][x] /= maxDist
			}
		}
	}

	return distanceMap, regionMap
}

// BlendNoiseMaps combines two noise maps with given weights
func BlendNoiseMaps(map1, map2 [][]float64, weight1, weight2 float64) [][]float64 {
	height := len(map1)
	if height == 0 {
		return map1
	}
	width := len(map1[0])

	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	totalWeight := weight1 + weight2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			result[y][x] = (map1[y][x]*weight1 + map2[y][x]*weight2) / totalWeight
		}
	}

	return result
}

// --- Internal helper functions ---

func (ng *NoiseGenerator) generateGradients(width, height int) [][]float64 {
	gradients := make([][]float64, height)
	for i := range gradients {
		gradients[i] = make([]float64, width)
		for j := range gradients[i] {
			gradients[i][j] = ng.rng.Float64() * 2.0 * math.Pi
		}
	}
	return gradients
}

func (ng *NoiseGenerator) perlinInterpolate(x, y, scale float64, gradients [][]float64, gridWidth, gridHeight int) float64 {
	scaledX := x * scale
	scaledY := y * scale

	gridX := int(scaledX)
	gridY := int(scaledY)

	if gridX < 0 || gridX >= gridWidth-1 || gridY < 0 || gridY >= gridHeight-1 {
		return 0
	}

	fx := scaledX - float64(gridX)
	fy := scaledY - float64(gridY)

	// Smoothstep interpolation
	u := fx * fx * (3.0 - 2.0*fx)
	v := fy * fy * (3.0 - 2.0*fy)

	// Sample gradients
	g00 := math.Sin(gradients[gridY][gridX])
	g10 := math.Sin(gradients[gridY][gridX+1])
	g01 := math.Sin(gradients[gridY+1][gridX])
	g11 := math.Sin(gradients[gridY+1][gridX+1])

	// Bilinear interpolation
	nx0 := g00*(1-u) + g10*u
	nx1 := g01*(1-u) + g11*u
	return nx0*(1-v) + nx1*v
}

func (ng *NoiseGenerator) sampleBilinear(noiseMap [][]float64, x, y float64, width, height int) float64 {
	// Clamp coordinates
	x = clamp(x, 0, float64(width-1))
	y = clamp(y, 0, float64(height-1))

	x0 := int(x)
	y0 := int(y)
	x1 := min(x0+1, width-1)
	y1 := min(y0+1, height-1)

	fx := x - float64(x0)
	fy := y - float64(y0)

	// Bilinear interpolation
	v00 := noiseMap[y0][x0]
	v10 := noiseMap[y0][x1]
	v01 := noiseMap[y1][x0]
	v11 := noiseMap[y1][x1]

	top := v00*(1-fx) + v10*fx
	bottom := v01*(1-fx) + v11*fx

	return top*(1-fy) + bottom*fy
}

func clamp(value, minVal, maxVal float64) float64 {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

// ============================================================================
// WAVELET NOISE FUNCTIONS
// ============================================================================

// WaveletBand represents a single frequency band in wavelet decomposition
type WaveletBand struct {
	Data      [][]float64
	Scale     int     // 1 = finest detail, higher = coarser
	Amplitude float64 // Weight of this band
}

// GenerateWaveletNoise creates wavelet-based noise with precise frequency control
// Parameters:
//   - width, height: dimensions
//   - numBands: number of wavelet bands (3-6 typical)
//   - baseAmplitude: amplitude of lowest frequency band
//   - amplitudeDecay: how much amplitude decreases per band (0.5 = halve each band)
func (ng *NoiseGenerator) GenerateWaveletNoise(width, height, numBands int, baseAmplitude, amplitudeDecay float64) [][]float64 {
	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	// Generate each wavelet band
	bands := make([]WaveletBand, numBands)
	amplitude := baseAmplitude

	for b := 0; b < numBands; b++ {
		scale := 1 << (numBands - 1 - b) // Coarsest to finest: 16, 8, 4, 2, 1
		bands[b] = WaveletBand{
			Data:      ng.generateWaveletBand(width, height, scale),
			Scale:     scale,
			Amplitude: amplitude,
		}
		amplitude *= amplitudeDecay
	}

	// Combine all bands
	totalAmplitude := 0.0
	for _, band := range bands {
		totalAmplitude += band.Amplitude
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := 0.0
			for _, band := range bands {
				value += band.Data[y][x] * band.Amplitude
			}
			// Normalize to 0-1
			result[y][x] = (value/totalAmplitude + 1.0) / 2.0
			result[y][x] = clamp(result[y][x], 0.0, 1.0)
		}
	}

	return result
}

// generateWaveletBand creates a single band of wavelet noise at given scale
func (ng *NoiseGenerator) generateWaveletBand(width, height, scale int) [][]float64 {
	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	// Generate coarse grid at this scale
	gridW := (width + scale - 1) / scale
	gridH := (height + scale - 1) / scale

	// Random coefficients for this band
	coeffs := make([][]float64, gridH+1)
	for i := range coeffs {
		coeffs[i] = make([]float64, gridW+1)
		for j := range coeffs[i] {
			coeffs[i][j] = ng.rng.Float64()*2.0 - 1.0 // -1 to 1
		}
	}

	// Interpolate using wavelet basis (Haar-like with smoothing)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Grid coordinates
			gx := float64(x) / float64(scale)
			gy := float64(y) / float64(scale)

			// Integer grid positions
			gx0 := int(gx)
			gy0 := int(gy)
			gx1 := min(gx0+1, gridW)
			gy1 := min(gy0+1, gridH)

			// Fractional part with wavelet interpolation
			fx := gx - float64(gx0)
			fy := gy - float64(gy0)

			// Wavelet basis function (smooth step with sharper transitions)
			wx := waveletBasis(fx)
			wy := waveletBasis(fy)

			// Bilinear with wavelet weights
			v00 := coeffs[gy0][gx0]
			v10 := coeffs[gy0][gx1]
			v01 := coeffs[gy1][gx0]
			v11 := coeffs[gy1][gx1]

			top := v00*(1-wx) + v10*wx
			bottom := v01*(1-wx) + v11*wx
			result[y][x] = top*(1-wy) + bottom*wy
		}
	}

	return result
}

// waveletBasis provides the wavelet interpolation function
// This creates sharper band separation than smoothstep
func waveletBasis(t float64) float64 {
	// Quintic interpolation for C2 continuity with wavelet-like response
	// This provides better band-limiting than cubic
	t2 := t * t
	t3 := t2 * t
	return 6*t2*t3 - 15*t2*t2 + 10*t3
}

// GenerateWaveletNoiseWithMask creates wavelet noise with per-band spatial masks
// This allows different regions to have different frequency characteristics
func (ng *NoiseGenerator) GenerateWaveletNoiseWithMask(width, height, numBands int, masks [][][]float64) [][]float64 {
	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	// Generate bands (3D: band -> y -> x)
	bands := make([][][]float64, numBands)
	for b := 0; b < numBands; b++ {
		scale := 1 << (numBands - 1 - b)
		bands[b] = ng.generateWaveletBand(width, height, scale)
	}

	// Combine with masks
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := 0.0
			totalWeight := 0.0

			for b := 0; b < numBands; b++ {
				weight := 1.0
				if b < len(masks) && masks[b] != nil {
					weight = masks[b][y][x]
				}
				value += bands[b][y][x] * weight
				totalWeight += weight
			}

			if totalWeight > 0 {
				result[y][x] = (value/totalWeight + 1.0) / 2.0
			} else {
				result[y][x] = 0.5
			}
			result[y][x] = clamp(result[y][x], 0.0, 1.0)
		}
	}

	return result
}

// GenerateRidgedWavelet creates ridge-like features using absolute wavelet noise
// Great for mountain ridges, river networks, canyon systems
func (ng *NoiseGenerator) GenerateRidgedWavelet(width, height, numBands int, ridgeOffset, ridgeSharpness float64) [][]float64 {
	// Generate base wavelet noise
	base := ng.GenerateWaveletNoise(width, height, numBands, 1.0, 0.5)

	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Convert to -1 to 1 range
			v := base[y][x]*2.0 - 1.0

			// Ridge transformation: 1 - |v| creates ridges at zero crossings
			ridge := 1.0 - math.Abs(v)

			// Sharpen the ridges
			ridge = math.Pow(ridge, ridgeSharpness)

			// Apply offset and normalize
			result[y][x] = clamp(ridge+ridgeOffset, 0.0, 1.0)
		}
	}

	return result
}

// GenerateTurbulentWavelet creates turbulent noise by summing absolute values
// Good for clouds, fire, chaotic terrain
func (ng *NoiseGenerator) GenerateTurbulentWavelet(width, height, numBands int) [][]float64 {
	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	totalAmplitude := 0.0
	amplitude := 1.0

	for b := 0; b < numBands; b++ {
		scale := 1 << (numBands - 1 - b)
		band := ng.generateWaveletBand(width, height, scale)

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// Sum absolute values for turbulence
				result[y][x] += math.Abs(band[y][x]) * amplitude
			}
		}

		totalAmplitude += amplitude
		amplitude *= 0.5
	}

	// Normalize
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			result[y][x] = clamp(result[y][x]/totalAmplitude, 0.0, 1.0)
		}
	}

	return result
}

// GenerateHybridWavelet combines regular and ridged wavelet noise
// Creates varied terrain with both smooth areas and sharp features
func (ng *NoiseGenerator) GenerateHybridWavelet(width, height, numBands int, ridgeWeight float64) [][]float64 {
	regular := ng.GenerateWaveletNoise(width, height, numBands, 1.0, 0.5)
	ridged := ng.GenerateRidgedWavelet(width, height, numBands, 0.0, 2.0)

	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Blend based on ridgeWeight
			result[y][x] = regular[y][x]*(1.0-ridgeWeight) + ridged[y][x]*ridgeWeight
		}
	}

	return result
}
