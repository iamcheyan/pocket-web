package main

func (s *Server) viewFor(id string) BrowserView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.views[id]
}

func (s *Server) putView(id string, view BrowserView) {
	s.mu.Lock()
	s.views[id] = view
	s.mu.Unlock()
}
