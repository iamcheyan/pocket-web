package main

type bookmarkInput struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	FolderID string `json:"folderId"`
}
type readingInput struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Read    bool   `json:"read"`
	Offline string `json:"offlineMetadata"`
}
