package workers

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

type Result struct {
	Value any
}

type TaskFunction func(...any) *Result

type Task struct {
	f         TaskFunction
	arguments []any
	res       chan Result
}

type Worker struct {
	Id       int
	Occupied bool
}

type Pool interface {
	Start()
	Stop()
	AddTask(t Task)
	AddTasks(t []Task)
	startWorkers()
	AllWorkersUnoccupied() bool
}

type Executor interface {
	Execute(t *Task)
}

type WorkerPool struct {
	Workers []Worker
	Tasks   chan Task
	quit    chan bool
	start   sync.Once
	stp     sync.Once
	timer   *time.Ticker
	wg      sync.WaitGroup
	sync.Mutex
}

func NewTask(function TaskFunction, args []any) Task {
	return Task{
		f:         function,
		arguments: args,
		res:       make(chan Result, 1),
	}
}

func NewWorkerPool(numWorkers int) *WorkerPool {

	wp := &WorkerPool{}

	for i := 0; i < numWorkers; i++ {
		w := Worker{
			Id:       i,
			Occupied: false,
		}
		wp.Workers = append(wp.Workers, w)
	}
	wp.Tasks = make(chan Task)
	wp.quit = make(chan bool)
	wp.timer = time.NewTicker(300 * time.Millisecond)

	return wp
}

func (p *WorkerPool) Add(t Task) {
	go p.AddTask(t)
}

func (p *WorkerPool) AddTask(t Task) {
	p.Tasks <- t
}

func (p *WorkerPool) AddTasks(ts []Task) {
	for _, t := range ts {
		p.Add(t)
	}
}

func (p *WorkerPool) Start() {

	log.Println("Starting workers")
	go p.start.Do(func() {
		p.startWorkers()
	})
}

func (p *WorkerPool) Stop() {
	p.stp.Do(func() {
		log.Println("Closing channels and stopping workers...")
		p.timer.Stop()
		close(p.quit)
		close(p.Tasks)
		p.Workers = []Worker{}
	})
}

func (p *WorkerPool) startWorkers() {

	for {
		select {
		case <-p.quit:
			return
		case <-p.timer.C:
			w := p.CheckAvailableWorker()
			if w == nil {
				break
			}
			select {
			case t := <-p.Tasks:
				go w.Execute(&t)
			}

		}

	}

}

func (p *WorkerPool) AllWorkersUnoccupied() bool {
	p.Lock()
	defer p.Unlock()
	for _, w := range p.Workers {
		if w.Occupied {
			return true
		}
	}
	return false
}

func (p *WorkerPool) CheckAvailableWorker() *Worker {
	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)
	availableWorkers := []Worker{}
	p.Lock()
	for _, w := range p.Workers {
		if !w.Occupied {
			availableWorkers = append(availableWorkers, w)
		}
	}
	p.Unlock()
	if len(availableWorkers) == 0 {
		return nil
	}
	w := availableWorkers[r.Intn(len(availableWorkers))]
	return &w
}

func (w *Worker) Execute(t *Task) {
	w.Occupied = true
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(task *Task) {
		defer wg.Done()
		res := task.f(t.arguments...)
		task.res <- *res
		return
	}(t)
	wg.Wait()
	w.Occupied = false
}

func (t *Task) GetResult() any {
	response := (<-t.res).Value
	close(t.res)
	return response
}

func (p *WorkerPool) WaitForResults(ts []Task) []any {

	results := []any{}
	currLen := len(ts)
	i := 0
	log.Println("Gathering results")
	for len(results) != currLen {
		t := ts[i]
		if len(t.res) == 1 {
			results = append(results, t.GetResult())
			if i == 0 {
				ts = ts[i:]
			} else {
				ts = append(ts[0:i], ts[i:]...)
			}
		}
		log.Printf("Pending %d result/s", currLen-len(results))
		if i+1 == currLen {
			i = 0
		} else {
			i++
		}

		time.Sleep(500 * time.Millisecond)
	}
	return results

}
