package ehr

type Storage struct {
	documents map[string][]byte
}

func New() *Storage {
	return &Storage{documents: make(map[string][]byte)}
}

func (s *Storage) Save(user string, document []byte) {
	s.documents[user] = document
}

func (s *Storage) Get(user string) []byte {
	if document, ok := s.documents[user]; ok {
		return document
	}
	return nil
}
