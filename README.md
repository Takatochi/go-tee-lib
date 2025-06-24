## go-tee-lib: Generic Tee Implementation for Go Channels
go-tee-lib is a Go module that provides a generic "Tee" pattern implementation for channels. It allows you to duplicate values from a single input channel to multiple output channels, enabling parallel processing or fan-out scenarios with ease.

## Why Use a Tee?
The Tee pattern is incredibly useful in concurrent programming when you need to send the same data to multiple independent consumers or processing pipelines. Instead of reading from the input channel multiple times (which would lead to race conditions or only one consumer receiving data), go-tee-lib ensures that each output channel receives an identical copy of the data.

## Features
- Generic: Works with any data type T.
- Buffered/Unbuffered Output Channels: Choose between unbuffered or buffered output channels based on your needs.
- Simple API: Easy to integrate into existing Go projects.
- Convenience Function: RunTeeAndProcess simplifies common use cases by handling channel setup, data feeding, and consumer coordination.
- Extensible: The ChannelTee interface allows for custom Tee implementations.

## Architectural Overview
The core idea of the Tee pattern is to take a single stream of data and fan it out to multiple parallel streams. Imagine a "T" junction where data flows in from the top and is split to go both left and right simultaneously.

Here's a console-based diagram illustrating the principle:

                  +-----------------+
                  |  Input Channel  |
                  +--------+--------+
                           |
                           | 
                           v
                  +-----------------+
                  |    The Tee      |
                  |    Mechanism    |
                  |  (Goroutine:    |
                  |  Splits & Copies)|
                  +-------+---------+
                          |
                  +-------+-------+
                  |               |
                  |  Duplication  |
                  |               |
                  +-----/--|---\----+
                       /   |    \
                      v    v     v
         +-------+  +-------+  +-------+
         | Output|  | Output|  | Output|
         | Chan 1|  | Chan 2|  | Chan 3|
         +-------+  +-------+  +-------+
             |          |          |
             v          v          v
         +-------+  +-------+    +-------+
         |Goroutine| |Goroutine| |Goroutine|
         |Consumer| |Consumer|   |Consumer|
         |   A   |  |   B   |    |   C   |
         +-------+  +-------+    +-------+

Principle: "Packet A" (and all subsequent packets)
is sent to Goroutine Consumer A, Goroutine Consumer B, AND Goroutine Consumer C.

- Input Channel: This is where your data (e.g., messages, tasks) enters the Tee.
- Tee Mechanism (The Box): Our Tee implementation acts as a central component. It reads each item from the input channel once, processes it (duplicates), and then sends identical copies of that item out. The internal "Splits & Copies" logic runs within a dedicated Goroutine managed by the Tee.
- Output Channels: For every item read, the Tee sends an identical copy to all its configured output channels.
- Consumers (Goroutines): Multiple independent goroutines or processes can read from these distinct output channels, each receiving the full stream of data without interference from others.
This ensures that all consumers receive the complete dataset, allowing for parallel and independent processing.
## Installation
To use go-tee-lib in your project, simply run go get:
```
go get github.com/Takatochi/go-tee-lib/tee
```
## Usage
Basic Usage with NewTee and RunTeeAndProcess
This is the recommended way for most common scenarios where you have a finite list of items to "tee" and a consistent processing logic for each consumer.

    package main

    import (
        "fmt"
        "time"
        "github.com/Takatochi/go-tee-lib/tee"
    )

    func main() {
    fmt.Println("--- Example using RunTeeAndProcess with NewTee ---")
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

	// Create a Tee instance (buffered in this case)
	myTee := tee.NewTee[int](numConsumers, bufferedSize)
	// Run the teeing process and wait for consumers to finish
	tee.RunTeeAndProcess(myTee, dataToSend, consumerProcessor)

	fmt.Println("--- RunTeeAndProcess example finished ---")
}

Advanced Usage with NewTee and Tee.Run (Manual Control)
For scenarios requiring more granular control over the input channel's lifecycle (e.g., streaming data, dynamic input), you can use Tee.Run directly.

    package main
    
    import (
    "fmt"
    "sync"
    "time"
    
    "github.com/Takatochi/go-tee-lib/tee" 
    )
    
    func main() {
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
}

## Contributing
We welcome contributions! If you find a bug or have a feature request, please open an issue. For code contributions, please fork the repository and submit a pull request.

## License
This project is licensed under the MIT License - see the LICENSE file for details.