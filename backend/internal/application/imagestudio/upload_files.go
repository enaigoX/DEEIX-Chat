package imagestudio

import (
	"context"
	"strings"

	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
)

func (s *Service) getActiveFileObjectByID(ctx context.Context, userID uint, fileID string) (*domainconversation.FileObject, error) {
	content, err := s.uploadSvc.OpenFileContent(ctx, userID, strings.TrimSpace(fileID))
	if err != nil {
		return nil, mapUploadError(err)
	}
	_ = content.Reader.Close()
	return &content.File, nil
}

func (s *Service) getActiveFileObjectsByIDs(ctx context.Context, userID uint, fileIDs []string) ([]domainconversation.FileObject, error) {
	results := make([]domainconversation.FileObject, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		file, err := s.getActiveFileObjectByID(ctx, userID, fileID)
		if err != nil {
			return nil, err
		}
		results = append(results, *file)
	}
	return results, nil
}
