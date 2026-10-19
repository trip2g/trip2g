package federatedassetlink

var SigningKey = signingKey

func (s *Spent) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.expiries)
}
