package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

func main() {
	http.HandleFunc("/grayscale", grayscaleHandler)
	http.HandleFunc("/items", findWordsHandler)
	fmt.Print("Server running on port 8080...\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func grayscaleHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body as a JPEG image
	img, err := jpeg.Decode(r.Body)
	if err != nil {
		http.Error(w, "Invalid image format", http.StatusBadRequest)
		return
	}

	// Convert the image to grayscale using multiple goroutines
	gray := image.NewGray(img.Bounds())
	numWorkers := 4
	workQueue := make(chan int, gray.Bounds().Max.Y-gray.Bounds().Min.Y)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for y := range workQueue {
				for x := gray.Bounds().Min.X; x < gray.Bounds().Max.X; x++ {
					oldColor := img.At(x, y)
					grayColor := color.GrayModel.Convert(oldColor).(color.Gray)
					gray.Set(x, y, grayColor)
				}
			}
		}()
	}
	for y := gray.Bounds().Min.Y; y < gray.Bounds().Max.Y; y++ {
		workQueue <- y
	}
	close(workQueue)
	wg.Wait()

	// Encode the grayscale image as a JPEG and write it to the response
	w.Header().Set("Content-Type", "image/jpeg")
	if err := jpeg.Encode(w, gray, nil); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func findWordsHandler(w http.ResponseWriter, r *http.Request) {
	words := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape", "honeydew", "imbe", "jackfruit", "kiwi", "lemon", "mango", "nectarine", "orange", "papaya", "quince", "raspberry", "strawberry", "tangerine", "watermelon"}

	query := r.URL.Query().Get("letter")
	if query == "" {
		http.Error(w, "Please provide a letter in the query string", http.StatusBadRequest)
		return
	}

	re := regexp.MustCompile(fmt.Sprintf(`\b\w*%s\w*\b`, query))

	var results []string
	numWorkers := 4
	workQueue := make(chan string, len(words))
	for _, word := range words {
		workQueue <- word
	}
	close(workQueue)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for word := range workQueue {
				if re.MatchString(word) {
					results = append(results, word)
				}
			}
		}()
	}
	wg.Wait()

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(results, "\n")))
}
