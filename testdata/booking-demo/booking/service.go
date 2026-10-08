package booking

import (
	"errors"
	"strings"
	"sync"
)

const MaxSeatsPerBooking = 4

type Status string

const (
	Confirmed Status = "confirmed"
	Cancelled Status = "cancelled"
)

var (
	ErrInvalid       = errors.New("invalid input")
	ErrNotFound      = errors.New("not found")
	ErrSessionExists = errors.New("session identifier already exists")
	ErrBookingExists = errors.New("booking identifier already exists")
	ErrNoSeats       = errors.New("insufficient available seats")
)

type Session struct {
	ID       string
	Capacity int
}

type Booking struct {
	ID        string
	SessionID string
	Email     string
	Seats     int
	Status    Status
}

type Service struct {
	mu       sync.Mutex
	sessions map[string]Session
	bookings map[string]Booking
}

func New() *Service {
	return &Service{sessions: map[string]Session{}, bookings: map[string]Booking{}}
}

func (s *Service) CreateSession(id string, capacity int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" || capacity <= 0 {
		return ErrInvalid
	}
	if _, exists := s.sessions[id]; exists {
		return ErrSessionExists
	}
	s.sessions[id] = Session{ID: id, Capacity: capacity}
	return nil
}

func (s *Service) Book(id, sessionID, email string, seats int) (Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	email = strings.ToLower(strings.TrimSpace(email))
	if id == "" || email == "" || seats <= 0 || seats > MaxSeatsPerBooking {
		return Booking{}, ErrInvalid
	}
	if _, exists := s.bookings[id]; exists {
		return Booking{}, ErrBookingExists
	}
	session, exists := s.sessions[sessionID]
	if !exists {
		return Booking{}, ErrNotFound
	}
	if session.Capacity-s.usedSeats(sessionID) < seats {
		return Booking{}, ErrNoSeats
	}
	result := Booking{ID: id, SessionID: sessionID, Email: email, Seats: seats, Status: Confirmed}
	s.bookings[id] = result
	return result, nil
}

func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, exists := s.bookings[id]
	if !exists {
		return ErrNotFound
	}
	if item.Status == Cancelled {
		return nil
	}
	item.Status = Cancelled
	s.bookings[id] = item
	return nil
}

func (s *Service) Available(sessionID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, exists := s.sessions[sessionID]
	if !exists {
		return 0, ErrNotFound
	}
	return session.Capacity - s.usedSeats(sessionID), nil
}

func (s *Service) GetBooking(id string) (Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, exists := s.bookings[id]
	if !exists {
		return Booking{}, ErrNotFound
	}
	return item, nil
}

// usedSeats is called only while holding mu.
func (s *Service) usedSeats(sessionID string) int {
	count := 0
	for _, item := range s.bookings {
		if item.SessionID == sessionID && item.Status == Confirmed {
			count += item.Seats
		}
	}
	return count
}
