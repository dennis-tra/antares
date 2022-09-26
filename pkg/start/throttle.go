package start

import "time"

type Throttle struct {
	C    <-chan time.Time // The channel on which the leases are delivered.
	done chan struct{}
}

// NewThrottle returns a new Throttle containing a channel that will send the time x number
// of times within a period specified by the duration argument. It drops leases to make up
// for slow receivers. The duration must be greater than zero; if not, NewThrottle will
// panic. Stop the throttle to release associated resources and close its channel.
func NewThrottle(x int, duration time.Duration) *Throttle {
	ch := make(chan time.Time, x-1)
	done := make(chan struct{}, 1)

	go func() {
		ticker := time.NewTicker(duration)

		now := time.Now()

		for i := 0; i < x-1; i++ {
			ch <- now
		}

	loop:
		for {
			select {
			case t := <-ticker.C:
				for i := 0; i < x; i++ {
					select {
					case ch <- t:
					default:
					}
				}
			case <-done:
				break loop
			}
		}
		ticker.Stop()
		close(ch)
	}()
	return &Throttle{C: ch, done: done}
}

// Stop turns off a throttle. After Stop, no more leases will be sent. Stop closes the
// channel.
func (t *Throttle) Stop() {
	select {
	case t.done <- struct{}{}:
	default:
	}
}
