package broker

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	// io is another useful file system related package
	"strconv"

	"github.com/chrisds24/kafka-clone-golang/internal/types"
	"github.com/gin-gonic/gin"
)

// Called when the producer sends a request to add batches of messages
func appendMessages(c *gin.Context) {
	var producerReq types.ProducerRequest

	// Bind the received JSON to producerReq
	// - From: https://go.dev/doc/tutorial/web-service-gin#write-the-code-2
	if err := c.BindJSON(&producerReq); err != nil {
		panic(err)
	}

	/*
		Loop through the batches. Then for each batch, loop through the
		  events to add them to the specified partition.

		Use the os, io, bufio, and path/filepath packages to interact with the
		  file system
		- https://gobyexample.com/reading-files
		- https://gobyexample.com/writing-files
	*/
	for _, batch := range producerReq.Batches {
		// strconv is best to convert to string
		// - https://www.reddit.com/r/golang/comments/hkcgms/which_is_the_best_way_to_convert_int_to_string_in/
		// Path is: data/<topic>/<partition number>/partition.log
		path := filepath.Join(
			"data",
			batch.Topic,
			strconv.Itoa(batch.Partition),
			"partition.log")

		// https://pkg.go.dev/os#OpenFile
		// - os.O_APPEND: always write at end of file
		// - os.O_RDWR: read and write
		f, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR, 0644)
		if err != nil {
			panic(err)
		}
		// Defer to close file: https://go.dev/blog/defer-panic-and-recover
		// - defer after the nil check is the correct order, we don't want to
		//   call close on a nil
		defer f.Close()

		// Need to get number of lines to get nextOffset
		// - TODO: Later, this should happen on boot up then nextOffset is
		//   tracked/updated in memory until flushed
		// - bufio scanner: https://pkg.go.dev/bufio#NewScanner
		scanner := bufio.NewScanner(f)
		baseOffset := 0 // Renamed to this from nextOffset
		// Scan returns false when there are no more tokens
		// - for is Go's "while": https://go.dev/tour/flowcontrol/3
		// - The code below is basically while (scanner.Scan())
		// nextOffset is the next available offset. If the file is empty,
		//   nextOffset remains 0. But if not, then the very last line is
		//   an empty line (since when I add, I simply add "key|value\n").
		//   Scanner doesn't count the trailing new line as an empty line.
		//   For example, if our lines are:
		//   1.) 123|hello 2.) 456|world 3.) <trailing new line>
		//   - Scanner would read 2 lines and nextOffset would be 2.
		//     + nextOffset = 0, read 123|hello -> nextOffset++ (becomes 1)
		//     + nextOffset = 1, read 456|world -> nextOffset++ (becomes 2)
		//     + nextOffset = 2, trailing newline -> Don't run loop body
		//   There's no need for a +1 since nextOffset points to the next
		//   empty spot.
		// IMPORTANT: I don't really need to do this to figure out where to
		//   append the next event since I'm already using os.O_APPEND.
		//   However, according to ChatGPT, returning the baseOffset is
		//   useful metadata.
		//   {
		// 	     "topic": "orders",
		// 	     "partition": 1,
		// 	     "baseOffset": 3,
		// 	     "appendedCount": 2
		//   }
		for scanner.Scan() {
			baseOffset++
		}
		// Scanner.Err method will return any error that occurred during
		//   scanning, except that if it was io.EOF, Scanner.Err will
		//   return nil.
		if err := scanner.Err(); err != nil {
			panic(err)
		}

		w := bufio.NewWriter(f)
		for _, event := range batch.Events {
			// I'll just add each event (only has a string value for now) as
			//   one string per line
			// - Format is    key|value
			// Using Fprintf is better than:
			//   w.WriteString(fmt.Sprintf("%s|%s\n", event.Key, event.Value))
			// _, err := fmt.Fprintf(w, "%s|%s\n", event.Key, event.Value)
			_, err := fmt.Fprintf(w, "%s\n", event.Value) // TODO: Add key later
			if err != nil {
				panic(err)
			}
		}

		w.Flush()
	}

	// TODO: Edit this to send the appropriate metadata to the producer
	c.IndentedJSON(http.StatusOK, "Messages saved!")
}

/*
	Multiple partition batches would look like this
	{
		"acks": "leader",
		"timeoutMs": 5000,
		"batches": [
			{
				"topic": "orders",
				"partition": 0,
				"messages": [
					{ "key": "123", "value": "hello" },
					{ "key": "123", "value": "world" }
				]
			},
			{
				"topic": "orders",
				"partition": 1,
				"messages": [
					{ "key": "456", "value": "foo" },
					{ "key": "456", "value": "bar" }
				]
			},
			{
				"topic": "payments",
				"partition": 0,
				"messages": [
					{ "key": "143", "value": "HELLO" },
					{ "key": "426", "value": "WORLD" }
				]
			}
		]
	}

	IMPORTANT:
	- If using key-based partitioning, messages in the same topic with the same key
	would go to the same partition
	- Different entries in the same batches array could have been produced
	using different partitioning strategies


	WHAT IF:
	"batches": [
		{
		"topic": "orders",
		"partition": 0,
		"messages": [
			{ "key": "1", "value": "A" }
		]
		},
		{
		"topic": "orders",
		"partition": 0,
		"messages": [
			{ "key": "2", "value": "B" }
		]
		}
	]
	- Then request order matters for that same partition.
	- The Kafka guarantee is that order of events per partition is preserved.
	- However, I should still preserve request order for batches and preserve event
	order for each batch since the situation shown above could happen
*/
