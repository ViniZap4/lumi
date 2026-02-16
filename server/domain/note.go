// server/domain/note.go
package domain

import "time"

type Note struct {
	ID        string    `json:"id" yaml:"id"`
	Title     string    `json:"title" yaml:"title"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	Tags      []string  `json:"tags" yaml:"tags"`
	Path      string    `json:"path" yaml:"-"`
	Content   string    `json:"content" yaml:"-"`
}

type Folder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
