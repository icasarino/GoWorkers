package main

import (
	"fmt"
	"test/workers"
)

func main() {

	wp := workers.NewWorkerPool(5)
	tasks := []workers.Task{}
	for i := 0; i < 10; i++ {
		tasks = append(tasks, workers.NewTask(AddNum, []any{i, 3, -1}))
	}
	wp.AddTasks(tasks)
	wp.Start()

	results := wp.WaitForResults(tasks)
	fmt.Println(results)

	wp.Stop()

}

func AddNum(nums ...any) *workers.Result {
	res := 0
	for _, n := range nums {
		res += n.(int)
	}
	return &workers.Result{
		Value: res,
	}

}
