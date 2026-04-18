package main // In Go, package name must be main for executables

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/chrisds24/kafka-clone-golang/internal/broker"
)

func main() {
	// TODO: Later on, I can load config here which I could pass to Start

	// First, remove any existing temporary topic and partition directories
	// - REASON: defer doesn't run when hard killing a process
	// - If path doesn't exist or path is successfully deleted, returns nil
	// - This returns an error on the first error it encounters
	// IMPORTANT: Where is the passed in path relative to?
	// - os.RemoveAll("./data/tmp") is relative to the process’s current
	//   working directory (CWD) at runtime — not automatically your project
	//   root, and not based on the file location where the code lives
	// - If "go run ./cmd/broker" while in "myprojectroot/", then
	//   os.RemoveAll("./data/tmp") targets "myprojectroot/data/tmp"
	// - If "go run ." while in "myprojectroot/cmd/broker", then it targets
	//   "myprojectroot/cmd/broker/data/tmp"
	// IMPORTANT:
	// - Run the broker from the root directory aka "go run ./cmd/broker"
	//   while in the root dir instead of cd into ./cmd/broker before running
	//   "go run ."
	//   + Root dir is the "kafka-clone-golang" directory
	// - This ensures that os.RemoveAll("./data/tmp") starts relative to the
	//   root dir
	if err := os.RemoveAll("./data/tmp"); err != nil {
		panic(err)
	}

	// Need to create the temporary directories that will be used for
	//   testing
	// - TODO: Move this to the test files itself later?
	for i := 0; i < 3; i++ {
		createTmpPartitionLog("orders", strconv.Itoa(i))
	}
	createTmpPartitionLog("payments", "0")
	createTmpPartitionLog("analytics", "0")
	createTmpPartitionLog("analytics", "1")

	// Remove the created temporary topic and partition directories
	defer os.RemoveAll("./data/tmp")

	// gin.Default().Run() inside broker.Start() is blocking, so the deferred
	//   os.RemoveAll above won't run until the server stops or exits due to
	//   an error
	broker.Start()
}

// Helper function to easily create a partition for the given topic
//   - Side note: In Go, if a function doesn't return anything, just omit the
//     return type
func createTmpPartitionLog(topic string, partition string) {
	tmpDataPath := "./data/tmp"
	topicPartitionPath := filepath.Join(tmpDataPath, topic, partition)
	// Use the code above instead of:
	// topicPartitionPath := fmt.Sprintf(
	// 	"%s/%s/%s",
	// 	tmpDataPath,
	// 	topic,
	// 	partition)

	// Create the topic and partition directories
	//
	// os.Mkdirall: https://pkg.go.dev/os#MkdirAll
	// - Only returns one value (err)
	// - If directory exists, returns nil so err == nil and we can
	//   just proceed without any errors
	// - If directory is successfully created, also just returns nil
	if err := os.MkdirAll(topicPartitionPath, 0755); err != nil {
		panic(err)
	}

	// Create the partition.log for this partition
	// - If partition.log doesn't exist, create it. Otherwise, truncate
	f, err := os.Create(topicPartitionPath + "/partition.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()
}

/*
To run the broker:
- In a terminal, cd into the "kafka-clone-golang" directory
- Then "go run ./cmd/broker", which runs the main function in main.go
*/
