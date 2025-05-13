CREATE TABLE cron_jobs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cron_schedule VARCHAR(50) NOT NULL,
    task_data TEXT NOT NULL,
    active BOOLEAN DEFAULT TRUE
);