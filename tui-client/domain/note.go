// tui-client/domain/note.go
package domain

import "time"

type Note struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
	Tags      []string  `yaml:"tags"`
	Path      string    `yaml:"-"`
	Content   string    `yaml:"-"`
}

type Folder struct {
	Name     string
	Path     string
	Parent   *Folder
	Children []*Folder
}
