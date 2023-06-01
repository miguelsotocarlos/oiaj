package oiajudge

import "time"

func (s *Server) GetTime() time.Time {
	t := s.MockTime.Load()
	if t == nil {
		return time.Now()
	} else {
		return *t
	}
}

func (s *Server) SetMockTime(t time.Time) {
	s.MockTime.Store(&t)
}

func (s *Server) UnmockTime() {
	s.MockTime.Store(nil)
}
