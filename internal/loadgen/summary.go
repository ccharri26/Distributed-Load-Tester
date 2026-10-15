package loadgen

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Summary struct {
	ScheduledRequests      int
	StartedRequests        int
	CompletedRequests      int
	LateCompletedRequests  int
	SuccessfulRequests     int
	FailedRequests         int
	MinRequestDuration     time.Duration
	MaxRequestDuration     time.Duration
	AverageRequestDuration time.Duration
	P99                    float64
	P95                    float64
	P50                    float64
}

func (s *Summary) OutputResults() {
	resultsString :=
		fmt.Sprintf("Load Testing Results: \n\n\t"+
			"Scheduled Requests: %d \n\t"+
			"Started Requests: %d \n\t"+
			"Completed Requests: %d \n\t"+
			"Requests Completed After Deadline: %d \n\t"+
			"Successful Requests: %d \n\t"+
			"Failed Requests: %d \n\t"+
			"Fastest Request: %s \n\t"+
			"Slowest Request: %s \n\t"+
			"Average Request Time: %s \n\t"+
			"P99: %f \n\t"+
			"P95: %f \n\t"+
			"P50: %f \n`",
			s.ScheduledRequests, s.StartedRequests, s.CompletedRequests, s.LateCompletedRequests,
			s.SuccessfulRequests, s.FailedRequests, s.MinRequestDuration.String(), s.MaxRequestDuration.String(),
			s.AverageRequestDuration.String(), s.P99, s.P95, s.P50)

	err := os.WriteFile("TestResults.txt", []byte(resultsString), 0644)
	if err != nil {
		log.Fatal("Failed to write results to output file")
		return
	}
}
