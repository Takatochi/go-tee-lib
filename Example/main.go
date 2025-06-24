package main

import (
	"fmt"
	"sync"
	"time"

	// Import your library by its GitHub path.
	// Replace 'your_username' with your actual GitHub username.
	"github.com/Takatochi/go-tee-lib/tee"
)

func main() {
	fmt.Println("Demonstration of the Tee library usage")
	fmt.Println("=======================================")

	// Example 1: Using RunTeeAndProcess for convenience with NewTee
	fmt.Println("\n--- Example using RunTeeAndProcess with NewTee ---")
	dataToSend := []int{10, 20, 30, 40, 50, 60}
	numConsumers := 3
	bufferedSize := 2 // Use 0 for unbuffered, > 0 for buffered channels

	// Function that will be executed by each consumer
	consumerProcessor := func(id int, ch <-chan int) {
		fmt.Printf("Consumer #%d: Starting processing...\n", id)
		for val := range ch {
			fmt.Printf("Consumer #%d: Received value %d\n", id, val)
			// Simulate processing delay
			time.Sleep(time.Millisecond * time.Duration(50+id*20))
		}
		fmt.Printf("Consumer #%d: Channel closed. Exiting.\n", id)
	}

	// Create a Tee instance using the unified NewTee function
	myTee := tee.NewTee[int](numConsumers, bufferedSize)
	// Run the teeing process and wait for consumers to finish
	tee.RunTeeAndProcess(myTee, dataToSend, consumerProcessor)

	fmt.Println("\n--- RunTeeAndProcess example finished ---\n")

	// Example 2: Manual usage of Tee.Run for more control with NewTee
	fmt.Println("--- Manual Tee.Run example with NewTee ---")
	inputCh := make(chan string)
	manualTee := tee.NewTee[string](2, 0) // 2 unbuffered output channels (bufferSize = 0)
	outputChansManual := manualTee.GetOutputChannels()

	// Start the Tee's Run method in a goroutine
	go manualTee.Run(inputCh)

	var wg sync.WaitGroup

	// Start consumers manually
	for i, outCh := range outputChansManual {
		wg.Add(1)
		go func(id int, ch <-chan string) {
			defer wg.Done()
			fmt.Printf("Manual Consumer #%d: Starting...\n", id)
			for val := range ch {
				fmt.Printf("Manual Consumer #%d: Received '%s'\n", id, val)
				time.Sleep(time.Millisecond * 70)
			}
			fmt.Printf("Manual Consumer #%d: Channel closed. Exiting.\n", id)
		}(i+1, outCh)
	}

	// Send data to the input channel
	messages := []string{"Hello", "World", "Go", "Channels", "Tee Pattern"}
	for _, msg := range messages {
		fmt.Printf("Main (Manual): Sending '%s'\n", msg)
		time.Sleep(time.Millisecond * 40)
		inputCh <- msg
	}

	fmt.Println("Main (Manual): Closing input channel")
	close(inputCh) // Important: close the input channel

	wg.Wait() // Wait until all manual consumers have finished
	fmt.Println("--- Manual Tee.Run example finished ---")

	fmt.Println("\nAll demonstrations completed.")
}
