// Package tee надає загальну реалізацію патерну Tee для Go каналів.
// Дозволяє дублювати значення з одного вхідного каналу до кількох вихідних каналів.
//
// Package tee provides a generic Tee implementation for Go channels.
// It allows duplicating values from a single input channel to multiple output channels.
//
// Патерн Tee корисний для:
// - Паралельної обробки одних і тих же даних різними споживачами
// - Розподілу потоку даних на кілька незалежних обробників
// - Створення копій даних для логування, моніторингу та основної обробки
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

// ChannelTee - інтерфейс для реалізації патерну Tee з каналами.
// ChannelTee interface for implementing the Tee pattern with channels.
//
// Приклад використання / Usage example:
//
//	// Створення Tee з 3 споживачами та буферизованими каналами
//	// Creating Tee with 3 consumers and buffered channels
//	teeInstance := tee.NewTee[int](3, 2)
//
//	// Функція обробки для кожного споживача
//	// Processing function for each consumer
//	processor := func(id int, ch <-chan int) {
//		fmt.Printf("Споживач #%d: Початок обробки\n", id)
//		for value := range ch {
//			fmt.Printf("Споживач #%d отримав: %d\n", id, value)
//		}
//	}
//
//	// Запуск обробки з даними
//	// Start processing with data
//	data := []int{10, 20, 30, 40, 50}
//	ctx := context.Background()
//	tee.RunTeeAndProcess(ctx, teeInstance, data, processor)
type ChannelTee[T any] interface {
	// GetOutputChannels повертає слайс вихідних каналів з Tee.
	// GetOutputChannels returns the slice of output channels from the Tee.
	GetOutputChannels() []chan T

	// Run запускає процес розподілу даних. Читає з inputChannel
	// та дублює повідомлення до всіх вихідних каналів.
	// Ця функція зазвичай повинна запускатися в горутині.
	// Блокується до закриття inputChannel та обробки всіх значень.
	// Після закриття inputChannel всі вихідні канали також будуть закриті.
	//
	// Run starts the teeing process. It reads from the inputChannel
	// and duplicates the messages to all output channels.
	// This function should typically be run in a goroutine.
	// It will block until the inputChannel is closed and all values are processed.
	// Once the inputChannel is closed, all output channels will also be closed.
	Run(ctx context.Context, inputChannel <-chan T)
}

// Tee [T any] структура, що містить вихідні канали.
// WaitGroup для координації горутин споживачів керується викликаючим кодом,
// а не самим Tee, відповідно до ідіоматичних патернів каналів Go.
//
// Tee [T any] struct holds the output channels.
// The WaitGroup for coordinating consumer goroutines is managed by the caller,
// not by the Tee itself, as per Go's idiomatic channel patterns.
type Tee[T any] struct {
	outputChannels []chan T
}

// NewTee створює новий екземпляр Tee з вказаною кількістю вихідних каналів.
// Якщо bufferSize дорівнює 0, створюються небуферизовані канали.
// Інакше створюються буферизовані канали з заданим bufferSize.
// Повертає інтерфейс ChannelTee, конкретно *Tee[T], який реалізує інтерфейс.
//
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

// GetOutputChannels повертає слайс вихідних каналів з Tee.
// GetOutputChannels returns the slice of output channels from the Tee.
func (t *Tee[T]) GetOutputChannels() []chan T {
	return t.outputChannels
}

// Run запускає процес розподілу даних. Читає з inputChannel
// та дублює повідомлення до всіх вихідних каналів.
// Ця функція зазвичай повинна запускатися в горутині.
// Блокується до закриття inputChannel та обробки всіх значень.
// Після закриття inputChannel всі вихідні канали також будуть закриті.
//
// Run starts the teeing process. It reads from the inputChannel
// and duplicates the messages to all output channels.
// This function should typically be run in a goroutine.
// It will block until the inputChannel is closed and all values are processed.
// Once the inputChannel is closed, all output channels will also be closed.
//
// Це правильна реалізація патерну Tee:
// Одна горутина читає з вхідного каналу та розподіляє
// кожне отримане значення до всіх своїх вихідних каналів.
//
// This is the correct implementation of the Tee pattern:
// A single goroutine reads from the input channel and then distributes
// each received value to all its output channels.
func (t *Tee[T]) Run(ctx context.Context, inputChannel <-chan T) {
	// Основна горутина, що читає з вхідного каналу та розподіляє дані
	// The main goroutine that reads from the input channel and distributes data
	go func() {
		defer func() {
			// Після закриття вхідного каналу та читання всіх значень,
			// ми повинні закрити всі вихідні канали. Це гарантує, що горутини споживачів,
			// які читають з цих каналів, вийдуть зі своїх циклів `for range`.
			//
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
				// Для кожного значення з вхідного каналу відправляємо його до кожного вихідного каналу.
				// For each value from the input channel, send it to every output channel.
				for _, outCh := range t.outputChannels {
					select {
					case <-ctx.Done():
						return
					case outCh <- val:
						// Успішно відправлено / Successfully sent
					}
				}
			}
		}
	}()
}

// RunTeeAndProcess [T any] - зручна функція, що організовує весь процес розподілу даних.
// Приймає екземпляр ChannelTee, слайс елементів для відправки та функцію обробки для споживачів.
// Відправляє елементи через наданий екземпляр tee та запускає горутини споживачів
// для кожного вихідного каналу, використовуючи надану функцію обробки.
// Блокується до обробки всіх елементів усіма споживачами.
//
// RunTeeAndProcess [T any] is a convenience function that orchestrates the entire teeing process.
// It takes a ChannelTee instance, a slice of items to send, and a process function for consumers.
// It sends the items through the provided tee instance and then spawns consumer goroutines
// for each output channel using the provided process function.
// It blocks until all items are processed by all consumers.
//
// Параметри / Parameters:
//   - ctx: Контекст для скасування операції / Context for cancellation
//   - teeInstance: Екземпляр ChannelTee (наприклад, створений NewTee),
//     або власна реалізація ChannelTee / An instance of ChannelTee or custom implementation
//   - items: Слайс даних для відправки через tee / The slice of data to be sent through the tee
//   - processFn: Функція, що виконується для кожного вихідного каналу.
//     Отримує ID споживача та вихідний канал як аргументи /
//     Function executed for each output channel, receives consumer ID and output channel
func RunTeeAndProcess[T any](ctx context.Context, teeInstance ChannelTee[T], items []T, processFn func(id int, ch <-chan T)) {
	// Створюємо вхідний канал для даних / Create an input channel for the data
	inputCh := make(chan T)

	outputChans := teeInstance.GetOutputChannels()

	var wg sync.WaitGroup // WaitGroup для очікування завершення горутин споживачів / WaitGroup for waiting for consumer goroutines to complete

	// Запускаємо Tee в окремій горутині для розподілу даних / Start the Tee in a separate goroutine to distribute the data
	go teeInstance.Run(ctx, inputCh)

	// Запускаємо горутини для читання з вихідних каналів (споживачі) / Start goroutines to read from the output channels (consumers)
	for i, outCh := range outputChans {
		wg.Add(1) // Додаємо лічильник для кожної горутини споживача / Add a counter for each consumer goroutine
		go func(id int, ch <-chan T) {
			defer wg.Done()   // Зменшуємо лічильник WaitGroup при завершенні горутини / Decrement the WaitGroup counter when the goroutine finishes
			processFn(id, ch) // Викликаємо надану функцію обробки для цього каналу / Call the provided processing function for this channel
		}(i+1, outCh) // Передаємо ID споживача (починаючи з 1) та канал / Pass the consumer ID (starting from 1) and the channel
	}

	// Відправляємо всі елементи до вхідного каналу в окремій горутині / Send all items to the input channel in a separate goroutine
	go func() {
		for _, item := range items {
			inputCh <- item
		}
		close(inputCh) // Важливо: закриваємо вхідний канал після відправки всіх даних / Important: close the input channel after sending all data
	}()

	wg.Wait() // Очікуємо завершення всіх горутин споживачів / Wait until all consumer goroutines have completed their work
}
