// Package tee provides a generic Tee implementation for Go channels.
// It allows duplicating values from a single input channel to multiple output channels.
package tee

import (
	"sync"
)

// ChannelTee # Typical usage
//
// ```go
// package main
//
// import (
//
//	"fmt"
//	"time"
//	"github.com/Takatochi/go-tee-lib/tee"
//
// )
//
//	func main() {
//		// --- Example 1: Using RunTeeAndProcess for convenience with NewTee ---
//		fmt.Println("--- Example using RunTeeAndProcess with NewTee ---")
//		dataToSend := []int{10, 20, 30}
//		numConsumers := 2
//		bufferedSize := 1 // Use 0 for unbuffered, > 0 for buffered
//
//		// Define the consumer processing function
//		consumerProcessor := func(id int, ch <-chan int) {
//			fmt.Printf("Consumer #%d: Starting processing...\n", id)
//			for val := range ch {
//				fmt.Printf("Consumer #%d: Received value %d\n", id, val)
//				time.Sleep(time.Millisecond * 50) // Simulate work
//			}
//			fmt.Printf("Consumer #%d: Channel closed. Exiting.\n", id)
//		}
//
//		// Create a Tee instance using the unified NewTee function
//		myTee := tee.NewTee[int](numConsumers, bufferedSize)
//		// Run the teeing process and wait for consumers to finish
//		tee.RunTeeAndProcess(myTee, dataToSend, consumerProcessor)
//
//		fmt.Println("\n--- RunTeeAndProcess example finished ---\n")
//
//		// --- Example 2: Manual usage of Tee.Run for more control with NewTee ---
//		fmt.Println("--- Manual Tee.Run example with NewTee ---")
//		inputCh := make(chan string)
//		manualTee := tee.NewTee[string](2, 0) // 2 unbuffered output channels (bufferSize = 0)
//		outputChansManual := manualTee.GetOutputChannels()
//
//		// Start the Tee's Run method in a goroutine
//		go manualTee.Run(inputCh)
//
//		var wg sync.WaitGroup
//
//		// Start consumers manually
//		for i, outCh := range outputChansManual {
//			wg.Add(1)
//			go func(id int, ch <-chan string) {
//				defer wg.Done()
//				fmt.Printf("Manual Consumer #%d: Starting...\n", id)
//				for val := range ch {
//					fmt.Printf("Manual Consumer #%d: Received '%s'\n", id, val)
//					time.Sleep(time.Millisecond * 70)
//				}
//				fmt.Printf("Manual Consumer #%d: Channel closed. Exiting.\n", id)
//			}(i+1, outCh)
//		}
//
//		// Send data to the input channel
//		messages := []string{"Hello", "World", "Go"}
//		for _, msg := range messages {
//			fmt.Printf("Main (Manual): Sending '%s'\n", msg)
//			inputCh <- msg
//			time.Sleep(time.Millisecond * 40)
//		}
//
//		fmt.Println("Main (Manual): Closing input channel")
//		close(inputCh) // Crucial: close the input channel
//
//		wg.Wait() // Wait for all manual consumers to finish
//		fmt.Println("--- Manual Tee.Run example finished ---")
//	}
//
// ```
type ChannelTee[T any] interface {
	// GetOutputChannels returns the slice of output channels from the Tee.
	GetOutputChannels() []chan T
	// Run starts the teeing process. It reads from the inputChannel
	// and duplicates the messages to all output channels.
	// This function should typically be run in a goroutine.
	// It will block until the inputChannel is closed and all values are processed.
	// Once the inputChannel is closed, all output channels will also be closed.
	Run(inputChannel <-chan T)
}

// Tee [T any] struct holds the output channels.
// The WaitGroup for coordinating consumer goroutines is managed by the caller,
// not by the Tee itself, as per Go's idiomatic channel patterns.
type Tee[T any] struct {
	outputChannels []chan T
}

// NewTee creates a new Tee instance with the specified number of output channels.
// If bufferSize is 0, unbuffered channels are created. Otherwise, buffered channels
// with the given bufferSize are created.
// It returns a ChannelTee interface, specifically a *Tee[T] which implements the interface.
func NewTee[T any](numOutputChans int, bufferSize int) ChannelTee[T] {
	outputChannels := make([]chan T, numOutputChans)
	for i := 0; i < numOutputChans; i++ {
		if bufferSize > 0 {
			outputChannels[i] = make(chan T, bufferSize)
		} else {
			outputChannels[i] = make(chan T)
		}
	}
	return &Tee[T]{
		outputChannels: outputChannels,
	}
}

// GetOutputChannels returns the slice of output channels from the Tee.
func (t *Tee[T]) GetOutputChannels() []chan T {
	return t.outputChannels
}

// Run starts the teeing process. It reads from the inputChannel
// and duplicates the messages to all output channels.
// This function should typically be run in a goroutine.
// It will block until the inputChannel is closed and all values are processed.
// Once the inputChannel is closed, all output channels will also be closed.
//
// This is the correct implementation of the Tee pattern:
// A single goroutine reads from the input channel and then distributes
// each received value to all its output channels.
func (t *Tee[T]) Run(inputChannel <-chan T) {
	// The main goroutine that reads from the input channel and distributes data
	go func() {
		defer func() {
			// After the input channel is closed and all values have been read,
			// we must close all output channels. This ensures that consumer
			// goroutines reading from these channels will exit their `for range` loops.
			for _, outCh := range t.outputChannels {
				close(outCh)
			}
		}()

		for val := range inputChannel {
			// For each value from the input channel, send it to every output channel.
			for _, outCh := range t.outputChannels {
				// This operation might block if an outCh is full or not being read from.
				// For production applications, a select statement with a timeout or context
				// could be added for better control and to prevent deadlocks in complex scenarios.
				outCh <- val
			}
		}
	}()
}

// RunTeeAndProcess [T any] is a convenience function that orchestrates the entire teeing process.
// It takes a ChannelTee instance, a slice of items to send, and a process function for consumers.
// It sends the items through the provided tee instance and then spawns consumer goroutines
// for each output channel using the provided process function.
// It blocks until all items are processed by all consumers.
//
// Parameters:
//   - teeInstance: An instance of ChannelTee (e.g., created by NewTeeBufferSize or NewTeeUnbufferSize),
//     or a custom implementation of ChannelTee.
//   - items: The slice of data to be sent through the tee.
//   - processFn: A function that will be executed for each output channel.
//     It receives the consumer's ID and the output channel (`<-chan T`) as arguments.
func RunTeeAndProcess[T any](teeInstance ChannelTee[T], items []T, processFn func(id int, ch <-chan T)) {
	// Create an input channel for the data
	inputCh := make(chan T)

	outputChans := teeInstance.GetOutputChannels()

	var wg sync.WaitGroup // WaitGroup for waiting for consumer goroutines to complete

	// Start the Tee in a separate goroutine to distribute the data
	go teeInstance.Run(inputCh)

	// Start goroutines to read from the output channels (consumers)
	for i, outCh := range outputChans {
		wg.Add(1) // Add a counter for each consumer goroutine
		go func(id int, ch <-chan T) {
			defer wg.Done()   // Decrement the WaitGroup counter when the goroutine finishes
			processFn(id, ch) // Call the provided processing function for this channel
		}(i+1, outCh) // Pass the consumer ID (starting from 1) and the channel
	}

	// Send all items to the input channel in a separate goroutine
	go func() {
		for _, item := range items {
			inputCh <- item
		}
		close(inputCh) // Important: close the input channel after sending all data
	}()

	wg.Wait() // Wait until all consumer goroutines have completed their work
	// fmt.Println("All consumer goroutines have finished.") // This log is now in the example main.go
}
