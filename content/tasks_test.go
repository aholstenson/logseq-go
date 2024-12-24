package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Task status", func() {
	DescribeTable("can be parsed from strings",
		func(input string, expectedStatus content.TaskStatus) {
			taskMarker := content.NewTaskMarkerFromString(input)
			Expect(taskMarker.Status).To(Equal(expectedStatus))
		},
		Entry("TODO", "todo", content.TaskStatusTodo),
		Entry("DONE", "done", content.TaskStatusDone),
		Entry("DOING", "doing", content.TaskStatusDoing),
		Entry("LATER", "LATER", content.TaskStatusLater),
		Entry("NOW", "NOW", content.TaskStatusNow),
		Entry("CANCELLED", "CANCELLED", content.TaskStatusCancelled),
		Entry("CANCELED", "CANCELED", content.TaskStatusCanceled),
		Entry("IN-PROGRESS", "IN-PROGRESS", content.TaskStatusInProgress),
		Entry("WAIT", "WAIT", content.TaskStatusWait),
		Entry("WAITING", "WAITING", content.TaskStatusWaiting),
	)

	DescribeTable("returns an error for invalid input",
		func(invalidInput string) {
			taskMarker := content.NewTaskMarkerFromString(invalidInput)
			Expect(taskMarker.Status).To(Equal(content.TaskStatusNone))
		},
		Entry("Invalid input", "Invalid"),
		Entry("Empty input", ""))
})
