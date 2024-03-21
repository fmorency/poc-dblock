package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
)

var db *sql.DB

type Job struct {
	ID        int64  `json:"id,omitempty"`
	Status    string `json:"status,omitempty"`
	Payload   string `json:"payload"`
	Timestamp string `json:"timestamp,omitempty"`
}

func initDB(dataSourceName string) {
	var err error
	db, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database: %q", err)
	}
}

func main() {
	initDB("host=postgres user=user password=password dbname=jobqueue sslmode=disable")

	router := gin.Default()

	router.GET("/jobs", listJobs)
	router.POST("/jobs", createJob)
	router.PUT("/jobs/claim", claimJob)
	router.PUT("/jobs/:id/claim", claimJobByID)

	err := router.Run(":8080")
	if err != nil {
		log.Fatalf("Error starting server: %q", err)
	}
}

// listJobs returns a list of all jobs in the queue
func listJobs(c *gin.Context) {
	var jobs []Job

	rows, err := db.Query("SELECT id, status, payload, timestamp FROM job_queue ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Status, &j.Payload, &j.Timestamp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		jobs = append(jobs, j)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// createJob adds a new job to the queue
func createJob(c *gin.Context) {
	var newJob Job

	if err := c.ShouldBindJSON(&newJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assuming "status" is set client-side; otherwise, set it to a default value here
	sqlStatement := `INSERT INTO job_queue (status, payload) VALUES ($1, $2) RETURNING id, timestamp`
	err := db.QueryRow(sqlStatement, newJob.Status, newJob.Payload).Scan(&newJob.ID, &newJob.Timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, newJob)
}

// claimJob selects and claims the first available job in the queue
func claimJob(c *gin.Context) {
	var job Job

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	row := tx.QueryRow("SELECT id, payload FROM job_queue WHERE status = 'available' ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1")
	//row := tx.QueryRow("SELECT id, payload FROM job_queue WHERE status = 'available' ORDER BY id") // Multiple clients can claim the same item
	if err := row.Scan(&job.ID, &job.Payload); err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No available jobs"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim job"})
		}
		return
	}

	_, err = tx.Exec("UPDATE job_queue SET status = 'claimed' WHERE id = $1", job.ID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	job.Status = "claimed"

	c.JSON(http.StatusOK, job)
}

// claimJobByID selects and claims a specific job in the queue by its ID
func claimJobByID(c *gin.Context) {
	jobID := c.Param("id")

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	var job Job
	err = tx.QueryRow("SELECT id, payload FROM job_queue WHERE id = $1 AND status = 'available' FOR UPDATE", jobID).Scan(&job.ID, &job.Payload)
	//err = tx.QueryRow("SELECT id, payload FROM job_queue WHERE id = $1 AND status = 'available'", jobID).Scan(&job.ID, &job.Payload) // Multiple clients can claim the same item
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or not available"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim job"})
		}
		return
	}

	_, err = tx.Exec("UPDATE job_queue SET status = 'claimed' WHERE id = $1", jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	job.Status = "claimed"

	// Return the claimed job
	c.JSON(http.StatusOK, job)
}
