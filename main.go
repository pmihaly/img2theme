package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"runtime"
	"sync"

	"github.com/lucasb-eyer/go-colorful"
)

type BoundedImage struct {
	Image  image.Image
	Bounds image.Rectangle
}

type Settings struct {
	Palette         []colorful.Color
	PaletteAffinity float64
}

func main() {
	inputImagePath := "input.jpg"
	outputImagePath := "output.jpg"

	boundedImage, settings := loadImageAndPalette(inputImagePath)

	mappedImage := image.NewRGBA(boundedImage.Bounds)

	mappedColorByColor := sync.Map{}
	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	rowCh := make(chan int, numCPU)

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mapImageRows(rowCh, boundedImage, settings, mappedImage, &mappedColorByColor)
		}()
	}

	for y := boundedImage.Bounds.Min.Y; y < boundedImage.Bounds.Max.Y; y++ {
		rowCh <- y
	}
	close(rowCh)
	wg.Wait()

	saveMappedImage(outputImagePath, mappedImage)
}

func loadImageAndPalette(inputImagePath string) (BoundedImage, Settings) {
	const paletteAffinity = 0.6

	inputFile, err := os.Open(inputImagePath)
	if err != nil {
		fmt.Println("Error opening input image:", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	img, _, err := image.Decode(inputFile)
	if err != nil {
		fmt.Println("Error decoding input image:", err)
		os.Exit(1)
	}

	palette := []string{
		"#2e3440", "#3b4252", "#434c5e", "#4c566a", "#d8dee9", "#e5e9f0", "#eceff4",
		"#8fbcbb", "#88c0d0", "#81a1c1", "#5e81ac", "#bf616a", "#d08770", "#ebcb8b",
		"#a3be8c", "#b48ead",
	}

	boundedImage := BoundedImage{Image: img, Bounds: img.Bounds()}
	settings := Settings{Palette: parsePalette(palette), PaletteAffinity: paletteAffinity}

	return boundedImage, settings
}

func parsePalette(palette []string) []colorful.Color {
	var parsedPalette []colorful.Color
	for _, hex := range palette {
		c, _ := colorful.Hex(hex)
		parsedPalette = append(parsedPalette, c)
	}
	return parsedPalette
}

func mapImageRows(rowCh chan int, boundedImage BoundedImage, settings Settings, mappedImage *image.RGBA, mappedColorByColor *sync.Map) {
	for row := range rowCh {
		for x := boundedImage.Bounds.Min.X; x < boundedImage.Bounds.Max.X; x++ {
			pixelColor := boundedImage.Image.At(x, row)
			mapAndSetPixelColor(pixelColor, boundedImage, settings, x, row, mappedImage, mappedColorByColor)
		}
	}
}

func mapAndSetPixelColor(pixelColor color.Color, boundedImage BoundedImage, settings Settings, x, y int, mappedImage *image.RGBA, mappedColorByColor *sync.Map) {
	if mappedColor, ok := mappedColorByColor.Load(pixelColor); ok {
		mappedImage.Set(x, y, mappedColor.(color.Color))
		return
	}

	targetLab, _ := colorful.MakeColor(pixelColor)
	minDistance := math.Inf(1)
	var mappedColor colorful.Color

	for _, c := range settings.Palette {
		distance := targetLab.DistanceLab(c)
		if distance < minDistance {
			minDistance = distance
			mappedColor = c
		}
	}

	adjustedColor := colorful.Color{
		R: targetLab.R + (mappedColor.R-targetLab.R)*settings.PaletteAffinity,
		G: targetLab.G + (mappedColor.G-targetLab.G)*settings.PaletteAffinity,
		B: targetLab.B + (mappedColor.B-targetLab.B)*settings.PaletteAffinity,
	}

	mappedImage.Set(x, y, adjustedColor)
	mappedColorByColor.Store(pixelColor, adjustedColor)
}

func saveMappedImage(outputImagePath string, mappedImage image.Image) {
	outputImageFile, err := os.Create(outputImagePath)
	if err != nil {
		fmt.Println("Error creating output image:", err)
		os.Exit(1)
	}
	defer outputImageFile.Close()

	jpeg.Encode(outputImageFile, mappedImage, nil)
	fmt.Println("Image mapping to palette complete. Output saved to", outputImagePath)
}
