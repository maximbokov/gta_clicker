package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	//"github.com/chromedp/cdproto/target"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

const mainurl = "https://clicgta.com/?id=maximbokov&pass=ec9e2034jekDWLsadPI&chatId=118213818"

//const url = "https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go"

type Coord struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func calculateWaitTime(ctx context.Context) int {
	// seconds to fill one percent of energy
	seconds := 10

	// Define the selector for the <span> element
	sel_span := `div.battle__footer-energy-timer > span`
	var onePercentTime string

	// Extract the text content of the <span> element
	chromedp.Run(ctx, chromedp.Text(sel_span, &onePercentTime, chromedp.NodeVisible))

	// get seconds to fill one percent of energy
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(onePercentTime)
	if match != "" {
		seconds, _ = strconv.Atoi(match)
	}

	// overall wait time in seconds
	return seconds * 100
}

func convertIntoSeconds(str string) (int, error) {
	seconds := 0
	parts := strings.Split(str, ":")
	if len(parts) > 3 || str == "" {
		return seconds, fmt.Errorf("invalid time format")
	}

	// parse the string backwards
	for i, k := len(parts), 1; i > 0; i-- {
		t, err := strconv.Atoi(parts[i-1])
		if err != nil {
			return seconds, err
		}

		// calculate seconds
		seconds += t * k
		k *= 60 // convert into seconds
	}

	return seconds, nil
}

func getButtonsTime(ctx context.Context) int {
	// Define the selector for the specific <span> element
	selector := `span.battle__footer-item-cooldown`

	// Variable to store the nodes
	var nodes []*cdp.Node

	// Extract all <span> elements with the specified class
	chromedp.Run(ctx, chromedp.Nodes(selector, &nodes, chromedp.ByQueryAll))

	// Loop through the nodes and extract the text content
	sec_max := 0
	for _, node := range nodes {
		var text string
		chromedp.Run(ctx, chromedp.Text(node.FullXPath(), &text))
		secs, _ := convertIntoSeconds(text)
		if secs > sec_max {
			sec_max = secs
		}
	}

	return sec_max
}

func oneRound(ctx context.Context) (int, error) {
	// Open a new tab and navigate to a different URL
	if err := chromedp.Run(ctx, chromedp.Navigate(mainurl)); err != nil {
		return 0, err
	}

	// wait a little bit after creation of a tab
	time.Sleep(2 * time.Second)

	// check if any dialog is up and we need to click button
	// Define the selector for the button
	selector := `button.btn.btn-black.greetings__btn`

	// Click the button
	chromedp.Run(ctx, chromedp.Click(selector))

	// wait a little bit
	time.Sleep(2 * time.Second)

	// Define the selector for the <span> element
	sel_span := `div.battle__footer-energy-percentage > span`
	var energy, prevEnergy string

	// check if energy is full
	// Extract the text content of the <span> element
	if err := chromedp.Run(ctx, chromedp.Text(sel_span, &energy, chromedp.NodeVisible)); err != nil {
		return 0, err
	}

	//////////////////////////////////////////////////////////////////////////////////////////////////
	// find the main scene coordinates
	// Define the selector for the canvas element
	sel_canvas := `canvas.spine-player-canvas`
	var coord Coord

	// Get the coordinates of the element within the canvas
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.EvaluateAsDevTools(`(() => {
            const canvas = document.querySelector('`+sel_canvas+`');
            const rect = canvas.getBoundingClientRect();
            return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
        })()`, &coord),
	}); err != nil {
		log.Fatal(err)
	}

	//////////////////////////////////////////////////////////////////////////////////////////////////
	// find gun button
	// Define the selector for the image
	sel_gun := `image#image0_154_7415`
	var coord_gun Coord

	// Get the coordinates of the image
	chromedp.Run(ctx, chromedp.Tasks{
		chromedp.EvaluateAsDevTools(`(() => {
            const canvas = document.querySelector('`+sel_gun+`');
            const rect = canvas.getBoundingClientRect();
            return { x: rect.left, y: rect.top };
        })()`, &coord_gun),
	})

	//////////////////////////////////////////////////////////////////////////////////////////////////
	// find machine gun button
	// Define the selector for the image
	sel_mgun := `image#image0_154_7417`
	var coord_mgun Coord

	// Get the coordinates of the image
	chromedp.Run(ctx, chromedp.Tasks{
		chromedp.EvaluateAsDevTools(`(() => {
            const canvas = document.querySelector('`+sel_mgun+`');
            const rect = canvas.getBoundingClientRect();
            return { x: rect.left, y: rect.top };
        })()`, &coord_mgun),
	})

	//////////////////////////////////////////////////////////////////////////////////////////////////
	// THE MAIN ALGO TO CLICK

	// define ticker
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// create a rectangle where to click
	coord.X -= 20
	coord.Y -= 20
	if coord.X < 0 {
		coord.X = 0
	}
	if coord.Y < 0 {
		coord.Y = 0
	}

	// Seed the random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	//////////////////////////////////////////////////////////////////////////////////////////////////
	// a loop to click bandit
	for k := 0; k < 2; k++ {
	outerloop:
		for {
			// define a new number of clicks per second
			clicksPerSecond := 10 + int(r.Float64()*10)
			interval := time.Duration(1e9 / clicksPerSecond) // nanoseconds
			ticker.Reset(interval)                           // set ticker for the interval
			n := int(r.Float64() * 100)                      // number of cicles with a particular frequency
			for i := 0; i < n; i++ {
				select {
				case <-ticker.C:
					x := coord.X + r.Float64()*20 // Random x coordinate from 0 to 20
					y := coord.Y + r.Float64()*20 // Random y coordinate from 0 to 20

					// make a click
					err := chromedp.Run(ctx, chromedp.MouseClickXY(x, y))
					if err != nil {
						log.Fatal(err)
					}

					// check if energy is zero
					// Extract the text content of the <span> element
					if err := chromedp.Run(ctx, chromedp.Text(sel_span, &energy, chromedp.NodeVisible)); err != nil {
						log.Fatal(err)
					}

					// log if energy is droped
					if energy != prevEnergy {
						prevEnergy = energy
						log.Printf("Energy is: %s", energy)
					}

					if energy == "0%" {
						break outerloop
					}
				}
			}
		}

		// click machine gun and then gun and do a loop one more time
		chromedp.Run(ctx, chromedp.MouseClickXY(coord_mgun.X, coord_mgun.Y))

		// wait a second
		time.Sleep(2 * time.Second)
		chromedp.Run(ctx, chromedp.MouseClickXY(coord_gun.X, coord_gun.Y))
	}

	return calculateWaitTime(ctx), nil
}

func main() {
	// create a log file first
	file, err := os.Create("output.log")
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	// Set the log output to the file
	log.SetOutput(file)

	// Connect to the existing Chrome instance
	allocatorCtx, cancelRem := chromedp.NewRemoteAllocator(context.Background(), "http://127.0.0.1:9222")
	defer cancelRem()

	// Seed the random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		// Create a new context
		ctx, cancel := chromedp.NewContext(allocatorCtx)
		waitSeconds, err := oneRound(ctx)

		// check if there was an critical error
		if err != nil {
			log.Fatal(err)
		}

		// calculate time till buttons are active
		secondsButtons := getButtonsTime(ctx) + (60 + int(r.Float64()*240))
		log.Printf("Button time: %d", secondsButtons)

		// calculate random wait time
		waitSeconds = waitSeconds + int(r.Float64()*float64(waitSeconds*2))
		log.Printf("Waiting time: %d", waitSeconds)

		// choose what time is less
		if secondsButtons < waitSeconds {
			waitSeconds = secondsButtons
		}

		// convert seconds into counts for the timer
		waitTime := time.Duration(waitSeconds) * time.Second

		log.Printf("Waiting: %d", waitTime)

		// sleep sometime before closing the tab
		time.Sleep(10 * time.Second)

		// clear previous context(close the tab)
		cancel()

		// wait for the next time to click
		log.Printf("Sleep until: %v", time.Now().Add(waitTime))
		time.Sleep(waitTime)
	}
}
