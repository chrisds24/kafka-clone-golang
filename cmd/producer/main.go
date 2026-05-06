package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/chrisds24/kafka-clone-golang/clients/producer"
	"github.com/chrisds24/kafka-clone-golang/common"
)

// To run:
//   - While in In kafka-clone-golang directory
//     -- "go run ./cmd/producer ./data/producer_test_data/test1.txt"
func main() {
	// TODO: Take into account producer configs and metadata the producer
	//   knows about the broker/s when implementing a simple producer.

	// FOR NOW:
	// 1.) Open the file to read
	// 2.) Keep reading line by line, sending each one to simulate
	//     an event happening
	//     - At least, I'm trying to simulate a user writing inputs into
	//       the terminal in the Kafka Quickstart page, but I'm just reading
	//       from a file so I don't have to type every time + I'm able to
	//       send more info
	// 3.) Close the file (deferred)

	// Args hold the command-line arguments, starting with the program name
	// - Source: https://pkg.go.dev/os:
	path := os.Args[1]
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// ------ FOR NOW: Hardcoded configs --------
	// lingerMs: I'll just do 1 just to see how the batching works
	// - 0 = lowest latency, but worst throughput + batching
	// - 5-20 = better batching + throughput, latency increase
	// - RecordAccumulator uses int as the type for lingerMs
	// batchSize: batchSize is the number of records a batch could hold
	// - TODO: Switch to bytes later
	// bufferMemory: bufferMemory is the number of unsent records
	// - So this is basically 5 full batches of size 10 each
	producerConfig := common.ProducerConfig{
		LingerMs:     1,
		BatchSize:    5,
		BufferMemory: 50,
		Acks:         "1"}

	// TODO: Create the producer instance
	prod := producer.New(producerConfig)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Each line in the test file would be: topic|value\n
		// Ex. payments|Bob
		line := scanner.Text()
		// https://pkg.go.dev/strings#Split
		splitStr := strings.Split(line, "|")
		topic := splitStr[0]
		partition := splitStr[1]
		key := splitStr[2]
		val := splitStr[3]

		// For testing if file reading works
		// fmt.Printf(
		// 	"Event with value <\"%s\"> with key <%s> sent to partition <%s> of topic <%s>\n",
		// 	val,
		// 	key,
		// 	partition,
		// 	topic)
		intPartition, err := strconv.Atoi(partition)
		if err != nil {
			panic(err)
		}

		prod.Send(common.ProducerRecord{
			Topic:     topic,
			Partition: intPartition,
			Key:       key,
			Value:     val})
	}
	// Scanner.Err method will return any error that occurred during
	//   scanning, except that if it was io.EOF, Scanner.Err will
	//   return nil.
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

/*
To run the producer:
- In a terminal, cd into the "kafka-clone-golang" directory
- Then "go run ./cmd/producer <file path from root>", which runs the main
  function in producer.go.
- I'm basing this on how a console producer is ran in Kafka's Quick Start page:
  https://kafka.apache.org/42/getting-started/quickstart/#step-4-write-some-events-into-the-topic
  -- However, I'm just reading events from a file for now

Refer to these pages for info about a ConsoleProducer:
- https://github.com/apache/kafka/blob/trunk/bin/kafka-console-producer.sh
- https://github.com/apache/kafka/blob/trunk/tools/src/main/java/org/apache/kafka/tools/ConsoleProducer.java

ConsoleProducer (I WON'T BE FOLLOWING THESE EXACT STEPS)
https://github.com/apache/kafka/blob/trunk/tools/src/main/java/org/apache/kafka/tools/ConsoleProducer.java
- I'll use a simplified version of what the ConsoleProducer does in the real
  Kafka
- Steps (I'll skip some)
  -- Sets up ConsoleProducerOptions based on args from the command line
     command
	 + Basically, I'll need to do something similar where I can set up
	   config here
  -- Calls loopreader, passing in the KafkaProducer and RecordReader
  -- In loopreader
     + The RecordReader calls its readRecords(System.in) function
       * This reads from input in the terminal
	   * This is assigned to an iterator variable
	 + While iterator has a next value, call send, passing in the producer
	   (the KafkaProducer) and iter.next() (which is just a ProducerRecord)
  -- In send: producer.send(record).get()    where record is just the
     ProducerRecord
- Simplified/summarized:
  -- Get the options/config from command line args
  -- Keep sending records (aka events) using the KafkaProducer's send method
     as long as there are records to send
--------------------------------
***** IMPORTANT ******: So the KafkaProducer is gonna be my
  clients/producer/producer.go, which has the implementation code for how a
  producer works

--------- Sample data (ordered here, but won't be in test1.txt) ------------

analytics|0|session_a1|User landed on homepage
analytics|1|session_b2|User viewed product page for wireless keyboard
analytics|0|session_a1|User signed up from landing page A
analytics|1|session_b2|User added item to cart
analytics|0|session_c3|User started checkout flow
analytics|1|session_b2|User abandoned cart at payment step

orders|0|1001|Order created for 2 wireless keyboards
orders|0|1001|Order payment initiated
orders|0|1001|Order confirmed and sent to warehouse

orders|1|1002|Order created for 1 monitor arm
orders|1|1002|Order shipped to San Diego warehouse
orders|1|1002|Order delivered to customer

orders|2|1003|Order created for 3 USB-C hubs
orders|2|1003|Order payment failed
orders|2|1003|Order cancelled due to failed payment

orders|0|1004|Order created for mechanical keyboard
orders|0|1004|Order refunded after return request approved

payments|0|1001|Payment of $84.99 initiated
payments|0|1001|Payment of $84.99 completed
payments|0|1003|Payment of $129.50 failed
*/
