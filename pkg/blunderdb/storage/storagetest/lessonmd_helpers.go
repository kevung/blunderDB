package storagetest

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

func domainStep(title, text string) domain.LessonStep {
	return domain.LessonStep{Title: title, Text: text}
}
