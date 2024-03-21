package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Job struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	Payload   string `json:"payload"`
	Timestamp string `json:"timestamp"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("job command expected ('create', 'list', 'claim' or 'claim-id')")
	}

	switch os.Args[1] {
	case "create":
		createCmd := flag.NewFlagSet("create", flag.ExitOnError)
		status := createCmd.String("status", "available", "Status of the job")
		payload := createCmd.String("payload", "", "Payload of the job")
		err := createCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalf("Error parsing create command: %v\n", err)
		}
		fmt.Printf("Creating job with status '%s' and payload '%s'\n", *status, *payload)
		createJob("http://localhost:8080/jobs", *status, *payload)

	case "list":
		fmt.Println("Listing available jobs")
		listJobs("http://localhost:8080/jobs")

	case "claim":
		fmt.Println("Claiming a job")
		claimJob("http://localhost:8080/jobs/claim")

	case "claim-id":
		claimIdCmd := flag.NewFlagSet("claim-id", flag.ExitOnError)
		jobID := claimIdCmd.Int("id", 0, "ID of the job to claim")
		err := claimIdCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalf("Error parsing claim-id command: %v\n", err)
		}
		fmt.Printf("Claiming a job with ID %d\n", *jobID)
		claimJobByID("http://localhost:8080", *jobID)

	default:
		log.Fatalf("Unknown command: '%s'\n", os.Args[1])
	}
}

// createJob sends a request to create a new job.
func createJob(url string, status string, payload string) {
	job := Job{
		Status:  status,
		Payload: payload,
	}

	jobJSON, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("Error marshaling job: %v\n", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jobJSON))
	if err != nil {
		log.Fatalf("Error creating request: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v\n", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err)
	}

	fmt.Println("Response:", string(body))
}

// listJobs sends a request to list all jobs.
func listJobs(url string) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error sending request to list jobs: %v\n", err)
	}
	defer response.Body.Close()

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err)
	}

	var jobs []Job
	if err := json.Unmarshal(responseData, &jobs); err != nil {
		log.Fatalf("Error decoding JSON: %v\n", err)
	}

	for _, job := range jobs {
		fmt.Printf("ID: %d, Status: %s, Payload: %s, Timestamp: %s\n", job.ID, job.Status, job.Payload, job.Timestamp)
	}
}

// claimJob sends a request to claim a job.
func claimJob(url string) {
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v\n", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request to claim a job: %v\n", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to claim a job, status code: %d\n", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err)
	}

	var job Job
	if err := json.Unmarshal(body, &job); err != nil {
		log.Fatalf("Error decoding response JSON: %v\n", err)
	}

	fmt.Printf("Claimed Job - ID: %d, Status: %s, Payload: %s\n", job.ID, job.Status, job.Payload)
}

// claimJobByID sends a request to claim a job with the given ID.
func claimJobByID(baseURL string, jobID int) {
	requestURL := fmt.Sprintf("%s/jobs/%d/claim", baseURL, jobID)

	req, err := http.NewRequest("PUT", requestURL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v\n", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request to claim job: %v\n", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err)
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully claimed job: %s\n", body)
	} else {
		log.Fatalf("Failed to claim job, status code: %d, response: %s\n", resp.StatusCode, body)
	}
}
