package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3" // Ensure this import is present
)

// Job represents a cron job in the database.
type Job struct {
	ID           int
	Name         string
	CronSchedule string
	TaskData     string
	Active       bool
}

// JobManager manages cron jobs.
type JobManager struct {
	db         *sql.DB
	cron       *cron.Cron
	jobs       map[int]cron.EntryID // Maps DB job ID to cron.EntryID (type from robfig/cron/v3)
	mu         sync.RWMutex         // Protects jobs map
	pollTicker *time.Ticker         // Polling ticker
}

// NewJobManager initializes a new JobManager.
func NewJobManager(dbConnString string, pollInterval time.Duration) (*JobManager, error) {
	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	cron1 := cron.New()
	return &JobManager{
		db:         db,
		cron:       cron1,
		jobs:       make(map[int]cron.EntryID), // Use cron.EntryID type
		pollTicker: time.NewTicker(pollInterval),
	}, nil
}

// Start begins the cron scheduler and polling loop.
func (jm *JobManager) Start() {
	jm.cron.Start()
	go jm.pollDatabase()
}

// Stop gracefully stops the cron scheduler and polling.
func (jm *JobManager) Stop() {
	jm.pollTicker.Stop()
	ctx := jm.cron.Stop()
	<-ctx.Done()
	jm.db.Close()
}

// pollDatabase periodically checks the database for changes.
func (jm *JobManager) pollDatabase() {
	for range jm.pollTicker.C {
		if err := jm.syncJobs(); err != nil {
			log.Printf("Error syncing jobs: %v", err)
		}
	}
}

// syncJobs synchronizes cron jobs with the database.
func (jm *JobManager) syncJobs() error {
	// Fetch active jobs from database
	rows, err := jm.db.Query("SELECT id, name, cron_schedule, task_data, active FROM cron_jobs WHERE active = TRUE")
	if err != nil {
		return fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	// Collect current jobs
	dbJobs := make(map[int]Job)
	for rows.Next() {
		var job Job
		if err := rows.Scan(&job.ID, &job.Name, &job.CronSchedule, &job.TaskData, &job.Active); err != nil {
			return fmt.Errorf("failed to scan job: %w", err)
		}
		dbJobs[job.ID] = job
	}

	jm.mu.Lock()
	defer jm.mu.Unlock()

	// Stop jobs that are no longer in the database or inactive
	for dbID, entryID := range jm.jobs {
		if _, exists := dbJobs[dbID]; !exists {
			log.Printf("Stopping job ID %d", dbID)
			jm.cron.Remove(entryID)
			delete(jm.jobs, dbID)
		}
	}

	// Start new jobs
	for _, job := range dbJobs {
		if _, exists := jm.jobs[job.ID]; !exists {
			log.Printf("Starting job ID %d: %s (%s)", job.ID, job.Name, job.CronSchedule)
			entryID, err := jm.cron.AddFunc(job.CronSchedule, jm.createJobFunc(job))
			if err != nil {
				log.Printf("Failed to add job ID %d: %v", job.ID, err)
				continue
			}
			jm.jobs[job.ID] = entryID
		}
	}

	return nil
}

// createJobFunc creates the function to run for a job.
func (jm *JobManager) createJobFunc(job Job) func() {
	return func() {
		log.Printf("Running job ID %d: %s, Data: %s", job.ID, job.Name, job.TaskData)
		// Implement your job logic here, e.g., process task_data
	}
}

func main() {
	// Database connection string (update with your credentials)
	dbConnString := "postgres://user:password@localhost:5432/dbname?sslmode=disable"
	// Poll every 10 seconds
	pollInterval := 10 * time.Second

	// Initialize job manager
	jm, err := NewJobManager(dbConnString, pollInterval)
	if err != nil {
		log.Fatalf("Failed to initialize job manager: %v", err)
	}

	// Start the job manager
	jm.Start()
	defer jm.Stop()

	// Keep the program running
	select {}
}
