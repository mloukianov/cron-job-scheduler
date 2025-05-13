## Cron Job Scheduler

### JobManager

JobManager - polls the DB, fetches the jobs, compares with current set of scheduled jobs, adds/removes/updates jobs if needed
Also maintains the mapping from job ID in the database to EntryId used by cron package

### Job

Job -  basically the same information as in the database

1. Create JobManager by calling NewJobManager and providing DB connection string and poll interval
2. Call Start() on newly created instance of the JobManager (make sure to defer Stop() on that JobManager)
3. JobManager starts polling the database and scheduling the jobs
