package domain

import "errors"

var (
	ErrNotFound      = errors.New("resource not found")
	ErrConflict      = errors.New("resource already exists or conflict state")
	ErrNoCandidates  = errors.New("no available candidates for review")
	ErrReviewerExist = errors.New("user is already a reviewer")
	ErrNotAssigned   = errors.New("user is not assigned as reviewer")
	ErrPRMerged      = errors.New("pull request is already merged")
)
