package booking

import (
	"errors"
	"sync"
	"testing"
)

func TestConfirmedBookingsRespectCapacityAndCancellation(t *testing.T) {
	service := New()
	if err := service.CreateSession("workshop", 2); err != nil {
		t.Fatal(err)
	}
	item, err := service.Book("first", "workshop", " ADA@EXAMPLE.INVALID ", 2)
	if err != nil || item.Status != Confirmed || item.Email != "ada@example.invalid" {
		t.Fatalf("confirmed normalized booking: %+v, err=%v", item, err)
	}
	if _, err := service.Book("overflow", "workshop", "ada@example.invalid", 1); !errors.Is(err, ErrNoSeats) {
		t.Fatalf("oversubscribed request: %v", err)
	}
	if _, err := service.GetBooking("overflow"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed booking mutated state: %v", err)
	}
	if err := service.Cancel("first"); err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel("first"); err != nil {
		t.Fatalf("repeat cancellation must be idempotent: %v", err)
	}
	if seats, err := service.Available("workshop"); err != nil || seats != 2 {
		t.Fatalf("released capacity=%d, err=%v", seats, err)
	}
	if _, err := service.Book("first", "workshop", "ada@example.invalid", 1); !errors.Is(err, ErrBookingExists) {
		t.Fatalf("cancelled identifier reused: %v", err)
	}
	for _, id := range []string{"second", "third"} {
		if _, err := service.Book(id, "workshop", "ada@example.invalid", 1); err != nil {
			t.Fatalf("same email may book again: %v", err)
		}
	}
}

func TestInvalidRequestsAndProcessMemoryBoundary(t *testing.T) {
	service := New()
	if err := service.CreateSession("", 1); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if err := service.CreateSession("workshop", 0); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if err := service.CreateSession("workshop", MaxSeatsPerBooking+1); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateSession("workshop", 1); !errors.Is(err, ErrSessionExists) {
		t.Fatal(err)
	}
	for _, seats := range []int{0, -1, MaxSeatsPerBooking + 1} {
		if _, err := service.Book("invalid", "workshop", "ada@example.invalid", seats); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid seats %d: %v", seats, err)
		}
	}
	if _, err := service.Book("missing", "unknown", "ada@example.invalid", 1); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := service.Book("empty-email", "workshop", "   ", 1); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := service.Book("at-limit", "workshop", "ada@example.invalid", MaxSeatsPerBooking); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Available("workshop"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("state persisted into a new service: %v", err)
	}
	if err := service.Cancel("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestConcurrentBookingsCannotOversubscribe(t *testing.T) {
	service := New()
	if err := service.CreateSession("workshop", 1); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, id := range []string{"first", "second"} {
		workers.Add(1)
		go func(id string) {
			defer workers.Done()
			_, err := service.Book(id, "workshop", "ada@example.invalid", 1)
			results <- err
		}(id)
	}
	workers.Wait()
	close(results)
	confirmed := 0
	for err := range results {
		if err == nil {
			confirmed++
		} else if !errors.Is(err, ErrNoSeats) {
			t.Fatal(err)
		}
	}
	if confirmed != 1 {
		t.Fatalf("confirmed bookings=%d, want one", confirmed)
	}
}
