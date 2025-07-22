package orch

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type JobProcessorInterface interface {
	AddJob(job *Job) error
}

type JobProcessor struct {
	log                *slog.Logger
	parallelSteps      int
	jobsByType         sync.Map
	processor          *jobTypeProcessor
	jobTimeOutSec      int
	totalParallelSteps atomic.Int32
	debug              bool
	chanJobs           chan *Job
}

func NewJobProcessor(maxJobs int, jobTimeoutSec int, logger *slog.Logger) *JobProcessor {
	return &JobProcessor{
		parallelSteps: maxJobs,
		jobTimeOutSec: jobTimeoutSec,
		chanJobs:      make(chan *Job, maxJobs),
		log:           logger,
	}
}

func (jp *JobProcessor) AddJob(job *Job) error {
	jp.chanJobs <- job
	return nil
}

func (jp *JobProcessor) PrintStatus() {
	jp.log.Info("Job Processor Status", "steps", jp.parallelSteps, "total_steps", jp.totalParallelSteps.Load())

}

func (jp *JobProcessor) HandleSteps(ctx context.Context, concurrency int) error {
	if concurrency == 0 {
		concurrency = 1
	}

	// Read jobs from channel and process them
	for job := range jp.chanJobs {
		jp.evaluateJob(ctx, job, concurrency)
	}

	return nil
}

func (jp *JobProcessor) evaluateJob(ctx context.Context, job *Job, concurrency int) {

	var releaseStep bool
	defer func() {
		if releaseStep {

		}
	}()

	jp.totalParallelSteps.Add(1)

	if jp.processor == nil {

		jp.processor = newStepTypeProcessor(ctx, WithMaxSteps(jp.parallelSteps*concurrency), WithWaitTerminate(func() error {
			jp.log.Info("job processor terminated")
			return nil
		}), WithDebug(func(spc *jobTypeProcessor) {
			if jp.debug {
				jp.log.Info("job processor status job_type: %s, total jobs: %d", job, spc.totalSteps.Load())
			}
		}))
	}

	if isFull := jp.processor.enqueue(jobProcess{
		executor: func() error {

			start := time.Now()
			jp.log.Debug("job start: %s", job.ID)

			ctxTimeout, cancel := context.WithTimeout(context.TODO(), time.Duration(jp.jobTimeOutSec)*time.Second)
			defer cancel()
			errCtx := jp.processStep(ctxTimeout, job)
			if errCtx != nil {
				jp.log.Info("error processing job type: %s %v", job.ID, errCtx.Error())
				return errCtx
			}
			jp.log.Debug("job finished: %s, took: %s", job.ID, time.Since(start))

			return nil
		}, onFinish: func() {
			jp.totalParallelSteps.Add(-1)
		},
	}); isFull {
		releaseStep = true
	}
}

func (jp *JobProcessor) processStep(ctx context.Context, job *Job) error {

	err := job.jobFunc(ctx)
	if err != nil {
		jp.log.Error("error executing job '%s'.", job.ID)
		return fmt.Errorf("error executing job '%s': %w", job.ID, err)
	}

	return nil
}

type jobProcess struct {
	executor func() error
	onFinish func()
	result   error
}

type jobTypeProcessor struct {
	totalSteps atomic.Int32
	job        chan jobProcess
	config     *jobProcessorConfig
}

type jobProcessorConfig struct {
	debug         func(*jobTypeProcessor)
	maxSteps      int32
	waitTerminate func() error
}

func WithDebug(debug func(*jobTypeProcessor)) func(spc *jobProcessorConfig) {
	return func(spc *jobProcessorConfig) {
		spc.debug = debug
	}
}

func WithMaxSteps(maxSteps int) func(spc *jobProcessorConfig) {
	return func(spc *jobProcessorConfig) {
		spc.maxSteps = int32(maxSteps)
	}
}

func WithWaitTerminate(fn func() error) func(spc *jobProcessorConfig) {
	return func(spc *jobProcessorConfig) {
		spc.waitTerminate = fn
	}

}

func newStepTypeProcessor(ctx context.Context, configs ...func(config *jobProcessorConfig)) *jobTypeProcessor {

	spc := &jobProcessorConfig{
		maxSteps: 4,
	}
	for _, config := range configs {
		config(spc)
	}

	sp := jobTypeProcessor{
		job:    make(chan jobProcess, spc.maxSteps),
		config: spc,
	}
	wg := &sync.WaitGroup{}
	wg.Add(int(spc.maxSteps))
	// create go routines to read jobs
	for i := 0; i < int(spc.maxSteps); i++ {
		go sp.process(ctx, wg)
	}

	if spc.debug != nil {
		go func() {
			for {
				spc.debug(&sp)
				time.Sleep(5 * time.Second)
			}
		}()
	}
	go func() {
		wg.Wait()
		sp.close()
		_ = spc.waitTerminate()
	}()

	return &sp
}

func (stp *jobTypeProcessor) enqueue(job jobProcess) (isFull bool) {
	//if stp.totalSteps.Load() < stp.config.maxSteps {
	stp.totalSteps.Add(1)
	stp.job <- job
	//	return
	//}
	//if job.onFinish != nil {
	//	job.onFinish()
	//}
	//return true
	return
}

func (stp *jobTypeProcessor) process(ctx context.Context, group *sync.WaitGroup) {
	defer group.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-stp.job:
			func() {
				defer func() {
					if job.onFinish != nil {
						job.onFinish()
					}
					stp.totalSteps.Add(-1)
				}()
				err := job.executor()
				job.result = err
			}()
		}
	}
}

func (stp *jobTypeProcessor) close() {
	close(stp.job)
}
