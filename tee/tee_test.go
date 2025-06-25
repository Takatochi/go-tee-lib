// Package tee provides tests for the generic Tee implementation for Go channels.
package tee

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewTeeUnbuffered ensures that NewTee creates the correct number of unbuffered channels.
func TestNewTeeUnbuffered(t *testing.T) {
	numChans := 5
	teeInstance := NewTee[int](numChans, 0) // 0 for unbuffered

	outputChans := teeInstance.GetOutputChannels()

	if len(outputChans) != numChans {
		t.Errorf("Expected %d output channels, got %d", numChans, len(outputChans))
	}

	for i, ch := range outputChans {
		select {
		case ch <- 1:
			t.Errorf("Channel %d should be unbuffered, but it accepted a value immediately", i)
		default:
			// Expected behavior for an unbuffered channel when nothing is reading
		}
	}
}

// TestNewTeeBuffered ensures that NewTee creates the correct number of buffered channels with the specified size.
func TestNewTeeBuffered(t *testing.T) {
	numChans := 3
	bufferSize := 10
	teeInstance := NewTee[string](numChans, bufferSize)

	outputChans := teeInstance.GetOutputChannels()

	if len(outputChans) != numChans {
		t.Errorf("Expected %d output channels, got %d", numChans, len(outputChans))
	}

	for i, ch := range outputChans {
		for j := 0; j < bufferSize; j++ {
			select {
			case ch <- fmt.Sprintf("test%d", j):
				// Expected: Should be able to write up to bufferSize without blocking
			default:
				t.Errorf("Channel %d should be buffered with size %d, but it blocked at %d writes", i, bufferSize, j)
			}
		}
		// Attempt to write one more than buffer size, it should block
		select {
		case ch <- "extra":
			t.Errorf("Channel %d should have blocked after %d writes, but it accepted an extra value", i, bufferSize)
		default:
			// Expected behavior: channel is full
		}
	}
}

// TestTeeRunDuplicatesData verifies that Tee.Run duplicates all data to all output channels.
func TestTeeRunDuplicatesData(t *testing.T) {
	numChans := 3
	dataCount := 10
	teeInstance := NewTee[int](numChans, 0) // Using unbuffered for strict verification
	inputCh := make(chan int)
	outputChans := teeInstance.GetOutputChannels()

	var wg sync.WaitGroup
	receivedCounts := make([]int, numChans)
	receivedData := make([][]int, numChans)

	for i := 0; i < numChans; i++ {
		receivedData[i] = make([]int, 0, dataCount)
		wg.Add(1)
		go func(consumerID int, ch <-chan int) {
			defer wg.Done()
			for val := range ch {
				receivedData[consumerID] = append(receivedData[consumerID], val)
				receivedCounts[consumerID]++
			}
		}(i, outputChans[i])
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2) // Timeout after 2 seconds
	defer cancel()
	go teeInstance.Run(ctx, inputCh)

	for i := 0; i < dataCount; i++ {
		inputCh <- i
	}
	close(inputCh) // Signal Tee to close output channels

	wg.Wait() // Wait for all consumers to finish

	// Verify all consumers received all data
	for i := 0; i < numChans; i++ {
		if receivedCounts[i] != dataCount {
			t.Errorf("Consumer %d: Expected to receive %d items, got %d", i, dataCount, receivedCounts[i])
		}
		for j := 0; j < dataCount; j++ {
			if receivedData[i][j] != j {
				t.Errorf("Consumer %d: Expected item at index %d to be %d, got %d", i, j, j, receivedData[i][j])
			}
		}
	}
}

// TestTeeRunClosesChannels ensures that output channels are closed after the input channel.
func TestTeeRunClosesChannels(t *testing.T) {
	numChans := 2
	teeInstance := NewTee[bool](numChans, 0)
	inputCh := make(chan bool)
	outputChans := teeInstance.GetOutputChannels()

	var wg sync.WaitGroup
	closeSignals := make(chan struct{}, numChans) // To signal when a consumer's channel is closed

	for i := 0; i < numChans; i++ {
		wg.Add(1)
		go func(ch <-chan bool) {
			defer wg.Done()
			for range ch {
				// Read any data sent
			}
			closeSignals <- struct{}{} // Signal that channel is closed
		}(outputChans[i])
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2) // Timeout after 2 seconds
	defer cancel()
	go teeInstance.Run(ctx, inputCh)

	inputCh <- true // Send some data
	close(inputCh)  // Close input channel

	// Wait for all output channels to be reported as closed
	for i := 0; i < numChans; i++ {
		select {
		case <-closeSignals:
			// Expected: channel was closed
		case <-time.After(time.Second):
			t.Fatalf("Timeout waiting for output channel %d to close", i)
		}
	}
	wg.Wait() // Ensure all goroutines have exited
}

// CustomTee is a custom implementation of the ChannelTee interface for testing purposes.
type CustomTee struct {
	outputChs []chan int
}

// GetOutputChannels returns the output channels for CustomTee.
func (ct *CustomTee) GetOutputChannels() []chan int {
	return ct.outputChs
}

// Run duplicates values from the input channel to all output channels for CustomTee.
func (ct *CustomTee) Run(ctx context.Context, inputChannel <-chan int) {
	go func() {
		defer func() {
			for _, ch := range ct.outputChs {
				close(ch)
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
				for _, ch := range ct.outputChs {
					select {
					case <-ctx.Done():
						return
					case ch <- val:
					}
				}
			}
		}
	}()
}

// TestRunTeeAndProcessFullCycle verifies the end-to-end functionality of RunTeeAndProcess.
func TestRunTeeAndProcessFullCycle(t *testing.T) {
	ctx := context.Background()
	dataToSend := []int{1, 2, 3, 4, 5}
	numConsumers := 2
	bufferedSize := 1 // Test with buffered channels
	receivedData := make([][]int, numConsumers)
	var mu sync.Mutex // Mutex to protect receivedData slices

	// Consumer processing function that records received data
	consumerProcessor := func(ctx context.Context, id int, ch <-chan int) {
		var localReceived []int
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				receivedData[id-1] = localReceived // id is 1-based, array is 0-based
				mu.Unlock()
				return
			case val, ok := <-ch:
				if !ok {
					mu.Lock()
					receivedData[id-1] = localReceived // id is 1-based, array is 0-based
					mu.Unlock()
					return
				}
				localReceived = append(localReceived, val)
			}
		}
	}

	// Create a Tee instance to pass to RunTeeAndProcess
	myTee := NewTee[int](numConsumers, bufferedSize)

	// Run the full process
	RunTeeAndProcess(ctx, myTee, dataToSend, consumerProcessor)

	// Verify that each consumer received all data
	for i := 0; i < numConsumers; i++ {
		if len(receivedData[i]) != len(dataToSend) {
			t.Errorf("Consumer %d: Expected %d items, got %d", i+1, len(dataToSend), len(receivedData[i]))
			continue
		}
		for j, val := range dataToSend {
			if receivedData[i][j] != val {
				t.Errorf("Consumer %d: Expected item %d to be %d, got %d", i+1, j, val, receivedData[i][j])
			}
		}
	}
}

// TestRunTeeAndProcessNoBuffer verifies RunTeeAndProcess with unbuffered channels.
func TestRunTeeAndProcessNoBuffer(t *testing.T) {
	ctx := context.Background()
	dataToSend := []string{"apple", "banana", "cherry"}
	numConsumers := 2
	bufferedSize := 0 // Unbuffered channels
	receivedData := make([][]string, numConsumers)
	var mu sync.Mutex

	consumerProcessor := func(ctx context.Context, id int, ch <-chan string) {
		var localReceived []string
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				receivedData[id-1] = localReceived
				mu.Unlock()
				return
			case val, ok := <-ch:
				if !ok {
					mu.Lock()
					receivedData[id-1] = localReceived
					mu.Unlock()
					return
				}
				localReceived = append(localReceived, val)
			}
		}
	}

	myTee := NewTee[string](numConsumers, bufferedSize)

	// Run the full process
	RunTeeAndProcess(ctx, myTee, dataToSend, consumerProcessor)

	for i := 0; i < numConsumers; i++ {
		if len(receivedData[i]) != len(dataToSend) {
			t.Errorf("Consumer %d: Expected %d items, got %d (unbuffered test)", i+1, len(dataToSend), len(receivedData[i]))
			continue
		}
		for j, val := range dataToSend {
			if receivedData[i][j] != val {
				t.Errorf("Consumer %d: Expected item %d to be '%s', got '%s' (unbuffered test)", i+1, j, val, receivedData[i][j])
			}
		}
	}
}

// TestCustomChannelTeeImplementation ensures RunTeeAndProcess works with custom ChannelTee implementations.
func TestCustomChannelTeeImplementation(t *testing.T) {
	// Create a custom Tee implementation
	numChans := 2
	customOutputChans := make([]chan int, numChans)
	for i := 0; i < numChans; i++ {
		customOutputChans[i] = make(chan int, 1) // Buffered channels for custom tee
	}
	customTeeInstance := &CustomTee{outputChs: customOutputChans}

	dataToSend := []int{100, 200, 300}
	receivedData := make([][]int, numChans)
	var mu sync.Mutex

	consumerProcessor := func(ctx context.Context, id int, ch <-chan int) {
		var localReceived []int
		for {
			select {
			case val, ok := <-ch:
				if !ok {
					mu.Lock()
					receivedData[id-1] = localReceived
					mu.Unlock()
					return
				}
				localReceived = append(localReceived, val)
			case <-ctx.Done():
				mu.Lock()
				receivedData[id-1] = localReceived
				mu.Unlock()
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2) // Timeout after 2 seconds
	defer cancel()
	// Run the full process
	// Use RunTeeAndProcess with the custom Tee
	RunTeeAndProcess(ctx, customTeeInstance, dataToSend, consumerProcessor)

	for i := 0; i < numChans; i++ {
		if len(receivedData[i]) != len(dataToSend) {
			t.Errorf("Custom Tee Consumer %d: Expected %d items, got %d", i+1, len(dataToSend), len(receivedData[i]))
			continue
		}
		for j, val := range dataToSend {
			if receivedData[i][j] != val {
				t.Errorf("Custom Tee Consumer %d: Expected item %d to be %d, got %d", i+1, j, val, receivedData[i][j])
			}
		}
	}
}

// TestRunTeeWithGenerator verifies the generator-based teeing functionality.
func TestRunTeeWithGenerator(t *testing.T) {
	ctx := context.Background()
	numConsumers := 2
	bufferedSize := 1
	receivedData := make([][]int, numConsumers)
	var mu sync.Mutex

	// Generator function that produces data
	generator := func(ctx context.Context, ch chan<- int) {
		defer close(ch) // Important: close the channel when done
		for i := 1; i <= 5; i++ {
			select {
			case <-ctx.Done():
				return // Exit early if context is canceled
			case ch <- i:
				// Successfully sent item
			}
		}
	}

	// Consumer processing function
	consumerProcessor := func(ctx context.Context, id int, ch <-chan int) {
		var localReceived []int
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				receivedData[id-1] = localReceived
				mu.Unlock()
				return
			case val, ok := <-ch:
				if !ok {
					mu.Lock()
					receivedData[id-1] = localReceived
					mu.Unlock()
					return
				}
				localReceived = append(localReceived, val)
			}
		}
	}

	myTee := NewTee[int](numConsumers, bufferedSize)

	// Run the generator-based process
	RunTeeWithGenerator(ctx, myTee, generator, consumerProcessor)

	// Verify that all consumers received all data
	expectedData := []int{1, 2, 3, 4, 5}
	for i := 0; i < numConsumers; i++ {
		if len(receivedData[i]) != len(expectedData) {
			t.Errorf("Consumer %d: Expected %d items, got %d", i+1, len(expectedData), len(receivedData[i]))
			continue
		}
		for j, val := range expectedData {
			if receivedData[i][j] != val {
				t.Errorf("Consumer %d: Expected item %d to be %d, got %d", i+1, j, val, receivedData[i][j])
			}
		}
	}
}

// TestRunTeeWithGeneratorCancellation verifies proper context cancellation handling.
func TestRunTeeWithGeneratorCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	numConsumers := 2
	bufferedSize := 0

	// Generator that would run indefinitely without cancellation
	generator := func(ctx context.Context, ch chan<- int) {
		defer close(ch)
		for i := 1; ; i++ {
			select {
			case <-ctx.Done():
				return // Exit when context is canceled
			case ch <- i:
				// Successfully sent item
			}
		}
	}

	// Consumer that counts received items
	itemCount := int32(0)
	consumerProcessor := func(ctx context.Context, _ int, ch <-chan int) {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
				atomic.AddInt32(&itemCount, 1)
			}
		}
	}

	myTee := NewTee[int](numConsumers, bufferedSize)

	// Start the process in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunTeeWithGenerator(ctx, myTee, generator, consumerProcessor)
	}()

	// Let it run for a short time, then cancel
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for completion with timeout
	select {
	case <-done:
		// Good, it completed
	case <-time.After(1 * time.Second):
		t.Error("RunTeeWithGenerator did not complete within timeout after cancellation")
	}

	// Verify that some items were processed (but not too many due to cancellation)
	count := atomic.LoadInt32(&itemCount)
	if count == 0 {
		t.Error("Expected some items to be processed before cancellation")
	}
	t.Logf("Processed %d items before cancellation", count)
}
