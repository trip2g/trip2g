package federatedassetlink

func SigningKey(secret string) []byte {
	return signingKey(secret)
}

func (s *Spent) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.expiries)
}
