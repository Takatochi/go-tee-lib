package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Takatochi/go-tee-lib/tee"
)

func main() {
	fmt.Println("Generator Pattern Example with go-tee-lib")
	fmt.Println("==========================================")

	// Example 1: Simple generator
	fmt.Println("\n--- Example 1: Simple Number Generator ---")
	simpleGeneratorExample()

	// Example 2: Generator with cancellation
	fmt.Println("\n--- Example 2: Generator with Context Cancellation ---")
	cancellationExample()

	// Example 3: Real-time data generator
	fmt.Println("\n--- Example 3: Real-time Data Stream Generator ---")
	realTimeExample()

	fmt.Println("\nAll generator examples completed!")
}

func simpleGeneratorExample() {
	ctx := context.Background()
	numConsumers := 2
	bufferedSize := 1

	// Generator function that produces numbers
	generator := func(ctx context.Context, ch chan<- int) {
		defer close(ch) // Always close the channel when done
		fmt.Println("Generator: Starting to produce numbers...")
		
		for i := 1; i <= 5; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("Generator: Context cancelled, stopping...")
				return
			case ch <- i:
				fmt.Printf("Generator: Produced %d\n", i)
				time.Sleep(100 * time.Millisecond) // Simulate work
			}
		}
		fmt.Println("Generator: Finished producing numbers")
	}

	// Consumer function
	processor := func(ctx context.Context, id int, ch <-chan int) {
		fmt.Printf("Consumer #%d: Starting...\n", id)
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("Consumer #%d: Context cancelled\n", id)
				return
			case val, ok := <-ch:
				if !ok {
					fmt.Printf("Consumer #%d: Channel closed, exiting\n", id)
					return
				}
				fmt.Printf("Consumer #%d: Processing %d\n", id, val)
				time.Sleep(50 * time.Millisecond) // Simulate processing
			}
		}
	}

	// Create Tee and run with generator
	teeInstance := tee.NewTee[int](numConsumers, bufferedSize)
	tee.RunTeeWithGenerator(ctx, teeInstance, generator, processor)
}

func cancellationExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	numConsumers := 2
	bufferedSize := 0

	// Generator that would run indefinitely without cancellation
	generator := func(ctx context.Context, ch chan<- int) {
		defer close(ch)
		fmt.Println("Infinite Generator: Starting...")
		
		for i := 1; ; i++ {
			select {
			case <-ctx.Done():
				fmt.Printf("Infinite Generator: Cancelled after producing %d items\n", i-1)
				return
			case ch <- i:
				fmt.Printf("Infinite Generator: Produced %d\n", i)
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	// Consumer that counts items
	processor := func(ctx context.Context, id int, ch <-chan int) {
		count := 0
		fmt.Printf("Counter #%d: Starting...\n", id)
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("Counter #%d: Cancelled after processing %d items\n", id, count)
				return
			case val, ok := <-ch:
				if !ok {
					fmt.Printf("Counter #%d: Channel closed after processing %d items\n", id, count)
					return
				}
				count++
				fmt.Printf("Counter #%d: Processed item %d (total: %d)\n", id, val, count)
			}
		}
	}

	teeInstance := tee.NewTee[int](numConsumers, bufferedSize)
	tee.RunTeeWithGenerator(ctx, teeInstance, generator, processor)
}

func realTimeExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	numConsumers := 3
	bufferedSize := 2

	// Simulate real-time data (like sensor readings)
	type SensorData struct {
		Timestamp time.Time
		Value     float64
		SensorID  string
	}

	// Real-time data generator
	generator := func(ctx context.Context, ch chan<- SensorData) {
		defer close(ch)
		fmt.Println("Sensor: Starting data collection...")
		
		sensorID := "TEMP_001"
		baseValue := 20.0
		
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				fmt.Printf("Sensor: Stopped after %d readings\n", i)
				return
			default:
				// Simulate sensor reading
				data := SensorData{
					Timestamp: time.Now(),
					Value:     baseValue + float64(i%10), // Simulate varying temperature
					SensorID:  sensorID,
				}
				
				select {
				case <-ctx.Done():
					return
				case ch <- data:
					fmt.Printf("Sensor: Reading %d - %.1f°C\n", i+1, data.Value)
					time.Sleep(80 * time.Millisecond) // Simulate sensor delay
				}
			}
		}
	}

	// Different types of consumers for real-time data
	processor := func(ctx context.Context, id int, ch <-chan SensorData) {
		var role string
		switch id {
		case 1:
			role = "Logger"
		case 2:
			role = "AlertSystem"
		case 3:
			role = "Dashboard"
		}
		
		fmt.Printf("%s: Starting...\n", role)
		count := 0
		
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("%s: Stopped after processing %d readings\n", role, count)
				return
			case data, ok := <-ch:
				if !ok {
					fmt.Printf("%s: Data stream ended after processing %d readings\n", role, count)
					return
				}
				count++
				
				switch role {
				case "Logger":
					fmt.Printf("Logger: Saved reading %.1f°C from %s\n", data.Value, data.SensorID)
				case "AlertSystem":
					if data.Value > 25.0 {
						fmt.Printf("AlertSystem: HIGH TEMP ALERT! %.1f°C\n", data.Value)
					}
				case "Dashboard":
					fmt.Printf("Dashboard: Updated display with %.1f°C\n", data.Value)
				}
			}
		}
	}

	teeInstance := tee.NewTee[SensorData](numConsumers, bufferedSize)
	tee.RunTeeWithGenerator(ctx, teeInstance, generator, processor)
}
