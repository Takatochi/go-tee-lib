// Package tee provides a generic Tee implementation for Go channels.
// It allows duplicating values from a single input channel to multiple output channels.
//
// The Tee pattern is useful for:
// - Parallel processing of the same data by different consumers
// - Distributing data stream to multiple independent processors
// - Creating data copies for logging, monitoring, and main processing
package tee

import (
	"context"
	"sync"
)

// ChannelTee interface for implementing the Tee pattern with channels.
//
// Usage example:
//
//	// Creating Tee with 3 consumers and buffered channels
//	teeInstance := tee.NewTee[int](3, 2)
//
//	// Processing function for each consumer
//	processor := func(id int, ch <-chan int) {
//		fmt.Printf("Consumer #%d: Starting processing\n", id)
//		for value := range ch {
//			fmt.Printf("Consumer #%d received: %d\n", id, value)
//		}
//	}
//
//	// Start processing with data
//	data := []int{10, 20, 30, 40, 50}
//	ctx := context.Background()
//	tee.RunTeeAndProcess(ctx, teeInstance, data, processor)
type ChannelTee[T any] interface {
	// GetOutputChannels returns the slice of output channels from the Tee.
	GetOutputChannels() []chan T

	// Run starts the teeing process. It reads from the inputChannel
	// and duplicates the messages to all output channels.
	// This function should typically be run in a goroutine.
	// It will block until the inputChannel is closed and all values are processed.
	// Once the inputChannel is closed, all output channels will also be closed.
	Run(ctx context.Context, inputChannel <-chan T)
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
func (t *Tee[T]) Run(ctx context.Context, inputChannel <-chan T) {
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

		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-inputChannel:
				if !ok {
					return
				}
				// For each value from the input channel, send it to every output channel.
				for _, outCh := range t.outputChannels {
					select {
					case <-ctx.Done():
						return
					case outCh <- val:
						// Successfully sent
					}
				}
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
//   - ctx: Context for cancellation
//   - teeInstance: An instance of ChannelTee or custom implementation
//   - items: The slice of data to be sent through the tee
//   - processFn: Function executed for each output channel, receives consumer ID and output channel
func RunTeeAndProcess[T any](ctx context.Context, teeInstance ChannelTee[T], items []T, processFn func(ctx context.Context, id int, ch <-chan T)) {
	// Create an input channel for the data
	inputCh := make(chan T)

	outputChans := teeInstance.GetOutputChannels()

	var wg sync.WaitGroup // WaitGroup for waiting for consumer goroutines to complete

	// Start the Tee in a separate goroutine to distribute the data
	go teeInstance.Run(ctx, inputCh)

	// Start goroutines to read from the output channels (consumers)
	for i, outCh := range outputChans {
		wg.Add(1) // Add a counter for each consumer goroutine
		go func(id int, ch <-chan T) {
			defer wg.Done()        // Decrement the WaitGroup counter when the goroutine finishes
			processFn(ctx, id, ch) // Call the provided processing function for this channel
		}(i+1, outCh) // Pass the consumer ID (starting from 1) and the channel
	}

	// Send all items to the input channel in a separate goroutine
	go func() {
		defer close(inputCh) // Important: close the input channel after sending all data
		for _, item := range items {
			select {
			case <-ctx.Done():
				return // Exit early if context is cancelled
			case inputCh <- item:
				// Successfully sent item
			}
		}
	}()

	wg.Wait() // Wait until all consumer goroutines have completed their work
}

// RunTeeWithGenerator [T any] is a generator-based function that orchestrates the teeing process.
// It takes a ChannelTee instance, a generator function, and a process function for consumers.
// The generator function should send items to the provided channel and close it when done.
// This pattern provides better control over data flow and resource management.
//
// Parameters:
//   - ctx: Context for cancellation
//   - teeInstance: An instance of ChannelTee or custom implementation
//   - generator: Function that generates data and sends it to the provided channel
//   - processFn: Function executed for each output channel, receives consumer ID and output channel
func RunTeeWithGenerator[T any](ctx context.Context, teeInstance ChannelTee[T], generator func(ctx context.Context, ch chan<- T), processFn func(ctx context.Context, id int, ch <-chan T)) {
	// Create an input channel for the data
	inputCh := make(chan T)

	outputChans := teeInstance.GetOutputChannels()

	var wg sync.WaitGroup // WaitGroup for waiting for consumer goroutines to complete

	// Start the Tee in a separate goroutine to distribute the data
	go teeInstance.Run(ctx, inputCh)

	// Start goroutines to read from the output channels (consumers)
	for i, outCh := range outputChans {
		wg.Add(1) // Add a counter for each consumer goroutine
		go func(id int, ch <-chan T) {
			defer wg.Done()        // Decrement the WaitGroup counter when the goroutine finishes
			processFn(ctx, id, ch) // Call the provided processing function for this channel
		}(i+1, outCh) // Pass the consumer ID (starting from 1) and the channel
	}

	// Run the generator in a separate goroutine
	go generator(ctx, inputCh)

	wg.Wait() // Wait until all consumer goroutines have completed their work
}
