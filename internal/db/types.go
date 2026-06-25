package db

type Session struct {
	ID           string
	ProjectID    string
	ParentID     string
	Directory    string
	Title        string
	Agent        string
	TimeCreated  int64
	TimeUpdated  int64
	TimeArchived int64
}

func (s Session) IsSubagent() bool {
	return s.ParentID != ""
}

func (s Session) IsArchived() bool {
	return s.TimeArchived > 0
}

type Project struct {
	ID       string
	Worktree string
	Name     string
	VCS      string
}
