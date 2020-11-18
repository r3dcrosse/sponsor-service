package circuitbreaker

import (
	"github.com/rubyist/circuitbreaker"
)

// Initialize data
var CB *circuit.Breaker

func InitCircuitBreaker() {
	CB = circuit.NewThresholdBreaker(10)

	events := CB.Subscribe()
	go func() {
		for {
			_ = <-events
			// Can check for breaker events here
			//fmt.Printf("Breaker Event | %s", e)
		}
	}()
}
