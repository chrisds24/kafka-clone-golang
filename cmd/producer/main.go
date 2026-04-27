package main

import (
	"bufio"
	"os"
	"strings"
)

func main() {
	// TODO: Take into account producer configs and metadata the producer
	//   knows about the broker/s when implementing a simple producer.

	// FOR NOW:
	// 1.) Open the file to read
	// 2.) Keep reading line by line, sending each one to simulate
	//     an event happening
	// 3.) Close the file (deferred)

	// Args hold the command-line arguments, starting with the program name
	// - Source: https://pkg.go.dev/os:
	path := os.Args[1]
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Each line in the test file would be: topic|value\n
		// Ex. payments|Bob
		line := scanner.Text()
		// https://pkg.go.dev/strings#Split
		splitStr := strings.Split(line, "|")
		topic := splitStr[0]
		eventVal := splitStr[1]
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

ConsoleProducer
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
- So the KafkaProducer is gonna be my clients/producer/producer.go, which
  has the implementation code for how a producer works
*/
